package tui

import (
	"encoding/json"
	"fmt"
	"sync"

	"tictac/internal/transport"

	"github.com/awesome-gocui/gocui"
	"github.com/gorilla/websocket"
)

var (
	wsConn     *websocket.Conn
	wsMu       sync.Mutex
	serverAddr string
)

// wsConnect dials the game server.
func wsConnect() error {
	url := "ws://" + serverAddr + "/game"
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		return fmt.Errorf("cannot connect to %s: %w", url, err)
	}
	wsMu.Lock()
	wsConn = conn
	wsMu.Unlock()
	return nil
}

// wsDisconnect closes the connection. Safe to call multiple times.
func wsDisconnect() {
	wsMu.Lock()
	defer wsMu.Unlock()
	if wsConn != nil {
		wsConn.Close()
		wsConn = nil
	}
}

// wsSend sends a message to the server.
func wsSend(msg transport.ClientMessage) error {
	wsMu.Lock()
	conn := wsConn
	wsMu.Unlock()
	if conn == nil {
		return fmt.Errorf("not connected to server")
	}
	return conn.WriteJSON(msg)
}

// wsListen reads server messages in its own goroutine. All state mutations are
// applied inside g.Update so they are synchronised with the gocui render loop.
func wsListen(g *gocui.Gui) {
	defer wsDisconnect()
	for {
		wsMu.Lock()
		conn := wsConn
		wsMu.Unlock()
		if conn == nil {
			return
		}
		_, raw, err := conn.ReadMessage()
		if err != nil {
			return // connection closed or network error
		}
		var msg transport.ServerMessage
		if err := json.Unmarshal(raw, &msg); err != nil {
			continue
		}
		applyServerMessage(g, msg)
	}
}

// applyServerMessage dispatches a server message and updates TUI state.
// Everything inside g.Update runs in the gocui main loop — no separate locking
// is needed for the shared state variables defined in app.go.
func applyServerMessage(g *gocui.Gui, msg transport.ServerMessage) {
	g.Update(func(g *gocui.Gui) error {
		switch msg.Type {

		case transport.CreateGame:
			// Server confirmed creation and assigned the real game ID.
			currentGameID = msg.GameID
			yourSymbol = msg.YourSymbol

		case transport.JoinGame:
			// Server confirmed we joined. A state_update arrives immediately after.
			currentGameID = msg.GameID
			yourSymbol = msg.YourSymbol

		case transport.StateUpdate:
			// Log the move that was made by diffing the board.
			old := boardState
			applyBoardUpdate(msg.Board, msg.CurrentTurn)
			if r, c, sym := boardDiff(old, boardState); r >= 0 {
				moveLog = append(moveLog, fmt.Sprintf("%s played (%d, %d)", sym, r, c))
			}
			// First state_update signals the game has started — leave waiting scene.
			if currentScene == sceneCreateGame || currentScene == sceneJoinGame {
				g.DeleteView("create_game")
				g.DeleteView("join_title")
				g.DeleteView("game_id_input")
				currentScene = sceneGame
			}

		case transport.GameOver:
			applyBoardUpdate(msg.Board, "")
			gameIsOver = true
			gameResult = msg.Winner
			if msg.Winner != "draw" {
				if _, line := checkWinner(); line != nil {
					winningCells = line
					winLine = winLineFromCells(line)
				}
			}
			if msg.Message != "" {
				moveLog = append(moveLog, msg.Message)
			}
			// Could arrive while still in a waiting scene if the game ends instantly.
			if currentScene == sceneCreateGame || currentScene == sceneJoinGame {
				g.DeleteView("create_game")
				g.DeleteView("join_title")
				g.DeleteView("game_id_input")
				currentScene = sceneGame
			}

		case transport.Error:
			moveLog = append(moveLog, "Server error: "+msg.Message)
		}
		return nil
	})
}

// applyBoardUpdate copies the server board into boardState and updates the turn.
func applyBoardUpdate(board [][]string, turn string) {
	if len(board) == 3 {
		for r := 0; r < 3; r++ {
			if len(board[r]) == 3 {
				for c := 0; c < 3; c++ {
					boardState[r][c] = board[r][c]
				}
			}
		}
	}
	if turn != "" {
		currentTurn = turn
	}
}

// boardDiff returns the single cell that changed between two board snapshots.
// Returns (-1, -1, "") if no difference is found.
func boardDiff(old, new [3][3]string) (int, int, string) {
	for r := 0; r < 3; r++ {
		for c := 0; c < 3; c++ {
			if old[r][c] == "" && new[r][c] != "" {
				return r, c, new[r][c]
			}
		}
	}
	return -1, -1, ""
}

// connectAndCreate runs in a goroutine: connects to the server and creates a game.
func connectAndCreate(g *gocui.Gui) {
	if err := wsConnect(); err != nil {
		g.Update(func(g *gocui.Gui) error {
			gameError = "Connection failed: " + err.Error()
			g.DeleteView("create_game")
			currentScene = sceneLobby
			return nil
		})
		return
	}
	go wsListen(g)
	if err := wsSend(transport.ClientMessage{Type: transport.CreateGame}); err != nil {
		g.Update(func(g *gocui.Gui) error {
			gameError = "Failed to create game: " + err.Error()
			g.DeleteView("create_game")
			currentScene = sceneLobby
			return nil
		})
	}
}

// connectAndJoin runs in a goroutine: connects to the server and joins a game.
func connectAndJoin(g *gocui.Gui, gameID string) {
	if err := wsConnect(); err != nil {
		g.Update(func(g *gocui.Gui) error {
			gameError = "Connection failed: " + err.Error()
			joiningGame = false
			return nil
		})
		return
	}
	go wsListen(g)
	if err := wsSend(transport.ClientMessage{
		Type:   transport.JoinGame,
		GameID: gameID,
	}); err != nil {
		g.Update(func(g *gocui.Gui) error {
			gameError = "Failed to join game: " + err.Error()
			joiningGame = false
			return nil
		})
	}
}

// wsSendMove sends the current cursor position as a move to the server.
func wsSendMove() {
	if gameIsOver || currentTurn != yourSymbol {
		return
	}
	r, c := boardCursorRow, boardCursorCol
	if boardState[r][c] != "" {
		return // cell already occupied
	}
	row, col := r, c
	if err := wsSend(transport.ClientMessage{
		Type: transport.MakeMove,
		Row:  &row,
		Col:  &col,
	}); err != nil {
		moveLog = append(moveLog, "Error sending move: "+err.Error())
	}
}
