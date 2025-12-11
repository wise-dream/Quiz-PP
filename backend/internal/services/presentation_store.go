package services

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"powerpoint-quiz/internal/models"
)

// PresentationStore manages presentations data in JSON file
type PresentationStore struct {
	mu       sync.RWMutex
	filePath string
	dataPath string
	data     *models.PresentationsFile
}

// NewPresentationStore creates a new presentation store and loads data
func NewPresentationStore(dataPath string) (*PresentationStore, error) {
	filePath := filepath.Join(dataPath, "presentations.json")
	
	store := &PresentationStore{
		filePath: filePath,
		dataPath: dataPath,
		data: &models.PresentationsFile{
			Presentations: make(map[string]*models.PresentationRecord),
		},
	}

	if err := store.Load(); err != nil {
		return nil, fmt.Errorf("failed to load presentations: %w", err)
	}

	return store, nil
}

// Load reads presentations.json file or creates empty structure if file doesn't exist
func (s *PresentationStore) Load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if file exists
	if _, err := os.Stat(s.filePath); os.IsNotExist(err) {
		log.Printf("Presentations file not found, creating empty structure: %s", s.filePath)
		return nil
	}

	// Read file
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		return fmt.Errorf("failed to read presentations file: %w", err)
	}

	// Parse JSON
	var file models.PresentationsFile
	if err := json.Unmarshal(data, &file); err != nil {
		log.Printf("Failed to parse presentations.json, using empty structure: %v", err)
		// Use empty structure instead of failing
		return nil
	}

	s.data = &file
	log.Printf("Loaded %d presentations from %s", len(s.data.Presentations), s.filePath)
	return nil
}

// save atomically writes presentations.json file (temp file → rename)
// Must be called with lock held
func (s *PresentationStore) save() error {
	// Marshal JSON with indentation
	data, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal presentations: %w", err)
	}

	// Write to temp file
	tempPath := s.filePath + ".tmp"
	if err := os.WriteFile(tempPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	// Sync to disk
	file, err := os.OpenFile(tempPath, os.O_RDWR, 0644)
	if err != nil {
		return fmt.Errorf("failed to open temp file for sync: %w", err)
	}
	if err := file.Sync(); err != nil {
		file.Close()
		return fmt.Errorf("failed to sync temp file: %w", err)
	}
	file.Close()

	// Atomic rename
	if err := os.Rename(tempPath, s.filePath); err != nil {
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	return nil
}

// Save atomically writes presentations.json file (temp file → rename)
func (s *PresentationStore) Save() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.save()
}

// GetOrCreate gets existing presentation or creates new one
func (s *PresentationStore) GetOrCreate(docKey string) *models.PresentationRecord {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.data.Presentations == nil {
		s.data.Presentations = make(map[string]*models.PresentationRecord)
	}

	if record, exists := s.data.Presentations[docKey]; exists {
		return record
	}

	// Create new record
	record := &models.PresentationRecord{
		DocKey:       docKey,
		LastRoomCode: "",
		Slides:       make(map[string]*models.SlideInfo),
	}
	s.data.Presentations[docKey] = record
	return record
}

// LinkRoom links a presentation to a room
func (s *PresentationStore) LinkRoom(docKey, roomCode string) error {
	if docKey == "" {
		return fmt.Errorf("docKey is required")
	}
	if roomCode == "" {
		return fmt.Errorf("roomCode is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Get or create record (already under lock)
	if s.data.Presentations == nil {
		s.data.Presentations = make(map[string]*models.PresentationRecord)
	}

	record, exists := s.data.Presentations[docKey]
	if !exists {
		record = &models.PresentationRecord{
			DocKey:       docKey,
			LastRoomCode: "",
			Slides:       make(map[string]*models.SlideInfo),
		}
		s.data.Presentations[docKey] = record
	}

	record.LastRoomCode = roomCode

	if err := s.save(); err != nil {
		return fmt.Errorf("failed to save after linking room: %w", err)
	}

	log.Printf("Linked presentation %s to room %s", docKey, roomCode)
	return nil
}

// FindRoomByDocKey finds room code for a presentation
func (s *PresentationStore) FindRoomByDocKey(docKey string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.data.Presentations == nil {
		return "", false
	}

	record, exists := s.data.Presentations[docKey]
	if !exists || record.LastRoomCode == "" {
		return "", false
	}

	return record.LastRoomCode, true
}

// SaveSlideImage saves PNG image to disk and returns relative path
func (s *PresentationStore) SaveSlideImage(docKey, slideId string, data []byte) (string, error) {
	// Create directory structure: presentations/{docKey}/slides/
	dirPath := filepath.Join(s.dataPath, "presentations", docKey, "slides")
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}

	// Save PNG file
	filePath := filepath.Join(dirPath, slideId+".png")
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return "", fmt.Errorf("failed to write image file: %w", err)
	}

	// Return relative path
	relativePath := filepath.Join("presentations", docKey, "slides", slideId+".png")
	return relativePath, nil
}

// UpdateSlideSnapshot decodes base64 image and saves it, updates JSON
func (s *PresentationStore) UpdateSlideSnapshot(docKey, slideId string, imageBase64 string) (string, error) {
	if docKey == "" {
		return "", fmt.Errorf("docKey is required")
	}
	if slideId == "" {
		return "", fmt.Errorf("slideId is required")
	}
	if imageBase64 == "" {
		return "", fmt.Errorf("imageBase64 is required")
	}

	// Remove data URL prefix if present
	base64Data := imageBase64
	if strings.HasPrefix(imageBase64, "data:image/png;base64,") {
		base64Data = strings.TrimPrefix(imageBase64, "data:image/png;base64,")
	}

	// Decode base64
	imageData, err := base64.StdEncoding.DecodeString(base64Data)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64: %w", err)
	}

	// Save image to disk
	imagePath, err := s.SaveSlideImage(docKey, slideId, imageData)
	if err != nil {
		return "", fmt.Errorf("failed to save image: %w", err)
	}

	// Update JSON
	s.mu.Lock()
	defer s.mu.Unlock()

	// Get or create record (already under lock)
	if s.data.Presentations == nil {
		s.data.Presentations = make(map[string]*models.PresentationRecord)
	}

	record, exists := s.data.Presentations[docKey]
	if !exists {
		record = &models.PresentationRecord{
			DocKey:       docKey,
			LastRoomCode: "",
			Slides:       make(map[string]*models.SlideInfo),
		}
		s.data.Presentations[docKey] = record
	}

	if record.Slides == nil {
		record.Slides = make(map[string]*models.SlideInfo)
	}

	slideInfo, exists := record.Slides[slideId]
	if !exists {
		slideInfo = &models.SlideInfo{}
		record.Slides[slideId] = slideInfo
	}

	slideInfo.ImagePath = imagePath
	// Don't touch config if it exists

	if err := s.save(); err != nil {
		return "", fmt.Errorf("failed to save after updating snapshot: %w", err)
	}

	log.Printf("Updated slide snapshot: docKey=%s, slideId=%s, path=%s", docKey, slideId, imagePath)
	return imagePath, nil
}

// UpdateSlideConfig updates configuration for a slide
func (s *PresentationStore) UpdateSlideConfig(docKey, slideId string, cfg *models.SlideConfig) error {
	if docKey == "" {
		return fmt.Errorf("docKey is required")
	}
	if slideId == "" {
		return fmt.Errorf("slideId is required")
	}
	if cfg == nil {
		return fmt.Errorf("config is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Get or create record (already under lock)
	if s.data.Presentations == nil {
		s.data.Presentations = make(map[string]*models.PresentationRecord)
	}

	record, exists := s.data.Presentations[docKey]
	if !exists {
		record = &models.PresentationRecord{
			DocKey:       docKey,
			LastRoomCode: "",
			Slides:       make(map[string]*models.SlideInfo),
		}
		s.data.Presentations[docKey] = record
	}

	if record.Slides == nil {
		record.Slides = make(map[string]*models.SlideInfo)
	}

	slideInfo, exists := record.Slides[slideId]
	if !exists {
		// Create minimal slide info if doesn't exist
		slideInfo = &models.SlideInfo{
			ImagePath: "",
		}
		record.Slides[slideId] = slideInfo
	}

	// Update config
	slideInfo.Config = cfg

	if err := s.save(); err != nil {
		return fmt.Errorf("failed to save after updating config: %w", err)
	}

	log.Printf("Updated slide config: docKey=%s, slideId=%s", docKey, slideId)
	return nil
}

