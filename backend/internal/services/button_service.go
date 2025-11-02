package services

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	"powerpoint-quiz/internal/models"
)

// ButtonService manages physical hardware buttons
type ButtonService struct {
	database *sql.DB
}

// NewButtonService creates a new button service
func NewButtonService(database *sql.DB) *ButtonService {
	return &ButtonService{
		database: database,
	}
}

// RegisterButton registers a new hardware button
func (bs *ButtonService) RegisterButton(macAddress, buttonID, name string) (*models.HardwareButton, error) {
	// Normalize MAC address (uppercase, remove colons/spaces)
	macAddress = normalizeMAC(macAddress)

	if buttonID == "" {
		buttonID = "1"
	}

	// Check if button already exists
	existing, err := bs.GetButtonByMAC(macAddress)
	if err == nil && existing != nil {
		log.Printf("Button already exists: MAC=%s", macAddress)
		return existing, nil
	}

	// Create new button
	buttonIDGenerated := fmt.Sprintf("btn_%s", macAddress[len(macAddress)-6:]) // Use last 6 chars of MAC as ID
	now := time.Now()

	query := `INSERT INTO hardware_buttons 
		(id, mac_address, button_id, name, is_active, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`

	_, err = bs.database.Exec(query, buttonIDGenerated, macAddress, buttonID, name, true, now, now)
	if err != nil {
		return nil, fmt.Errorf("failed to insert button: %w", err)
	}

	log.Printf("Button registered: MAC=%s, ID=%s, ButtonID=%s", macAddress, buttonIDGenerated, buttonID)

	return bs.GetButtonByMAC(macAddress)
}

// normalizeMAC normalizes MAC address format
func normalizeMAC(macAddress string) string {
	return strings.ToUpper(strings.ReplaceAll(strings.ReplaceAll(macAddress, ":", ""), " ", ""))
}

// GetButtonByMAC returns a button by its MAC address
func (bs *ButtonService) GetButtonByMAC(macAddress string) (*models.HardwareButton, error) {
	macAddress = normalizeMAC(macAddress)

	query := `SELECT id, mac_address, button_id, name, room_code, team_id, team_name, 
		is_active, press_count, last_press, created_at, updated_at
		FROM hardware_buttons WHERE mac_address = ?`

	var button models.HardwareButton
	var lastPress sql.NullTime

	err := bs.database.QueryRow(query, macAddress).Scan(
		&button.ID,
		&button.MACAddress,
		&button.ButtonID,
		&button.Name,
		&button.RoomCode,
		&button.TeamID,
		&button.TeamName,
		&button.IsActive,
		&button.PressCount,
		&lastPress,
		&button.CreatedAt,
		&button.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("button not found: %s", macAddress)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query button: %w", err)
	}

	if lastPress.Valid {
		button.LastPress = lastPress.Time
	}

	return &button, nil
}

// AssignButtonToTeam assigns a button to a team in a room
func (bs *ButtonService) AssignButtonToTeam(macAddress, roomCode, teamID, teamName string) error {
	macAddress = normalizeMAC(macAddress)

	query := `UPDATE hardware_buttons 
		SET room_code = ?, team_id = ?, team_name = ?, updated_at = ?
		WHERE mac_address = ?`

	result, err := bs.database.Exec(query, roomCode, teamID, teamName, time.Now(), macAddress)
	if err != nil {
		return fmt.Errorf("failed to update button: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("button not found: %s", macAddress)
	}

	log.Printf("Button %s assigned to team %s (%s) in room %s", macAddress, teamName, teamID, roomCode)
	return nil
}

// UnassignButton removes button assignment from team
func (bs *ButtonService) UnassignButton(macAddress string) error {
	macAddress = normalizeMAC(macAddress)

	query := `UPDATE hardware_buttons 
		SET room_code = '', team_id = '', team_name = '', updated_at = ?
		WHERE mac_address = ?`

	result, err := bs.database.Exec(query, time.Now(), macAddress)
	if err != nil {
		return fmt.Errorf("failed to update button: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("button not found: %s", macAddress)
	}

	log.Printf("Button %s unassigned", macAddress)
	return nil
}

// RecordButtonPress records a button press and returns the button info
func (bs *ButtonService) RecordButtonPress(macAddress string) (*models.HardwareButton, error) {
	macAddress = normalizeMAC(macAddress)

	// First, check if button exists and is active
	button, err := bs.GetButtonByMAC(macAddress)
	if err != nil {
		return nil, err
	}

	if !button.IsActive {
		return nil, fmt.Errorf("button is not active: %s", macAddress)
	}

	// Update press count and last press time
	now := time.Now()
	query := `UPDATE hardware_buttons 
		SET press_count = press_count + 1, last_press = ?, updated_at = ?
		WHERE mac_address = ?`

	_, err = bs.database.Exec(query, now, now, macAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to update button press: %w", err)
	}

	// Return updated button
	button.PressCount++
	button.LastPress = now
	button.UpdatedAt = now

	log.Printf("Button press recorded: MAC=%s, Total presses=%d", macAddress, button.PressCount)

	return button, nil
}

// GetButtonsByRoom returns all buttons assigned to a room
func (bs *ButtonService) GetButtonsByRoom(roomCode string) ([]*models.HardwareButton, error) {
	query := `SELECT id, mac_address, button_id, name, room_code, team_id, team_name, 
		is_active, press_count, last_press, created_at, updated_at
		FROM hardware_buttons WHERE room_code = ? ORDER BY created_at DESC`

	rows, err := bs.database.Query(query, roomCode)
	if err != nil {
		return nil, fmt.Errorf("failed to query buttons: %w", err)
	}
	defer rows.Close()

	var buttons []*models.HardwareButton
	for rows.Next() {
		var button models.HardwareButton
		var lastPress sql.NullTime

		err := rows.Scan(
			&button.ID,
			&button.MACAddress,
			&button.ButtonID,
			&button.Name,
			&button.RoomCode,
			&button.TeamID,
			&button.TeamName,
			&button.IsActive,
			&button.PressCount,
			&lastPress,
			&button.CreatedAt,
			&button.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan button: %w", err)
		}

		if lastPress.Valid {
			button.LastPress = lastPress.Time
		}

		buttons = append(buttons, &button)
	}

	return buttons, rows.Err()
}

// GetAllButtons returns all registered buttons
func (bs *ButtonService) GetAllButtons() ([]*models.HardwareButton, error) {
	query := `SELECT id, mac_address, button_id, name, room_code, team_id, team_name, 
		is_active, press_count, last_press, created_at, updated_at
		FROM hardware_buttons ORDER BY created_at DESC`

	rows, err := bs.database.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query buttons: %w", err)
	}
	defer rows.Close()

	var buttons []*models.HardwareButton
	for rows.Next() {
		var button models.HardwareButton
		var lastPress sql.NullTime

		err := rows.Scan(
			&button.ID,
			&button.MACAddress,
			&button.ButtonID,
			&button.Name,
			&button.RoomCode,
			&button.TeamID,
			&button.TeamName,
			&button.IsActive,
			&button.PressCount,
			&lastPress,
			&button.CreatedAt,
			&button.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan button: %w", err)
		}

		if lastPress.Valid {
			button.LastPress = lastPress.Time
		}

		buttons = append(buttons, &button)
	}

	return buttons, rows.Err()
}

// DeleteButton removes a button from the system
func (bs *ButtonService) DeleteButton(macAddress string) error {
	macAddress = normalizeMAC(macAddress)

	query := `DELETE FROM hardware_buttons WHERE mac_address = ?`
	result, err := bs.database.Exec(query, macAddress)
	if err != nil {
		return fmt.Errorf("failed to delete button: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("button not found: %s", macAddress)
	}

	log.Printf("Button deleted: %s", macAddress)
	return nil
}

