package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"powerpoint-quiz/internal/models"
	"powerpoint-quiz/internal/services"

	"github.com/gorilla/mux"
)

// ButtonHandler handles HTTP requests for hardware buttons
type ButtonHandler struct {
	wsService    *services.WebSocketService
	buttonService *services.ButtonService
}

// NewButtonHandler creates a new button handler
func NewButtonHandler(wsService *services.WebSocketService, buttonService *services.ButtonService) *ButtonHandler {
	return &ButtonHandler{
		wsService:     wsService,
		buttonService: buttonService,
	}
}

// ButtonPressRequest represents a button press request
type ButtonPressRequest struct {
	MACAddress string `json:"macAddress"`
	ButtonID   string `json:"buttonId,omitempty"` // Optional: button number from device firmware
}

// ButtonPressResponse represents the response to a button press
type ButtonPressResponse struct {
	Success   bool   `json:"success"`
	Message   string `json:"message"`
	Processed bool   `json:"processed"` // Whether the press was processed (question was active)
}

// RegisterButtonRequest represents a button registration request
type RegisterButtonRequest struct {
	MACAddress string `json:"macAddress"`
	ButtonID   string `json:"buttonId"`
	Name       string `json:"name,omitempty"`
}

// AssignButtonRequest represents a button assignment request
type AssignButtonRequest struct {
	MACAddress string `json:"macAddress"`
	RoomCode   string `json:"roomCode"`
	TeamID     string `json:"teamId"`
}

// PressButton handles button press events from physical buttons
// POST /api/button/press
func (bh *ButtonHandler) PressButton(w http.ResponseWriter, r *http.Request) {
	var req ButtonPressRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.MACAddress == "" {
		http.Error(w, "MAC address is required", http.StatusBadRequest)
		return
	}

	// Try to record the button press
	button, err := bh.buttonService.RecordButtonPress(req.MACAddress)
	if err != nil {
		// If button doesn't exist, auto-register it
		log.Printf("Button not found, attempting auto-registration: MAC=%s, Error=%v", req.MACAddress, err)
		buttonID := req.ButtonID
		if buttonID == "" {
			buttonID = "1"
		}
		
		registeredButton, registerErr := bh.buttonService.RegisterButton(req.MACAddress, buttonID, "")
		if registerErr != nil {
			log.Printf("Auto-registration failed: %v", registerErr)
			response := ButtonPressResponse{
				Success:   false,
				Message:   fmt.Sprintf("Button not found and auto-registration failed: %v", registerErr),
				Processed: false,
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(response)
			return
		}
		
		log.Printf("Button auto-registered: MAC=%s", req.MACAddress)
		button = registeredButton
		
		// Now try to record the press again (this will work since button exists now)
		button, err = bh.buttonService.RecordButtonPress(req.MACAddress)
		if err != nil {
			log.Printf("Button press error after registration: %v", err)
			response := ButtonPressResponse{
				Success:   false,
				Message:   err.Error(),
				Processed: false,
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(response)
			return
		}
	}

	// Check if button is assigned to a team
	if button.RoomCode == "" || button.TeamID == "" {
		response := ButtonPressResponse{
			Success:   true,
			Message:   "Button not assigned to a team",
			Processed: false,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	// Process the button press through WebSocket service
	err = bh.wsService.HandleButtonPress(button.RoomCode, button.TeamID)
	if err != nil {
		log.Printf("Button press processing error: %v", err)
		response := ButtonPressResponse{
			Success:   true,
			Message:   err.Error(),
			Processed: false,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	// Success - button press was processed
	response := ButtonPressResponse{
		Success:   true,
		Message:   "Button press processed successfully",
		Processed: true,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// RegisterButton registers a new hardware button
// POST /api/button/register
func (bh *ButtonHandler) RegisterButton(w http.ResponseWriter, r *http.Request) {
	var req RegisterButtonRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.MACAddress == "" {
		http.Error(w, "MAC address is required", http.StatusBadRequest)
		return
	}

	if req.ButtonID == "" {
		req.ButtonID = "1" // Default button ID if not provided
	}

	button, err := bh.buttonService.RegisterButton(req.MACAddress, req.ButtonID, req.Name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(button)
}

// AssignButton assigns a button to a team in a room
// POST/PUT /api/button/assign
func (bh *ButtonHandler) AssignButton(w http.ResponseWriter, r *http.Request) {
	var req AssignButtonRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.MACAddress == "" || req.RoomCode == "" || req.TeamID == "" {
		http.Error(w, "MAC address, room code, and team ID are required", http.StatusBadRequest)
		return
	}

	// Get room to validate team exists
	room := bh.wsService.GetRoom(req.RoomCode)
	if room == nil {
		http.Error(w, "Room not found", http.StatusNotFound)
		return
	}

	room.Mu.RLock()
	team, exists := room.Teams[req.TeamID]
	room.Mu.RUnlock()

	if !exists {
		http.Error(w, "Team not found in room", http.StatusNotFound)
		return
	}

	err := bh.buttonService.AssignButtonToTeam(req.MACAddress, req.RoomCode, req.TeamID, team.Name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	button, err := bh.buttonService.GetButtonByMAC(req.MACAddress)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(button)
}

// UnassignButton removes button assignment from team
// POST /api/button/unassign
func (bh *ButtonHandler) UnassignButton(w http.ResponseWriter, r *http.Request) {
	var req struct {
		MACAddress string `json:"macAddress"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.MACAddress == "" {
		http.Error(w, "MAC address is required", http.StatusBadRequest)
		return
	}

	err := bh.buttonService.UnassignButton(req.MACAddress)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ListButtons returns all registered buttons
// GET /api/button/list
func (bh *ButtonHandler) ListButtons(w http.ResponseWriter, r *http.Request) {
	buttons, err := bh.buttonService.GetAllButtons()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	// Always return an array, even if empty
	if buttons == nil {
		buttons = []*models.HardwareButton{}
	}
	json.NewEncoder(w).Encode(buttons)
}

// GetButtonsByRoom returns all buttons assigned to a room
// GET /api/button/room/{roomCode}
func (bh *ButtonHandler) GetButtonsByRoom(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	roomCode := vars["roomCode"]

	if roomCode == "" {
		http.Error(w, "Room code is required", http.StatusBadRequest)
		return
	}

	buttons, err := bh.buttonService.GetButtonsByRoom(roomCode)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	// Always return an array, even if empty
	if buttons == nil {
		buttons = []*models.HardwareButton{}
	}
	json.NewEncoder(w).Encode(buttons)
}

// GetButton returns a specific button by MAC address
// GET /api/button/{macAddress}
func (bh *ButtonHandler) GetButton(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	macAddress := vars["macAddress"]

	if macAddress == "" {
		http.Error(w, "MAC address is required", http.StatusBadRequest)
		return
	}

	button, err := bh.buttonService.GetButtonByMAC(macAddress)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(button)
}

// DeleteButton removes a button from the system
// DELETE /api/button/{macAddress}
func (bh *ButtonHandler) DeleteButton(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	macAddress := vars["macAddress"]

	if macAddress == "" {
		http.Error(w, "MAC address is required", http.StatusBadRequest)
		return
	}

	err := bh.buttonService.DeleteButton(macAddress)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

