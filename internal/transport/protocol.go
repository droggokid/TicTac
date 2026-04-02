package transport

// MessageType sent between client and server
type MessageType string

const (
	CreateGame  MessageType = "create_game"
	JoinGame    MessageType = "join_game"
	MakeMove    MessageType = "make_move"
	StateUpdate MessageType = "state_update"
	GameOver    MessageType = "game_over"
	Error       MessageType = "error"
)

// ClientMessage - messages from client to server
type ClientMessage struct {
	Type   MessageType `json:"type"`
	GameID string      `json:"gameId,omitempty"`
	Row    *int        `json:"row,omitempty"`
	Col    *int        `json:"col,omitempty"`
}

// ServerMessage - messages from server to client
type ServerMessage struct {
	Type        MessageType `json:"type"`
	GameID      string      `json:"gameId,omitempty"`
	Board       [][]string  `json:"board,omitempty"`
	CurrentTurn string      `json:"currentTurn,omitempty"`
	Winner      string      `json:"winner,omitempty"`
	Message     string      `json:"message,omitempty"`
	YourSymbol  string      `json:"yourSymbol,omitempty"`
}
