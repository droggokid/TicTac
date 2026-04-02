package game

import (
	"errors"
	"sync"

	"github.com/gorilla/websocket"
)

type Player struct {
	Conn   *websocket.Conn
	Symbol Symbol
}

type Game struct {
	ID      string
	Board   *Board
	PlayerX *Player
	PlayerO *Player
	Turn    Symbol
	mutex   sync.RWMutex
}

func NewGame(id string) *Game {
	return &Game{
		ID:    id,
		Board: NewBoard(),
		Turn:  X, // X always starts
	}
}

func (g *Game) AddPlayer(conn *websocket.Conn) (Symbol, error) {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	if g.PlayerX == nil {
		g.PlayerX = &Player{Conn: conn, Symbol: X}
		return X, nil
	}
	if g.PlayerO == nil {
		g.PlayerO = &Player{Conn: conn, Symbol: O}
		return O, nil
	}
	return Empty, errors.New("game is full")
}

func (g *Game) MakeMove(symbol Symbol, row, col int) error {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	if g.Turn != symbol {
		return errors.New("not your turn")
	}

	pos := row*3 + col
	if err := g.Board.Set(pos, symbol); err != nil {
		return err
	}

	if g.Turn == X {
		g.Turn = O
	} else {
		g.Turn = X
	}

	return nil
}

// IsFull returns true if both players have joined
func (g *Game) IsFull() bool {
	g.mutex.RLock()
	defer g.mutex.RUnlock()
	return g.PlayerX != nil && g.PlayerO != nil
}

// GetBoardState returns board as 2D string array for JSON
func (g *Game) GetBoardState() [][]string {
	g.mutex.RLock()
	defer g.mutex.RUnlock()

	result := make([][]string, 3)
	for i := 0; i < 3; i++ {
		result[i] = make([]string, 3)
		for j := 0; j < 3; j++ {
			pos := i*3 + j
			symbol := g.Board.Get(pos)
			if symbol == Empty {
				result[i][j] = ""
			} else {
				result[i][j] = string(symbol)
			}
		}
	}
	return result
}
