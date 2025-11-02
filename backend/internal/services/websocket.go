package services

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"powerpoint-quiz/internal/models"

	"github.com/gorilla/websocket"
)

// WebSocketService handles WebSocket connections and events
type WebSocketService struct {
	hub           *models.Hub
	buttonService *ButtonService
}

// SetButtonService sets the button service for hardware button support
func (ws *WebSocketService) SetButtonService(bs *ButtonService) {
	ws.buttonService = bs
}

// NewWebSocketService creates a new WebSocket service
func NewWebSocketService() *WebSocketService {
	return &WebSocketService{
		hub: &models.Hub{
			Rooms:      make(map[string]*models.Room),
			Clients:    make(map[*models.Client]bool),
			Register:   make(chan *models.Client),
			Unregister: make(chan *models.Client),
			Broadcast:  make(chan []byte),
		},
	}
}

// GetHub returns the hub instance
func (ws *WebSocketService) GetHub() *models.Hub {
	return ws.hub
}

// generateRoomCode generates a random 4-character room code
func generateRoomCode() string {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	code := make([]byte, 4)
	for i := range code {
		num, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		code[i] = charset[num.Int64()]
	}
	return string(code)
}

// generateAdminPassword generates a random admin password
func generateAdminPassword() string {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	password := make([]byte, 6)
	for i := range password {
		num, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		password[i] = charset[num.Int64()]
	}
	return string(password)
}

// Run starts the hub's main loop
func (ws *WebSocketService) Run() {
	for {
		select {
		case client := <-ws.hub.Register:
			ws.hub.Clients[client] = true
			log.Printf("Client connected: %s", client.UserID)

		case client := <-ws.hub.Unregister:
			if _, ok := ws.hub.Clients[client]; ok {
				delete(ws.hub.Clients, client)
				close(client.Send)
				log.Printf("Client disconnected: %s", client.UserID)
			}

		case message := <-ws.hub.Broadcast:
			for client := range ws.hub.Clients {
				select {
				case client.Send <- message:
				default:
					close(client.Send)
					delete(ws.hub.Clients, client)
				}
			}
		}
	}
}

// HandleEvent processes incoming WebSocket events
func (ws *WebSocketService) HandleEvent(client *models.Client, event models.Event) {
	ws.hub.Mu.Lock()
	defer ws.hub.Mu.Unlock()

	roomID := event.QuizID
	if roomID == "" {
		// For admin_auth events, use RoomCode instead of QuizID
		if event.Type == models.EventAdminAuth {
			roomID = event.RoomCode
		} else {
			roomID = client.RoomID // Use client's room if not specified
		}
	}

	// Get or create room
	room, exists := ws.hub.Rooms[roomID]
	
	// If not found by ID, try to find by Code (rooms are stored by Code)
	if !exists && roomID != "" {
		for _, r := range ws.hub.Rooms {
			if r.Code == roomID {
				room = r
				exists = true
				log.Printf("Room found by Code: %s (ID: %s)", r.Code, r.ID)
				break
			}
		}
	}
	
	if !exists && event.Type != models.EventCreateRoom {
		// For non-create-room events, check if room exists
		if roomID != "" {
			log.Printf("Room not found for event %s: roomID=%s, client.RoomID=%s", event.Type, roomID, client.RoomID)
		}
		room = nil
	} else if !exists {
		// Only create room for create_room events
		room = &models.Room{
			ID:        roomID,
			Code:      roomID, // Set Code same as ID for new rooms
			Phase:     models.PhaseLobby,
			Players:   make(map[string]*models.Player),
			Teams:     make(map[string]*models.Team),
			CreatedAt: time.Now(),
		}
		ws.hub.Rooms[roomID] = room
	}

	switch event.Type {
	case models.EventCreateRoom:
		ws.handleCreateRoom(client, event)

	case models.EventAdminAuth:
		if room != nil {
			ws.handleAdminAuth(client, room, event)
		} else {
			ws.sendErrorToClient(client, "Room not found")
		}

	case models.EventJoinTeam:
		if room != nil {
			room.Mu.Lock()
			ws.handleJoinTeam(client, room, event)
			room.Mu.Unlock()
		}

	case models.EventCreateTeam:
		if room != nil {
			room.Mu.Lock()
			ws.handleCreateTeam(client, room, event)
			room.Mu.Unlock()
		} else {
			log.Printf("Cannot create team: room not found. QuizID=%s, RoomID=%s", event.QuizID, client.RoomID)
			ws.sendErrorToClient(client, fmt.Sprintf("Room not found: %s", event.QuizID))
		}

	case models.EventJoin:
		ws.handleJoin(client, room, event)

	case models.EventClick:
		if room != nil {
			room.Mu.Lock()
			ws.handleClick(client, room, event)
			room.Mu.Unlock()
		}

	case models.EventHostSetState:
		if room != nil {
			room.Mu.Lock()
			ws.handleHostSetState(client, room, event)
			room.Mu.Unlock()
		}

	case models.EventStartQuestion:
		if room != nil {
			room.Mu.Lock()
			ws.handleStartQuestion(client, room, event)
			room.Mu.Unlock()
		}

	case models.EventAnswerReceived:
		if room != nil {
			room.Mu.Lock()
			ws.handleAnswerReceived(client, room, event)
			room.Mu.Unlock()
		}

	case models.EventAnswerConfirmation:
		if room != nil {
			room.Mu.Lock()
			ws.handleAnswerConfirmation(client, room, event)
			room.Mu.Unlock()
		}

	case models.EventShowAnswer:
		if room != nil {
			room.Mu.Lock()
			ws.handleShowAnswer(client, room, event)
			room.Mu.Unlock()
		}

	case models.EventNextQuestion:
		if room != nil {
			room.Mu.Lock()
			ws.handleNextQuestion(client, room, event)
			room.Mu.Unlock()
		}
	}
}

// handleJoin processes player join events
func (ws *WebSocketService) handleJoin(client *models.Client, room *models.Room, event models.Event) {
	// Check if room exists
	if room == nil {
		log.Printf("Room not found for join request: %s", event.QuizID)
		errorEvent := models.Event{
			Type:    models.EventJoinError,
			Message: "Room not found",
		}
		ws.sendEventToClient(client, errorEvent)
		return
	}

	room.Mu.Lock()
	defer room.Mu.Unlock()

	player := &models.Player{
		ID:        event.UserID,
		UserID:    event.UserID,
		ButtonID:  event.ButtonID,
		Name:      event.Nickname,
		Connected: true,
	}
	room.Players[event.UserID] = player
	client.UserID = event.UserID
	client.RoomID = room.Code
	log.Printf("Player %s joined room %s", event.UserID, room.Code)

	// Send success response with specific event type
	successEvent := models.Event{
		Type: models.EventJoinSuccess,
		Data: room,
	}
	ws.sendEventToClient(client, successEvent)

	// Broadcast player joined event to all clients in the room
	playerJoinedEvent := models.Event{
		Type:   models.EventPlayerJoined,
		UserID: event.UserID,
		Data:   player,
	}
	ws.broadcastToRoom(room, playerJoinedEvent)

	// Broadcast to all clients in the room (this will send state event)
	ws.broadcastRoomState(room)
}

// handleClick processes player click events
func (ws *WebSocketService) handleClick(client *models.Client, room *models.Room, event models.Event) {
	player, exists := room.Players[event.UserID]
	if !exists {
		// Auto-create player if not exists
		player = &models.Player{
			ID:        event.UserID,
			UserID:    event.UserID,
			ButtonID:  event.ButtonID,
			Name:      fmt.Sprintf("Player %s", event.UserID),
			Connected: true,
		}
		room.Players[event.UserID] = player
	}

	clickTime := time.Now()
	player.LastClick = clickTime
	player.ClickCount++

	// Check for false start - allow clicks in both started and active phases
	if (room.Phase != models.PhaseStarted && room.Phase != models.PhaseActive) || clickTime.Before(room.EnableAt) {
		player.FalseStarts++
		log.Printf("False start by player %s (phase: %s)", event.UserID, room.Phase)
	} else if room.Phase == models.PhaseActive && room.QuestionActive {
		// This is a valid answer - set first answerer and deactivate question
		if room.FirstAnswerer == "" {
			room.FirstAnswerer = event.UserID
			room.QuestionActive = false // Deactivate question after first answer
			log.Printf("First answer received from player %s", event.UserID)
		}
	}

	log.Printf("Player %s clicked (total: %d, false starts: %d)",
		event.UserID, player.ClickCount, player.FalseStarts)
	ws.broadcastRoomState(room)
}

// handleHostSetState processes host state change events
func (ws *WebSocketService) handleHostSetState(client *models.Client, room *models.Room, event models.Event) {
	// Only allow admin/host role to change state
	if client.Role != "admin" && client.Role != "host" {
		log.Printf("Non-admin client attempted to change state, role: %s", client.Role)
		return
	}

	if event.Phase == models.PhaseStarted {
		// Transition to started phase - players can see button but it's not active yet
		room.Phase = models.PhaseStarted
		room.EnableAt = time.Now().Add(time.Duration(event.DelayMs) * time.Millisecond)

		// Send phase changed event
		phaseChangedEvent := models.Event{
			Type:  models.EventPhaseChanged,
			Phase: event.Phase,
		}
		ws.broadcastToRoom(room, phaseChangedEvent)

		// Auto-transition to active after delay (if delay is specified)
		if event.DelayMs > 0 {
			go func() {
				time.Sleep(time.Duration(event.DelayMs) * time.Millisecond)
				room.Mu.Lock()
				room.Phase = models.PhaseActive
				room.QuestionActive = true // Enable question answering
				room.Mu.Unlock()

				// Send phase changed event for active phase
				phaseChangedEvent := models.Event{
					Type:  models.EventPhaseChanged,
					Phase: models.PhaseActive,
				}
				ws.broadcastToRoom(room, phaseChangedEvent)
				ws.broadcastRoomState(room)
			}()
		}
	} else if event.Phase == models.PhaseActive {
		// Direct transition to active phase - players can now click
		room.Phase = models.PhaseActive
		room.EnableAt = time.Now()
		room.QuestionActive = true // Enable question answering
		// Clear previous answer state when transitioning to active
		room.FirstAnswerer = ""

		// Send phase changed event
		phaseChangedEvent := models.Event{
			Type:  models.EventPhaseChanged,
			Phase: event.Phase,
		}
		ws.broadcastToRoom(room, phaseChangedEvent)
	} else if event.Phase == models.PhaseStarted && room.Phase == models.PhaseActive {
		// Transition from active to started - clear question state
		room.Phase = models.PhaseStarted
		room.QuestionActive = false
		room.FirstAnswerer = ""

		// Send phase changed event
		phaseChangedEvent := models.Event{
			Type:  models.EventPhaseChanged,
			Phase: event.Phase,
		}
		ws.broadcastToRoom(room, phaseChangedEvent)
	} else {
		room.Phase = event.Phase
		// Reset question state for non-active phases
		if event.Phase != models.PhaseActive {
			room.QuestionActive = false
		}

		// Send phase changed event
		phaseChangedEvent := models.Event{
			Type:  models.EventPhaseChanged,
			Phase: event.Phase,
		}
		ws.broadcastToRoom(room, phaseChangedEvent)
	}

	log.Printf("Room %s phase changed to %s by admin", room.ID, event.Phase)
	ws.broadcastRoomState(room)
}

// broadcastRoomState sends room state to all clients in the room
func (ws *WebSocketService) broadcastRoomState(room *models.Room) {
	stateEvent := models.Event{
		Type: models.EventState,
		Data: room,
	}

	message, err := json.Marshal(stateEvent)
	if err != nil {
		log.Printf("Error marshaling state: %v", err)
		return
	}

	// Broadcast to all clients in the room
	for client := range ws.hub.Clients {
		if client.RoomID == room.Code {
			select {
			case client.Send <- message:
			default:
				close(client.Send)
				delete(ws.hub.Clients, client)
			}
		}
	}
}

// broadcastToRoom sends an event to all clients in a specific room
func (ws *WebSocketService) broadcastToRoom(room *models.Room, event models.Event) {
	message, err := json.Marshal(event)
	if err != nil {
		log.Printf("Error marshaling event: %v", err)
		return
	}

	// Broadcast to all clients in the room
	for client := range ws.hub.Clients {
		if client.RoomID == room.Code {
			select {
			case client.Send <- message:
			default:
				close(client.Send)
				delete(ws.hub.Clients, client)
			}
		}
	}
}

// handleCreateRoom processes room creation events
func (ws *WebSocketService) handleCreateRoom(client *models.Client, event models.Event) {
	roomCode := generateRoomCode()
	adminPassword := generateAdminPassword()

	room := &models.Room{
		ID:            fmt.Sprintf("room_%d", time.Now().Unix()),
		Code:          roomCode,
		Phase:         models.PhaseLobby,
		Players:       make(map[string]*models.Player),
		Teams:         make(map[string]*models.Team),
		CreatedAt:     time.Now(),
		AdminPassword: adminPassword,
	}

	ws.hub.Rooms[roomCode] = room
	client.RoomID = roomCode
	client.Role = "admin"

	log.Printf("Room created: %s, Admin password: %s", roomCode, adminPassword)

	// Send room creation response with specific event type
	response := models.Event{
		Type:       models.EventRoomCreated,
		Data:       room,
		AdminToken: adminPassword,
	}

	ws.sendEventToClient(client, response)

	// Also broadcast to all clients in the room (this will send state event)
	ws.broadcastRoomState(room)
}

// handleAdminAuth processes admin authentication
func (ws *WebSocketService) handleAdminAuth(client *models.Client, room *models.Room, event models.Event) {
	if room.AdminPassword != event.Password {
		ws.sendErrorToClient(client, "Invalid admin password")
		return
	}

	client.RoomID = event.RoomCode
	client.Role = "admin"

	log.Printf("Admin authenticated for room: %s", event.RoomCode)
	ws.broadcastRoomState(room)
}

// handleJoinTeam processes team join events
func (ws *WebSocketService) handleJoinTeam(client *models.Client, room *models.Room, event models.Event) {
	player, exists := room.Players[event.UserID]
	if !exists {
		log.Printf("Player %s not found in room, cannot join team", event.UserID)
		ws.sendErrorToClient(client, "Player not found in room")
		return
	}

	// Update player name if provided
	if event.Nickname != "" {
		player.Name = event.Nickname
	}

	// Add player to team
	if team, exists := room.Teams[event.TeamID]; exists {
		// Remove player from other teams first
		for _, t := range room.Teams {
			for i, p := range t.Players {
				if p == event.UserID {
					t.Players = append(t.Players[:i], t.Players[i+1:]...)
					break
				}
			}
		}

		// Add to new team
		team.Players = append(team.Players, event.UserID)
		log.Printf("Player %s joined team %s", player.Name, team.Name)

		// Send team joined event to all clients in the room
		teamJoinedEvent := models.Event{
			Type:     models.EventTeamJoined,
			UserID:   event.UserID,
			TeamID:   event.TeamID,
			TeamName: team.Name,
			Data:     player,
		}
		ws.broadcastToRoom(room, teamJoinedEvent)
	} else {
		log.Printf("Team %s not found", event.TeamID)
		ws.sendErrorToClient(client, "Team not found")
		return
	}

	ws.broadcastRoomState(room)
}

// handleCreateTeam processes team creation events
func (ws *WebSocketService) handleCreateTeam(client *models.Client, room *models.Room, event models.Event) {
	log.Printf("handleCreateTeam: client.Role=%s, room.Code=%s, event.TeamName=%s", client.Role, room.Code, event.TeamName)
	
	if client.Role != "admin" && client.Role != "host" {
		log.Printf("Access denied: client role is '%s', need 'admin' or 'host'", client.Role)
		ws.sendErrorToClient(client, fmt.Sprintf("Only admin can create teams (current role: %s)", client.Role))
		return
	}

	teamID := fmt.Sprintf("team_%d", time.Now().Unix())
	team := &models.Team{
		ID:        teamID,
		Name:      event.TeamName,
		Color:     event.TeamColor,
		Players:   []string{},
		Score:     0,
		CreatedAt: time.Now(),
	}

	room.Teams[teamID] = team
	log.Printf("Team created: %s (%s)", event.TeamName, teamID)

	// Send team created event to all clients in the room
	teamCreatedEvent := models.Event{
		Type:      models.EventTeamCreated,
		TeamID:    teamID,
		TeamName:  event.TeamName,
		TeamColor: event.TeamColor,
		Data:      team,
	}
	ws.broadcastToRoom(room, teamCreatedEvent)

	// Also broadcast room state
	ws.broadcastRoomState(room)
}

// sendEventToClient sends an event to a specific client
func (ws *WebSocketService) sendEventToClient(client *models.Client, event models.Event) {
	message, err := json.Marshal(event)
	if err != nil {
		log.Printf("Error marshaling event: %v", err)
		return
	}

	select {
	case client.Send <- message:
	default:
		close(client.Send)
		delete(ws.hub.Clients, client)
	}
}

// sendErrorToClient sends an error message to a specific client
func (ws *WebSocketService) sendErrorToClient(client *models.Client, message string) {
	errorEvent := models.Event{
		Type:    models.EventError,
		Message: message,
	}
	ws.sendEventToClient(client, errorEvent)
}

func parseAllowedOrigins() []string {
	raw := os.Getenv("WS_ALLOWED_ORIGINS")
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, strings.TrimRight(p, "/"))
		}
	}
	return out
}

var allowedOrigins = parseAllowedOrigins()

func checkSameHost(origin string, host string) bool {
	u, err := url.Parse(origin)
	if err != nil {
		return false
	}
	originHost := u.Hostname()
	return (u.Scheme == "https" || u.Scheme == "http") && strings.EqualFold(originHost, host)
}

func originAllowed(origin, host string) bool {
	if origin == "" {
		return true
	}

	for _, allowed := range allowedOrigins {
		if strings.EqualFold(strings.TrimRight(origin, "/"), allowed) {
			return true
		}
	}

	if checkSameHost(origin, host) {
		return true
	}

	return false
}

// GetUpgrader returns a configured WebSocket upgrader
func GetUpgrader() websocket.Upgrader {
	return websocket.Upgrader{
		ReadBufferSize:  4096,
		WriteBufferSize: 4096,
		CheckOrigin: func(r *http.Request) bool {
			origin := r.Header.Get("Origin")
			ok := originAllowed(origin, r.Host)
			if !ok {
				log.Printf("[WS] CheckOrigin reject: origin=%q host=%q allowed=%v ua=%q",
					origin, r.Host, allowedOrigins, r.UserAgent())
			} else {
				log.Printf("[WS] CheckOrigin ok: origin=%q host=%q", origin, r.Host)
			}
			return ok
		},
	}
}

// updateRoomActivity updates the last activity timestamp for a room
func (ws *WebSocketService) updateRoomActivity(room *models.Room) {
	if room != nil {
		room.Mu.Lock()
		room.LastActivity = time.Now()
		room.Mu.Unlock()
		log.Printf("Updated activity for room %s", room.Code)
	}
}

// cleanupInactiveRooms removes rooms that haven't been active for more than 1 hour
func (ws *WebSocketService) cleanupInactiveRooms() {
	ws.hub.Mu.Lock()
	defer ws.hub.Mu.Unlock()

	cutoffTime := time.Now().Add(-1 * time.Hour)
	var roomsToDelete []string

	for roomCode, room := range ws.hub.Rooms {
		room.Mu.RLock()
		lastActivity := room.LastActivity
		room.Mu.RUnlock()

		if lastActivity.Before(cutoffTime) {
			roomsToDelete = append(roomsToDelete, roomCode)
			log.Printf("Room %s marked for deletion (last activity: %v)", roomCode, lastActivity)
		}
	}

	// Delete inactive rooms
	for _, roomCode := range roomsToDelete {
		delete(ws.hub.Rooms, roomCode)
		log.Printf("Deleted inactive room: %s", roomCode)
	}

	if len(roomsToDelete) > 0 {
		log.Printf("Cleaned up %d inactive rooms", len(roomsToDelete))
	}
}

// handleStartQuestion processes question start events
func (ws *WebSocketService) handleStartQuestion(client *models.Client, room *models.Room, event models.Event) {
	// Only allow admin to start questions
	if client.Role != "admin" && client.Role != "host" {
		log.Printf("Non-admin client attempted to start question, role: %s", client.Role)
		return
	}

	// Start the question (no need to set correct answer - it will be handled in PowerPoint)
	room.QuestionActive = true
	room.FirstAnswerer = ""
	room.QuestionStartTime = time.Now()

	log.Printf("Question started in room %s", room.Code)

	// Broadcast question start to all clients
	questionStartEvent := models.Event{
		Type: models.EventStartQuestion,
	}
	ws.broadcastToRoom(room, questionStartEvent)

	ws.broadcastRoomState(room)
}

// handleAnswerReceived processes answer events from players
func (ws *WebSocketService) handleAnswerReceived(client *models.Client, room *models.Room, event models.Event) {
	// Check if question is active
	if !room.QuestionActive {
		log.Printf("Question not active, ignoring answer from %s", event.UserID)
		return
	}

	// Check if someone already answered
	if room.FirstAnswerer != "" {
		log.Printf("Someone already answered, ignoring answer from %s", event.UserID)
		return
	}

	// Set first answerer
	room.FirstAnswerer = event.UserID
	room.QuestionActive = false // Stop accepting more answers

	log.Printf("First answer received from %s", event.UserID)

	// Broadcast answer received event
	answerEvent := models.Event{
		Type:   models.EventAnswerReceived,
		UserID: event.UserID,
		Answer: event.Answer,
	}
	ws.broadcastToRoom(room, answerEvent)

	ws.broadcastRoomState(room)
}

// handleAnswerConfirmation processes answer confirmation events
func (ws *WebSocketService) handleAnswerConfirmation(client *models.Client, room *models.Room, event models.Event) {
	// Check if user is admin/host
	if client.Role != "admin" && client.Role != "host" {
		log.Printf("Non-admin client attempted to confirm answer, role: %s", client.Role)
		return
	}

	// Check if there's a first answerer
	if room.FirstAnswerer == "" {
		log.Printf("No first answerer to confirm")
		return
	}

	// Get player info
	player, exists := room.Players[room.FirstAnswerer]
	if !exists {
		log.Printf("First answerer player not found: %s", room.FirstAnswerer)
		return
	}

	// Get player's team
	var playerTeam *models.Team
	for _, team := range room.Teams {
		for _, playerID := range team.Players {
			if playerID == room.FirstAnswerer {
				playerTeam = team
				break
			}
		}
		if playerTeam != nil {
			break
		}
	}

	// Award points if correct
	if event.IsCorrect && playerTeam != nil {
		points := event.Points
		if points <= 0 {
			points = 10 // Default points
		}
		playerTeam.Score += points
		log.Printf("Awarded %d points to team %s for correct answer", points, playerTeam.Name)
	}

	// Reset question state
	room.QuestionActive = false
	room.FirstAnswerer = ""

	log.Printf("Answer confirmed: correct=%v, points=%d", event.IsCorrect, event.Points)

	// Broadcast confirmation event
	confirmationEvent := models.Event{
		Type:          models.EventAnswerConfirmation,
		IsCorrect:     event.IsCorrect,
		Points:        event.Points,
		PlayerName:    player.Name,
		CorrectAnswer: event.CorrectAnswer,
	}
	ws.broadcastToRoom(room, confirmationEvent)

	ws.broadcastRoomState(room)
}

// handleShowAnswer processes show answer events
func (ws *WebSocketService) handleShowAnswer(client *models.Client, room *models.Room, event models.Event) {
	// Only allow admin to show answers
	if client.Role != "admin" && client.Role != "host" {
		log.Printf("Non-admin client attempted to show answer, role: %s", client.Role)
		return
	}

	log.Printf("Showing answer in room %s", room.Code)

	// Broadcast show answer event
	showAnswerEvent := models.Event{
		Type: models.EventShowAnswer,
	}
	ws.broadcastToRoom(room, showAnswerEvent)

	ws.broadcastRoomState(room)
}

// handleNextQuestion processes next question events
func (ws *WebSocketService) handleNextQuestion(client *models.Client, room *models.Room, event models.Event) {
	// Only allow admin to go to next question
	if client.Role != "admin" && client.Role != "host" {
		log.Printf("Non-admin client attempted to go to next question, role: %s", client.Role)
		return
	}

	// Reset question state
	room.QuestionActive = false
	room.FirstAnswerer = ""
	room.CorrectAnswer = ""
	room.QuestionStartTime = time.Time{}

	log.Printf("Next question in room %s", room.Code)

	// Broadcast next question event
	nextQuestionEvent := models.Event{
		Type: models.EventNextQuestion,
	}
	ws.broadcastToRoom(room, nextQuestionEvent)

	ws.broadcastRoomState(room)
}

// GetRoom returns a room by its code
func (ws *WebSocketService) GetRoom(roomCode string) *models.Room {
	ws.hub.Mu.RLock()
	defer ws.hub.Mu.RUnlock()
	return ws.hub.Rooms[roomCode]
}

// BroadcastToRoom sends an event to all clients in a specific room
func (ws *WebSocketService) BroadcastToRoom(room *models.Room, event models.Event) {
	ws.broadcastToRoom(room, event)
}

// StartRoomCleanup starts a background goroutine to clean up inactive rooms every 30 minutes
func (ws *WebSocketService) StartRoomCleanup() {
	go func() {
		ticker := time.NewTicker(30 * time.Minute)
		defer ticker.Stop()

		log.Println("Room cleanup service started (runs every 30 minutes)")

		for {
			select {
			case <-ticker.C:
				log.Println("Running room cleanup...")
				ws.cleanupInactiveRooms()
			}
		}
	}()
}

// HandleButtonPress processes a button press from a hardware button
// This method can be called from REST API handlers
func (ws *WebSocketService) HandleButtonPress(roomCode, teamID string) error {
	ws.hub.Mu.RLock()
	room, exists := ws.hub.Rooms[roomCode]
	ws.hub.Mu.RUnlock()

	if !exists {
		return fmt.Errorf("room not found: %s", roomCode)
	}

	room.Mu.Lock()
	defer room.Mu.Unlock()

	// Check if room is in active phase - button can only be pressed in active phase
	if room.Phase != models.PhaseActive {
		return fmt.Errorf("room %s is not in active phase (current phase: %s)", roomCode, room.Phase)
	}

	// Check if question is active
	if !room.QuestionActive {
		return fmt.Errorf("question not active in room %s", roomCode)
	}

	// Check if someone already answered
	if room.FirstAnswerer != "" {
		return fmt.Errorf("someone already answered in room %s", roomCode)
	}

	// Find team to get team name
	team, exists := room.Teams[teamID]
	if !exists {
		return fmt.Errorf("team not found: %s", teamID)
	}

	// Set first answerer - use team ID as identifier for hardware buttons
	room.FirstAnswerer = fmt.Sprintf("team_%s", teamID)
	room.QuestionActive = false // Stop accepting more answers
	
	// Change phase from active to started after button press
	room.Phase = models.PhaseStarted

	log.Printf("First answer received from hardware button - Team: %s (%s) in room %s. Phase changed to started.", team.Name, teamID, roomCode)

	// Broadcast phase changed event
	phaseChangedEvent := models.Event{
		Type:  models.EventPhaseChanged,
		Phase: models.PhaseStarted,
	}
	ws.broadcastToRoom(room, phaseChangedEvent)

	// Broadcast answer received event
	answerEvent := models.Event{
		Type:     models.EventAnswerReceived,
		UserID:   room.FirstAnswerer,
		TeamID:   teamID,
		TeamName: team.Name,
		Answer:   "BUTTON_PRESS",
	}
	ws.broadcastToRoom(room, answerEvent)

	ws.broadcastRoomState(room)

	return nil
}
