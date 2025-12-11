package models

// SlideConfig represents configuration for a slide question
type SlideConfig struct {
	TimeLimitSeconds int `json:"timeLimitSeconds"`
	PointsCorrect    int `json:"pointsCorrect"`
	PointsWrong      int `json:"pointsWrong"`
}

// SlideInfo represents information about a slide
type SlideInfo struct {
	ImagePath string       `json:"imagePath"`
	Config    *SlideConfig `json:"config,omitempty"`
}

// PresentationRecord represents a presentation linked to a room
type PresentationRecord struct {
	DocKey       string                `json:"docKey"`
	LastRoomCode string                `json:"lastRoomCode"`
	Slides       map[string]*SlideInfo `json:"slides"`
}

// PresentationsFile represents the root structure of presentations.json
type PresentationsFile struct {
	Presentations map[string]*PresentationRecord `json:"presentations"`
}

