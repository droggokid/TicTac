package tui

import (
	"fmt"
	"log"
	"strings"

	"github.com/awesome-gocui/gocui"
)

var asciiArt = ` __    __    ___  _        __   ___   ___ ___    ___      ______   ___       ______  ____   __ ______
|  |__|  |  /  _]| |      /  ] /   \ |   |   |  /  _]    |      | /   \     |      ||    | /  ]      | /    |  /  ]
|  |  |  | /  [_ | |     /  / |     || _   _ | /  [_     |      ||     |    |      | |  | /  /|      ||  o  | /  /
|  |  |  ||    _]| |___ /  /  |  O  ||  \_/  ||    _]    |_|  |_||  O  |    |_|  |_| |  |/  / |_|  |_||     |/  /
|  '  '  ||   [_ |     /   \_ |     ||   |   ||   [_       |  |  |     |      |  |   |  /   \_  |  |  |  _  /   \_
 \      / |     ||     \     ||     ||   |   ||     |      |  |  |     |      |  |   |  \     | |  |  |  |  \     |
  \_/\_/  |_____||_____|\____| \___/ |___|___||_____|      |__|   \___/       |__|  |____\____| |__|  |__|__|\____|
                                                                                                                    '
`

type scene int

const (
	sceneLobby      scene = iota
	sceneCreateGame scene = iota
	sceneJoinGame   scene = iota
	sceneGame       scene = iota
)

// Board dimensions (in terminal cells).
// Each board cell is 7 wide, 5 tall with a frame. Adjacent cells share their
// borders, so the step between cells is 6 (x) and 4 (y).
//   boardW = 2*6 + 7 = 19   (3 cells, rightmost adds 6+1)
//   boardH = 2*4 + 5 = 13
const (
	boardW = 19
	boardH = 13
	logW   = 30
)

// ── Lobby state ────────────────────────────────────────────────────────────

var currentScene = sceneLobby
var focusedItem = 0
var menuItems = []string{"Create Game", "Join Game"}

// gameError is shown in the lobby when a connection attempt fails.
var gameError string

// ── Game state ─────────────────────────────────────────────────────────────

var currentGameID string
var boardState [3][3]string // "", "X", "O"
var yourSymbol string
var currentTurn string
var boardCursorRow, boardCursorCol int
var moveLog []string
var gameIsOver bool
var gameResult string  // "X", "O", or "draw"
var winningCells []int // flat positions 0-8 that form the winning line
var winLine winLineType

// joiningGame is true while submitGameID has fired but the server hasn't
// responded yet — prevents drawJoinGame from re-enabling the input.
var joiningGame bool

type winLineType int

const (
	noLine    winLineType = iota
	horizLine winLineType = iota // ──
	vertLine  winLineType = iota // │
	diagDown  winLineType = iota // \
	diagUp    winLineType = iota // /
)

// ── Start ──────────────────────────────────────────────────────────────────

func Start(addr string) {
	serverAddr = addr

	g, err := gocui.NewGui(gocui.OutputTrue, true)
	if err != nil {
		log.Fatal(err)
	}
	defer g.Close()

	g.SetManagerFunc(layout)

	bindings := []struct {
		view    string
		key     interface{}
		handler func(*gocui.Gui, *gocui.View) error
	}{
		{"", gocui.KeyCtrlC, quit},
		{"", 'q', quit},
		{"", gocui.KeyArrowUp, arrowUp},
		{"", gocui.KeyArrowDown, arrowDown},
		{"", gocui.KeyArrowLeft, arrowLeft},
		{"", gocui.KeyArrowRight, arrowRight},
		{"", gocui.KeyEnter, confirmAction},
		{"", gocui.KeyEsc, goBack},
		{"game_id_input", gocui.KeyEnter, submitGameID},
	}

	for _, b := range bindings {
		var err error
		switch k := b.key.(type) {
		case gocui.Key:
			err = g.SetKeybinding(b.view, k, gocui.ModNone, b.handler)
		case rune:
			err = g.SetKeybinding(b.view, k, gocui.ModNone, b.handler)
		}
		if err != nil {
			log.Fatal(err)
		}
	}

	if err := g.MainLoop(); err != nil && err != gocui.ErrQuit {
		log.Fatal(err)
	}
}

// ── Layout dispatcher ──────────────────────────────────────────────────────

func layout(g *gocui.Gui) error {
	switch currentScene {
	case sceneLobby:
		return drawLobby(g)
	case sceneCreateGame:
		return drawCreateGame(g)
	case sceneJoinGame:
		return drawJoinGame(g)
	case sceneGame:
		return drawGame(g)
	}
	return nil
}

// ── Lobby ──────────────────────────────────────────────────────────────────

func drawLobby(g *gocui.Gui) error {
	maxX, maxY := g.Size()

	bannerWidth := 160
	bannerHeight := 8
	x0 := (maxX - bannerWidth) / 2
	y0 := (maxY - bannerHeight - 6) / 2

	v, err := g.SetView("welcome", x0, y0, x0+bannerWidth, y0+bannerHeight, 0)
	if err != nil && err != gocui.ErrUnknownView {
		return err
	}
	v.Clear()
	v.Frame = false
	fmt.Fprint(v, asciiArt)

	menuY := y0 + bannerHeight + 1
	menuWidth := 20
	menuX := (maxX - menuWidth) / 2

	for i, item := range menuItems {
		name := fmt.Sprintf("menu_%d", i)
		mv, err := g.SetView(name, menuX, menuY+i*2, menuX+menuWidth, menuY+i*2+2, 0)
		if err != nil && err != gocui.ErrUnknownView {
			return err
		}
		mv.Clear()
		mv.Frame = false
		if i == focusedItem {
			mv.FgColor = gocui.ColorDefault | gocui.AttrUnderline
		} else {
			mv.FgColor = gocui.ColorDefault
		}
		fmt.Fprintf(mv, "  %s", item)
	}

	// Error message (shown when a connection attempt failed).
	errY := menuY + len(menuItems)*2 + 1
	ev, err := g.SetView("error_msg", menuX-2, errY, menuX+menuWidth+2, errY+2, 0)
	if err != nil && err != gocui.ErrUnknownView {
		return err
	}
	ev.Clear()
	ev.Frame = false
	ev.FgColor = gocui.ColorRed
	if gameError != "" {
		fmt.Fprint(ev, gameError)
	}

	if _, err := g.SetCurrentView("welcome"); err != nil {
		return err
	}

	return drawFooter(g, "enter - select | q - quit")
}

// ── Create Game ────────────────────────────────────────────────────────────

func drawCreateGame(g *gocui.Gui) error {
	maxX, maxY := g.Size()

	centerX := maxX / 2
	centerY := maxY / 2

	var line1, line2 string
	if currentGameID == "" {
		line1 = "Connecting to server..."
		line2 = ""
	} else {
		line1 = fmt.Sprintf("Game successfully created. game_id: %s", currentGameID)
		line2 = "Waiting for player to join..."
	}

	width := len(line1) + 4
	v, err := g.SetView("create_game", centerX-width/2, centerY-2, centerX+width/2, centerY+2, 0)
	if err != nil && err != gocui.ErrUnknownView {
		return err
	}
	v.Clear()
	v.Frame = false
	fmt.Fprintln(v, line1)
	fmt.Fprint(v, line2)

	return drawFooter(g, "esc - back to main menu | q - quit")
}

// ── Join Game ──────────────────────────────────────────────────────────────

func drawJoinGame(g *gocui.Gui) error {
	maxX, maxY := g.Size()

	centerX := maxX / 2
	centerY := maxY / 2

	title := "Join a Game"
	tv, err := g.SetView("join_title", centerX-len(title)/2-1, centerY-3, centerX+len(title)/2+1, centerY-1, 0)
	if err != nil && err != gocui.ErrUnknownView {
		return err
	}
	tv.Clear()
	tv.Frame = false
	if joiningGame {
		fmt.Fprint(tv, "Connecting to server...")
	} else {
		fmt.Fprint(tv, title)
	}

	// Error display below title (connection errors from connectAndJoin).
	if gameError != "" {
		ev, err := g.SetView("join_error", centerX-len(gameError)/2-1, centerY-1, centerX+len(gameError)/2+1, centerY+1, 0)
		if err != nil && err != gocui.ErrUnknownView {
			return err
		}
		ev.Clear()
		ev.Frame = false
		ev.FgColor = gocui.ColorRed
		fmt.Fprint(ev, gameError)
	}

	inputWidth := 30
	iv, err := g.SetView("game_id_input", centerX-inputWidth/2, centerY-1, centerX+inputWidth/2, centerY+1, 0)
	if err != nil && err != gocui.ErrUnknownView {
		return err
	}
	iv.Title = "Game ID"
	iv.Editable = !joiningGame

	if !joiningGame {
		if _, err := g.SetCurrentView("game_id_input"); err != nil {
			return err
		}
	}

	return drawFooter(g, "enter - join | esc - back to main menu")
}

// ── Game Board ─────────────────────────────────────────────────────────────

func drawGame(g *gocui.Gui) error {
	maxX, maxY := g.Size()

	// Centre board + log panel horizontally.
	totalW := boardW + 2 + logW
	boardX := (maxX - totalW) / 2

	// Reserve 4 lines at top (header + status) and 2 at bottom (footer).
	extra := (maxY - 6 - boardH) / 2
	if extra < 0 {
		extra = 0
	}
	boardY := 4 + extra

	if err := drawGameHeader(g, maxX); err != nil {
		return err
	}
	if err := drawGameStatus(g, maxX); err != nil {
		return err
	}
	if err := drawBoardCells(g, boardX, boardY); err != nil {
		return err
	}

	logX := boardX + boardW + 2
	if err := drawMoveLog(g, logX, boardY); err != nil {
		return err
	}

	if _, err := g.SetCurrentView("game_header"); err != nil {
		return err
	}

	return drawFooter(g, "arrows - move cursor  enter - place  esc - back to menu")
}

func drawGameHeader(g *gocui.Gui, maxX int) error {
	v, err := g.SetView("game_header", 0, 0, maxX-1, 2, 0)
	if err != nil && err != gocui.ErrUnknownView {
		return err
	}
	v.Clear()
	v.Frame = false
	fmt.Fprintf(v, "  Game: %s   |   You are: %s", currentGameID, yourSymbol)
	return nil
}

func drawGameStatus(g *gocui.Gui, maxX int) error {
	v, err := g.SetView("game_status", 0, 2, maxX-1, 4, 0)
	if err != nil && err != gocui.ErrUnknownView {
		return err
	}
	v.Clear()
	v.Frame = false

	switch {
	case gameIsOver && gameResult == "draw":
		v.FgColor = gocui.ColorYellow
		fmt.Fprint(v, "  It's a draw!")
	case gameIsOver:
		v.FgColor = gocui.ColorGreen
		fmt.Fprintf(v, "  Player %s wins!", gameResult)
	case currentTurn == yourSymbol:
		v.FgColor = gocui.ColorDefault
		fmt.Fprintf(v, "  Your turn! (%s)", yourSymbol)
	default:
		v.FgColor = gocui.ColorDefault
		fmt.Fprintf(v, "  Waiting for opponent... (%s's turn)", currentTurn)
	}
	return nil
}

func drawBoardCells(g *gocui.Gui, boardX, boardY int) error {
	// Build a set of winning positions for O(1) lookup.
	winning := make(map[int]bool, len(winningCells))
	for _, pos := range winningCells {
		winning[pos] = true
	}

	for r := 0; r < 3; r++ {
		for c := 0; c < 3; c++ {
			x0 := boardX + c*6
			y0 := boardY + r*4
			name := fmt.Sprintf("cell_%d_%d", r, c)

			v, err := g.SetView(name, x0, y0, x0+6, y0+4, 0)
			if err != nil && err != gocui.ErrUnknownView {
				return err
			}
			v.Clear()

			pos := r*3 + c
			isCursor := r == boardCursorRow && c == boardCursorCol && !gameIsOver
			isWinner := winning[pos]

			v.BgColor = gocui.ColorDefault
			switch {
			case isCursor:
				v.BgColor = gocui.NewRGBColor(80, 80, 80)
				v.FgColor = gocui.ColorDefault
			case isWinner:
				switch boardState[r][c] {
				case "X":
					v.FgColor = gocui.ColorRed | gocui.AttrBold
				case "O":
					v.FgColor = gocui.ColorBlue | gocui.AttrBold
				}
			default:
				switch boardState[r][c] {
				case "X":
					v.FgColor = gocui.ColorRed | gocui.AttrBold
				case "O":
					v.FgColor = gocui.ColorBlue | gocui.AttrBold
				default:
					v.FgColor = gocui.ColorDefault
				}
			}

			sym := boardState[r][c]
			if sym == "" {
				sym = " "
			}
			if isWinner {
				fmt.Fprint(v, cellContent(sym, winLine))
			} else {
				// Centre the symbol vertically: blank first line, symbol on second.
				fmt.Fprintf(v, "\n  %s  ", sym)
			}
		}
	}
	return nil
}

func drawMoveLog(g *gocui.Gui, logX, logY int) error {
	v, err := g.SetView("move_log", logX, logY, logX+logW, logY+boardH-1, 0)
	if err != nil && err != gocui.ErrUnknownView {
		return err
	}
	v.Clear()
	v.Title = " Move Log "
	v.Autoscroll = true
	v.Wrap = true

	for _, entry := range moveLog {
		fmt.Fprintln(v, entry)
	}
	return nil
}

// ── Footer ─────────────────────────────────────────────────────────────────

func drawFooter(g *gocui.Gui, text string) error {
	maxX, maxY := g.Size()

	v, err := g.SetView("footer", 0, maxY-2, maxX-1, maxY, 0)
	if err != nil && err != gocui.ErrUnknownView {
		return err
	}
	v.Clear()
	v.Frame = false
	v.FgColor = gocui.ColorYellow
	fmt.Fprint(v, text)
	return nil
}

// ── Input handlers ─────────────────────────────────────────────────────────

func confirmAction(g *gocui.Gui, v *gocui.View) error {
	switch currentScene {
	case sceneLobby:
		return lobbySelect(g)
	case sceneGame:
		wsSendMove()
	}
	return nil
}

func lobbySelect(g *gocui.Gui) error {
	gameError = ""
	g.DeleteView("welcome")
	g.DeleteView("error_msg")
	for i := range menuItems {
		g.DeleteView(fmt.Sprintf("menu_%d", i))
	}
	switch focusedItem {
	case 0:
		yourSymbol = ""
		currentGameID = ""
		currentScene = sceneCreateGame
		go connectAndCreate(g)
	case 1:
		currentScene = sceneJoinGame
	}
	return nil
}

func arrowUp(g *gocui.Gui, v *gocui.View) error {
	switch currentScene {
	case sceneLobby:
		if focusedItem > 0 {
			focusedItem--
		}
	case sceneGame:
		if boardCursorRow > 0 {
			boardCursorRow--
		}
	}
	return nil
}

func arrowDown(g *gocui.Gui, v *gocui.View) error {
	switch currentScene {
	case sceneLobby:
		if focusedItem < len(menuItems)-1 {
			focusedItem++
		}
	case sceneGame:
		if boardCursorRow < 2 {
			boardCursorRow++
		}
	}
	return nil
}

func arrowLeft(g *gocui.Gui, v *gocui.View) error {
	if currentScene == sceneGame && boardCursorCol > 0 {
		boardCursorCol--
	}
	return nil
}

func arrowRight(g *gocui.Gui, v *gocui.View) error {
	if currentScene == sceneGame && boardCursorCol < 2 {
		boardCursorCol++
	}
	return nil
}

func goBack(g *gocui.Gui, v *gocui.View) error {
	switch currentScene {
	case sceneCreateGame:
		g.DeleteView("create_game")
		wsDisconnect()
	case sceneJoinGame:
		g.DeleteView("join_title")
		g.DeleteView("join_error")
		g.DeleteView("game_id_input")
		joiningGame = false
		wsDisconnect()
	case sceneGame:
		g.DeleteView("game_header")
		g.DeleteView("game_status")
		g.DeleteView("move_log")
		for r := 0; r < 3; r++ {
			for c := 0; c < 3; c++ {
				g.DeleteView(fmt.Sprintf("cell_%d_%d", r, c))
			}
		}
		wsDisconnect()
		resetGameState()
	default:
		return nil
	}
	currentScene = sceneLobby
	return nil
}

func quit(g *gocui.Gui, v *gocui.View) error {
	if currentScene == sceneJoinGame {
		return nil // don't quit while the user is typing a game ID
	}
	return gocui.ErrQuit
}

func submitGameID(g *gocui.Gui, v *gocui.View) error {
	gameID := strings.TrimSpace(v.Buffer())
	if gameID == "" {
		return nil
	}
	gameError = ""
	joiningGame = true
	go connectAndJoin(g, gameID)
	return nil
}

// ── Game helpers ───────────────────────────────────────────────────────────

// checkWinner checks boardState for a winner. Returns ("X"/"O"/"draw"/"") and
// the flat cell positions (0-8) of the winning line.
func checkWinner() (string, []int) {
	b := boardState
	patterns := [][3]int{
		{0, 1, 2}, {3, 4, 5}, {6, 7, 8}, // rows
		{0, 3, 6}, {1, 4, 7}, {2, 5, 8}, // cols
		{0, 4, 8}, {2, 4, 6},             // diagonals
	}
	for _, p := range patterns {
		r0, c0 := p[0]/3, p[0]%3
		r1, c1 := p[1]/3, p[1]%3
		r2, c2 := p[2]/3, p[2]%3
		if b[r0][c0] != "" && b[r0][c0] == b[r1][c1] && b[r1][c1] == b[r2][c2] {
			return b[r0][c0], []int{p[0], p[1], p[2]}
		}
	}
	for r := 0; r < 3; r++ {
		for c := 0; c < 3; c++ {
			if b[r][c] == "" {
				return "", nil
			}
		}
	}
	return "draw", nil
}

// winLineFromCells derives the line direction from three winning positions.
// Positions returned by checkWinner are always ascending, so consecutive
// differences uniquely identify the pattern:
//
//	diff 1 → same row (horizontal)   diff 3 → same column (vertical)
//	diff 4 → diagonal \              diff 2 → diagonal /
func winLineFromCells(cells []int) winLineType {
	if len(cells) != 3 {
		return noLine
	}
	switch cells[1] - cells[0] {
	case 1:
		return horizLine
	case 3:
		return vertLine
	case 4:
		return diagDown
	case 2:
		return diagUp
	}
	return noLine
}

// cellContent returns the string to write into a winning cell view.
// Each cell's content area is 5 chars wide and 3 lines tall.
func cellContent(sym string, lt winLineType) string {
	switch lt {
	case horizLine:
		return fmt.Sprintf("\n──%s──", sym)
	case vertLine:
		return fmt.Sprintf("  │  \n  %s  \n  │  ", sym)
	case diagDown:
		return fmt.Sprintf("\\    \n  %s  \n    \\", sym)
	case diagUp:
		return fmt.Sprintf("    /\n  %s  \n/    ", sym)
	}
	return fmt.Sprintf("\n  %s  ", sym)
}

func resetGameState() {
	boardState = [3][3]string{}
	yourSymbol = ""
	currentTurn = ""
	currentGameID = ""
	boardCursorRow, boardCursorCol = 0, 0
	moveLog = nil
	gameIsOver = false
	gameResult = ""
	winningCells = nil
	winLine = noLine
	joiningGame = false
	gameError = ""
}
