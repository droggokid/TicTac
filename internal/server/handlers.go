package server

import (
	"log"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"tictac/internal/game"
	"tictac/internal/transport"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// GameServer manages active games
type GameServer struct {
	games map[string]*game.Game
	mutex sync.RWMutex
}

var gameServer = &GameServer{
	games: make(map[string]*game.Game),
}

// generateGameID creates a random 4-character game ID
func generateGameID() string {
	const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	rand.Seed(time.Now().UnixNano())
	b := make([]byte, 4)
	for i := range b {
		b[i] = chars[rand.Intn(len(chars))]
	}
	return string(b)
}

// GameHandler handles WebSocket connections for the game
func GameHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("upgrade error:", err)
		return
	}
	defer conn.Close()

	var currentGame *game.Game
	var playerSymbol game.Symbol

	// Read messages from client
	for {
		var msg transport.ClientMessage
		err := conn.ReadJSON(&msg)
		if err != nil {
			log.Println("read error:", err)
			break
		}

		switch msg.Type {
		case transport.CreateGame:
			handleCreateGame(conn, &currentGame, &playerSymbol)

		case transport.JoinGame:
			handleJoinGame(conn, msg.GameID, &currentGame,
				&playerSymbol)

		case transport.MakeMove:
			if currentGame != nil && msg.Row != nil && msg.Col != nil {
				handleMakeMove(currentGame, playerSymbol, *msg.Row,
					*msg.Col)
			}
		}
	}

	// Clean up on disconnect
	if currentGame != nil {
		log.Printf("Player %s disconnected from game %s",
			string(playerSymbol), currentGame.ID)
	}
}

func handleCreateGame(conn *websocket.Conn, currentGame **game.Game, playerSymbol *game.Symbol) {
	gameServer.mutex.Lock()
	defer gameServer.mutex.Unlock()

	gameID := generateGameID()
	newGame := game.NewGame(gameID)
	gameServer.games[gameID] = newGame

	symbol, _ := newGame.AddPlayer(conn)
	*currentGame = newGame
	*playerSymbol = symbol

	response := transport.ServerMessage{
		Type:       transport.CreateGame,
		GameID:     gameID,
		YourSymbol: string(symbol),
		Message:    "Game created. Waiting for opponent...",
	}
	conn.WriteJSON(response)
	log.Printf("Game %s created by player %s", gameID, string(symbol))
}

func handleJoinGame(conn *websocket.Conn, gameID string, currentGame **game.Game, playerSymbol *game.Symbol,
) {
	gameServer.mutex.RLock()
	existingGame, exists := gameServer.games[gameID]
	gameServer.mutex.RUnlock()

	if !exists {
		conn.WriteJSON(transport.ServerMessage{
			Type:    transport.Error,
			Message: "Game not found",
		})
		return
	}

	if existingGame.IsFull() {
		conn.WriteJSON(transport.ServerMessage{
			Type:    transport.Error,
			Message: "Game is full",
		})
		return
	}

	symbol, err := existingGame.AddPlayer(conn)
	if err != nil {
		conn.WriteJSON(transport.ServerMessage{
			Type:    transport.Error,
			Message: err.Error(),
		})
		return
	}

	*currentGame = existingGame
	*playerSymbol = symbol

	// Notify both players game is starting
	response := transport.ServerMessage{
		Type:       transport.JoinGame,
		GameID:     gameID,
		YourSymbol: string(symbol),
		Message:    "Joined game successfully",
	}
	conn.WriteJSON(response)

	// Send initial state to both players
	broadcastGameState(existingGame)
	log.Printf("Player %s joined game %s", string(symbol), gameID)
}

func handleMakeMove(g *game.Game, symbol game.Symbol, row, col int) {
	err := g.MakeMove(symbol, row, col)
	if err != nil {
		// Send error to player who made invalid move
		var playerConn *websocket.Conn
		if symbol == game.X && g.PlayerX != nil {
			playerConn = g.PlayerX.Conn
		} else if symbol == game.O && g.PlayerO != nil {
			playerConn = g.PlayerO.Conn
		}
		if playerConn != nil {
			playerConn.WriteJSON(transport.ServerMessage{
				Type:    transport.Error,
				Message: err.Error(),
			})
		}
		return
	}

	// Check for game over
	winner := g.Board.CheckWinner()
	if winner != game.Empty {
		broadcastGameOver(g, string(winner))
		return
	}

	if g.Board.IsFull() {
		broadcastGameOver(g, "draw")
		return
	}

	// Broadcast updated state
	broadcastGameState(g)
}

func broadcastGameState(g *game.Game) {
	state := transport.ServerMessage{
		Type:        transport.StateUpdate,
		Board:       g.GetBoardState(),
		CurrentTurn: string(g.Turn),
	}

	if g.PlayerX != nil {
		g.PlayerX.Conn.WriteJSON(state)
	}
	if g.PlayerO != nil {
		g.PlayerO.Conn.WriteJSON(state)
	}
}

func broadcastGameOver(g *game.Game, winner string) {
	msg := transport.ServerMessage{
		Type:   transport.GameOver,
		Winner: winner,
		Board:  g.GetBoardState(),
	}

	if winner == "draw" {
		msg.Message = "Game ended in a draw"
	} else {
		msg.Message = "Player " + winner + " wins!"
	}

	if g.PlayerX != nil {
		g.PlayerX.Conn.WriteJSON(msg)
	}
	if g.PlayerO != nil {
		g.PlayerO.Conn.WriteJSON(msg)
	}

	// Clean up game
	gameServer.mutex.Lock()
	delete(gameServer.games, g.ID)
	gameServer.mutex.Unlock()
	log.Printf("Game %s finished: %s", g.ID, msg.Message)
}
