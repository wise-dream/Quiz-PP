package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"powerpoint-quiz/internal/models"
	"powerpoint-quiz/internal/services"
)

// PresentationHandler handles HTTP requests for presentations
type PresentationHandler struct {
	store *services.PresentationStore
}

// NewPresentationHandler creates a new presentation handler
func NewPresentationHandler(store *services.PresentationStore) *PresentationHandler {
	return &PresentationHandler{
		store: store,
	}
}

// LinkPresentationRequest represents a request to link presentation to room
type LinkPresentationRequest struct {
	DocKey   string `json:"docKey"`
	RoomCode string `json:"roomCode"`
}

// LinkPresentationResponse represents the response
type LinkPresentationResponse struct {
	Success bool `json:"success"`
}

// LinkPresentation links a presentation to a room
// POST /quiz/api/presentation/link
func (h *PresentationHandler) LinkPresentation(w http.ResponseWriter, r *http.Request) {
	var req LinkPresentationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.DocKey == "" {
		http.Error(w, "docKey is required", http.StatusBadRequest)
		return
	}
	if req.RoomCode == "" {
		http.Error(w, "roomCode is required", http.StatusBadRequest)
		return
	}

	if err := h.store.LinkRoom(req.DocKey, req.RoomCode); err != nil {
		log.Printf("Failed to link presentation: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := LinkPresentationResponse{
		Success: true,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetRoomResponse represents the response for getting room by docKey
type GetRoomResponse struct {
	Success  bool   `json:"success"`
	RoomCode string `json:"roomCode,omitempty"`
}

// GetRoomByPresentation gets room code for a presentation
// GET /quiz/api/presentation/room?docKey=...
func (h *PresentationHandler) GetRoomByPresentation(w http.ResponseWriter, r *http.Request) {
	docKey := r.URL.Query().Get("docKey")
	if docKey == "" {
		http.Error(w, "docKey query parameter is required", http.StatusBadRequest)
		return
	}

	roomCode, found := h.store.FindRoomByDocKey(docKey)
	if !found {
		response := GetRoomResponse{
			Success: false,
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := GetRoomResponse{
		Success:  true,
		RoomCode: roomCode,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// SlideSnapshotRequest represents a request to save slide snapshot
type SlideSnapshotRequest struct {
	DocKey      string `json:"docKey"`
	SlideID     string `json:"slideId"`
	ImageBase64 string `json:"imageBase64"`
}

// SlideSnapshotResponse represents the response
type SlideSnapshotResponse struct {
	Success   bool   `json:"success"`
	ImagePath string `json:"imagePath,omitempty"`
}

// SaveSlideSnapshot saves a slide snapshot image
// POST /quiz/api/presentation/slide-snapshot
func (h *PresentationHandler) SaveSlideSnapshot(w http.ResponseWriter, r *http.Request) {
	var req SlideSnapshotRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.DocKey == "" {
		http.Error(w, "docKey is required", http.StatusBadRequest)
		return
	}
	if req.SlideID == "" {
		http.Error(w, "slideId is required", http.StatusBadRequest)
		return
	}
	if req.ImageBase64 == "" {
		http.Error(w, "imageBase64 is required", http.StatusBadRequest)
		return
	}

	imagePath, err := h.store.UpdateSlideSnapshot(req.DocKey, req.SlideID, req.ImageBase64)
	if err != nil {
		log.Printf("Failed to save slide snapshot: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := SlideSnapshotResponse{
		Success:   true,
		ImagePath: imagePath,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// SlideConfigRequest represents a request to save slide config
type SlideConfigRequest struct {
	DocKey  string           `json:"docKey"`
	SlideID string           `json:"slideId"`
	Config  *models.SlideConfig `json:"config"`
}

// SlideConfigResponse represents the response
type SlideConfigResponse struct {
	Success bool `json:"success"`
}

// SaveSlideConfig saves slide configuration
// POST /quiz/api/presentation/slide-config
func (h *PresentationHandler) SaveSlideConfig(w http.ResponseWriter, r *http.Request) {
	var req SlideConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.DocKey == "" {
		http.Error(w, "docKey is required", http.StatusBadRequest)
		return
	}
	if req.SlideID == "" {
		http.Error(w, "slideId is required", http.StatusBadRequest)
		return
	}
	if req.Config == nil {
		http.Error(w, "config is required", http.StatusBadRequest)
		return
	}

	if err := h.store.UpdateSlideConfig(req.DocKey, req.SlideID, req.Config); err != nil {
		log.Printf("Failed to save slide config: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := SlideConfigResponse{
		Success: true,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

