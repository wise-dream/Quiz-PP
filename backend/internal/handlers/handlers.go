package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"powerpoint-quiz/internal/models"
	"powerpoint-quiz/internal/services"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

// ActivateQuestionRequest represents a request to activate a question
type ActivateQuestionRequest struct {
	RoomCode string `json:"roomCode"`
	Duration int    `json:"duration"` // Duration in seconds, 0 means no timer
}

// ActivateQuestionResponse represents the response
type ActivateQuestionResponse struct {
	Success  bool   `json:"success"`
	Message  string `json:"message"`
	RoomCode string `json:"roomCode,omitempty"`
}

// WebSocketHandler handles WebSocket connections
type WebSocketHandler struct {
	wsService *services.WebSocketService
	upgrader  websocket.Upgrader
}

// NewWebSocketHandler creates a new WebSocket handler
func NewWebSocketHandler(wsService *services.WebSocketService) *WebSocketHandler {
	return &WebSocketHandler{
		wsService: wsService,
		upgrader:  services.GetUpgrader(),
	}
}

// ServeWS handles WebSocket upgrade requests
func (h *WebSocketHandler) ServeWS(w http.ResponseWriter, r *http.Request) {
	log.Printf("WebSocket connection attempt from %s", r.RemoteAddr)

	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	log.Printf("WebSocket connection established from %s", r.RemoteAddr)

	// Parse query parameters
	roomID := r.URL.Query().Get("room")
	if roomID == "" {
		roomID = "default"
	}
	role := r.URL.Query().Get("role")
	if role == "" {
		role = "viewer"
	}

	client := &models.Client{
		Conn:   conn,
		Send:   make(chan []byte, 256),
		RoomID: roomID,
		Role:   role,
	}

	h.wsService.GetHub().Register <- client

	go h.writePump(client)
	go h.readPump(client)
}

// readPump handles reading messages from WebSocket connection
func (h *WebSocketHandler) readPump(client *models.Client) {
	defer func() {
		h.wsService.GetHub().Unregister <- client
		client.Conn.Close()
	}()

	client.Conn.SetReadLimit(512)
	client.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	client.Conn.SetPongHandler(func(string) error {
		client.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := client.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		log.Printf("Received message from %s: %s", client.Conn.RemoteAddr(), string(message))

		var event models.Event
		if err := json.Unmarshal(message, &event); err != nil {
			log.Printf("Error unmarshaling event: %v", err)
			continue
		}

		log.Printf("Handling event: %+v", event)
		h.wsService.HandleEvent(client, event)
	}
}

// writePump handles writing messages to WebSocket connection
func (h *WebSocketHandler) writePump(client *models.Client) {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		client.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-client.Send:
			client.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				client.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := client.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			n := len(client.Send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-client.Send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			client.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := client.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// StaticHandler handles static file serving
type StaticHandler struct{}

// NewStaticHandler creates a new static file handler
func NewStaticHandler() *StaticHandler {
	return &StaticHandler{}
}

// ServeStatic serves static files
func (h *StaticHandler) ServeStatic(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path[1:] // Remove leading slash

	// Serve add-in files
	if strings.HasPrefix(path, "content-addin/") ||
		strings.HasPrefix(path, "taskpane-addin/") ||
		strings.HasPrefix(path, "shared/") {
		http.ServeFile(w, r, "addin/"+path)
		return
	}

	// Serve web files
	http.ServeFile(w, r, "web/"+path)
}

// SetupRoutes configures all HTTP routes
func SetupRoutes(wsHandler *WebSocketHandler, staticHandler *StaticHandler, buttonHandler *ButtonHandler, presentationHandler *PresentationHandler) *mux.Router {
	r := mux.NewRouter()

	// CORS middleware
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			// Set CORS headers
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

			// Handle preflight requests
			if req.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, req)
		})
	})

	// WebSocket endpoint
	r.HandleFunc("/ws", wsHandler.ServeWS)

	// PowerPoint integration endpoints
	r.HandleFunc("/api/activate-question", wsHandler.ActivateQuestion).Methods("POST")
	r.HandleFunc("/api/deactivate-question", wsHandler.DeactivateQuestion).Methods("POST")

	// Hardware button API endpoints
	r.HandleFunc("/api/button/press", buttonHandler.PressButton).Methods("POST")
	r.HandleFunc("/api/button/register", buttonHandler.RegisterButton).Methods("POST")
	r.HandleFunc("/api/button/assign", buttonHandler.AssignButton).Methods("POST", "PUT")
	r.HandleFunc("/api/button/unassign", buttonHandler.UnassignButton).Methods("POST")
	r.HandleFunc("/api/button/list", buttonHandler.ListButtons).Methods("GET")
	r.HandleFunc("/api/button/room/{roomCode}", buttonHandler.GetButtonsByRoom).Methods("GET")
	r.HandleFunc("/api/button/{macAddress}", buttonHandler.GetButton).Methods("GET")
	r.HandleFunc("/api/button/{macAddress}", buttonHandler.DeleteButton).Methods("DELETE")

	// Presentation API endpoints
	r.HandleFunc("/quiz/api/presentation/link", presentationHandler.LinkPresentation).Methods("POST")
	r.HandleFunc("/quiz/api/presentation/room", presentationHandler.GetRoomByPresentation).Methods("GET")
	r.HandleFunc("/quiz/api/presentation/slide-snapshot", presentationHandler.SaveSlideSnapshot).Methods("POST")
	r.HandleFunc("/quiz/api/presentation/slide-config", presentationHandler.SaveSlideConfig).Methods("POST")

	// Health check endpoint
	r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Static files
	r.PathPrefix("/").HandlerFunc(staticHandler.ServeStatic)

	return r
}

// ActivateQuestion activates the question button for a specific room
func (h *WebSocketHandler) ActivateQuestion(w http.ResponseWriter, r *http.Request) {
	var req ActivateQuestionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.RoomCode == "" {
		http.Error(w, "Room code is required", http.StatusBadRequest)
		return
	}

	// Find the room
	room := h.wsService.GetRoom(req.RoomCode)
	if room == nil {
		http.Error(w, "Room not found", http.StatusNotFound)
		return
	}

	// Activate the question
	room.Mu.Lock()
	room.QuestionActive = true
	room.FirstAnswerer = ""
	room.QuestionStartTime = time.Now()
	room.Mu.Unlock()

	log.Printf("Question activated for room %s via PowerPoint API", req.RoomCode)

	// Broadcast to all clients in the room
	questionStartEvent := models.Event{
		Type: models.EventStartQuestion,
	}
	h.wsService.BroadcastToRoom(room, questionStartEvent)

	// If duration is specified, set up auto-deactivation
	if req.Duration > 0 {
		go func() {
			time.Sleep(time.Duration(req.Duration) * time.Second)
			h.DeactivateQuestionInternal(req.RoomCode)
		}()
	}

	// Send response
	response := ActivateQuestionResponse{
		Success:  true,
		Message:  "Question activated successfully",
		RoomCode: req.RoomCode,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// DeactivateQuestion deactivates the question button for a specific room
func (h *WebSocketHandler) DeactivateQuestion(w http.ResponseWriter, r *http.Request) {
	var req ActivateQuestionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.RoomCode == "" {
		http.Error(w, "Room code is required", http.StatusBadRequest)
		return
	}

	success := h.DeactivateQuestionInternal(req.RoomCode)
	if !success {
		http.Error(w, "Room not found", http.StatusNotFound)
		return
	}

	// Send response
	response := ActivateQuestionResponse{
		Success:  true,
		Message:  "Question deactivated successfully",
		RoomCode: req.RoomCode,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// DeactivateQuestionInternal is a helper method to deactivate question
func (h *WebSocketHandler) DeactivateQuestionInternal(roomCode string) bool {
	room := h.wsService.GetRoom(roomCode)
	if room == nil {
		return false
	}

	room.Mu.Lock()
	room.QuestionActive = false
	room.Mu.Unlock()

	log.Printf("Question deactivated for room %s", roomCode)

	// Broadcast to all clients in the room
	nextQuestionEvent := models.Event{
		Type: models.EventNextQuestion,
	}
	h.wsService.BroadcastToRoom(room, nextQuestionEvent)

	return true
}
