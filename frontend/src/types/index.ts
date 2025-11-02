// Types for PowerPoint Quiz Frontend

export type Phase = 'lobby' | 'started' | 'active' | 'finished';

export type EventType = 
  | 'join' 
  | 'click' 
  | 'host_set_state' 
  | 'state' 
  | 'leave' 
  | 'error'
  | 'create_room'
  | 'join_team'
  | 'create_team'
  | 'admin_auth'
  | 'room_created'
  | 'join_success'
  | 'join_error'
  | 'admin_reconnect'
  | 'admin_reconnect_success'
  | 'admin_reconnect_error'
  | 'team_created'
  | 'player_joined'
  | 'player_left'
  | 'team_joined'
  | 'phase_changed'
  | 'start_question'
  | 'answer_received'
  | 'show_answer'
  | 'next_question';

export interface Player {
  id: string;
  userId: string;
  buttonId?: string;
  name: string;
  clickCount: number;
  falseStarts: number;
  lastClick: string;
  connected: boolean;
}

export interface Team {
  id: string;
  name: string;
  color: string;
  players: string[]; // UserIDs
  score: number;
  createdAt: string;
}

export interface Room {
  id: string;
  code: string;
  phase: Phase;
  players: Record<string, Player>;
  teams: Record<string, Team>;
  enableAt: string;
  createdAt: string;
  // Quiz management fields
  questionActive: boolean;
  firstAnswerer: string;
  correctAnswer: string;
  questionStartTime: string;
  // Timer fields
  questionDuration?: number; // Duration in seconds
  timeRemaining?: number;   // Time remaining in seconds
}

export interface Event {
  type: EventType;
  quizId?: string;
  userId?: string;
  buttonId?: string;
  phase?: Phase;
  delayMs?: number;
  tsClient?: number;
  optionId?: string;
  message?: string;
  data?: any;
  // New fields for team management
  roomCode?: string;
  nickname?: string;
  teamId?: string;
  teamName?: string;
  teamColor?: string;
  password?: string;
  adminToken?: string;
  adminName?: string;
  adminEmail?: string;
  // Quiz management fields
  answer?: string;
  correctAnswer?: string;
  isCorrect?: boolean;
  questionActive?: boolean;
}

export interface WebSocketMessage {
  type: 'message' | 'error' | 'close';
  data?: any;
  error?: string;
}

export interface User {
  id: string;
  nickname: string;
  role: 'admin' | 'participant';
  roomCode?: string;
  teamId?: string;
}

export interface QuizState {
  room: Room | null;
  user: User | null;
  isConnected: boolean;
  isAdmin: boolean;
  error: string | null;
}

export interface HardwareButton {
  id: string;
  macAddress: string;
  buttonId: string;
  name: string;
  // Note: roomCode, teamId, teamName are assignment data, not part of button itself
  // These should ideally be in a separate assignment table, but kept here for compatibility
  roomCode?: string;  // Optional - only set when assigned to a room
  teamId?: string;    // Optional - only set when assigned to a team
  teamName?: string;  // Optional - only set when assigned to a team
  isActive: boolean;
  lastPress?: string; // Optional - only set after first press
  pressCount: number;
  createdAt: string;
  updatedAt: string;
}
