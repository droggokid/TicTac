package game

import "errors"

// Symbol represents the player's mark on the board
type Symbol rune

const (
	Empty Symbol = ' '
	X     Symbol = 'X'
	O     Symbol = 'O'
)

// Board represents a 3x3 Tic Tac Toe Board
type Board struct {
	cells [9]Symbol // 0-8
}

// NewBoard creates a new empty board
func NewBoard() *Board {
	b := &Board{}
	for i := range b.cells {
		b.cells[i] = Empty
	}
	return b
}

func (b *Board) Get(pos int) Symbol {
	if pos < 0 || pos > 8 {
		return Empty
	}
	return b.cells[pos]
}

func (b *Board) Set(pos int, symbol Symbol) error {
	if pos < 0 || pos > 8 {
		return errors.New("position out of bounds")
	}
	if b.cells[pos] != Empty {
		return errors.New("position already occupied")
	}
	b.cells[pos] = symbol
	return nil
}

// CheckWinner returns the winning symbol, or Empty if no winner
func (b *Board) CheckWinner() Symbol {
	// Win patterns: rows, columns, diagonals
	winPatterns := [][]int{
		{0, 1, 2}, {3, 4, 5}, {6, 7, 8}, // rows
		{0, 3, 6}, {1, 4, 7}, {2, 5, 8}, // columns
		{0, 4, 8}, {2, 4, 6}, // diagonals
	}

	for _, pattern := range winPatterns {
		if b.cells[pattern[0]] != Empty &&
			b.cells[pattern[0]] == b.cells[pattern[1]] &&
			b.cells[pattern[1]] == b.cells[pattern[2]] {
			return b.cells[pattern[0]]
		}
	}
	return Empty
}

func (b *Board) IsFull() bool {
	for _, cell := range b.cells {
		if cell == Empty {
			return false
		}
	}
	return true
}

// IsGameOver returns true if there's a winner or draw
func (b *Board) IsGameOver() bool {
	return b.CheckWinner() != Empty || b.IsFull()
}

// GetBoard returns the board
func (b *Board) GetBoard() [9]Symbol {
	return b.cells
}

// WinningLine returns the three cell positions (0-8) that form the winning
// line, or nil if there is no winner yet. Useful for highlighting in the TUI
// and for including in the game_over broadcast.
func (b *Board) WinningLine() []int {
	winPatterns := [][]int{
		{0, 1, 2}, {3, 4, 5}, {6, 7, 8},
		{0, 3, 6}, {1, 4, 7}, {2, 5, 8},
		{0, 4, 8}, {2, 4, 6},
	}
	for _, pattern := range winPatterns {
		if b.cells[pattern[0]] != Empty &&
			b.cells[pattern[0]] == b.cells[pattern[1]] &&
			b.cells[pattern[1]] == b.cells[pattern[2]] {
			return pattern
		}
	}
	return nil
}
