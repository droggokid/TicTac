package tui

import (
	"fmt"
	"log"
	"math/rand"
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
)

var currentScene = sceneLobby
var focusedItem = 0
var menuItems = []string{"Create Game", "Join Game"}
var currentGameID = ""

func Start() {
	g, err := gocui.NewGui(gocui.OutputNormal, true)
	if err != nil {
		log.Fatal(err)
	}
	defer g.Close()

	g.SetManagerFunc(layout)

	if err := g.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, quit); err != nil {
		log.Fatal(err)
	}
	if err := g.SetKeybinding("", 'q', gocui.ModNone, quit); err != nil {
		log.Fatal(err)
	}
	if err := g.SetKeybinding("", gocui.KeyArrowUp, gocui.ModNone, menuUp); err != nil {
		log.Fatal(err)
	}
	if err := g.SetKeybinding("", gocui.KeyArrowDown, gocui.ModNone, menuDown); err != nil {
		log.Fatal(err)
	}
	if err := g.SetKeybinding("", gocui.KeyEnter, gocui.ModNone, menuSelect); err != nil {
		log.Fatal(err)
	}
	if err := g.SetKeybinding("", gocui.KeyEsc, gocui.ModNone, goBack); err != nil {
		log.Fatal(err)
	}
	if err := g.SetKeybinding("game_id_input", gocui.KeyEnter, gocui.ModNone, submitGameID); err != nil {
		log.Fatal(err)
	}

	if err := g.MainLoop(); err != nil && err != gocui.ErrQuit {
		log.Fatal(err)
	}
}

func layout(g *gocui.Gui) error {
	switch currentScene {
	case sceneLobby:
		return drawLobby(g)
	case sceneCreateGame:
		return drawCreateGame(g)
	case sceneJoinGame:
		return drawJoinGame(g)
	}
	return nil
}

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
		viewName := fmt.Sprintf("menu_%d", i)
		mv, err := g.SetView(viewName, menuX, menuY+i*2, menuX+menuWidth, menuY+i*2+2, 0)
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
	if _, err := g.SetCurrentView("welcome"); err != nil {
		return err
	}
	if err := drawFooter(g, "enter - select | q - quit"); err != nil {
		return err
	}
	return nil
}

func drawCreateGame(g *gocui.Gui) error {
	maxX, maxY := g.Size()

	centerX := maxX / 2
	centerY := maxY / 2

	line1 := fmt.Sprintf("Game successfully created. game_id: %s", currentGameID)
	line2 := "Waiting for player to join..."

	width := len(line1) + 4
	v, err := g.SetView("create_game", centerX-width/2, centerY-2, centerX+width/2, centerY+2, 0)
	if err != nil && err != gocui.ErrUnknownView {
		return err
	}
	v.Clear()
	v.Frame = false
	fmt.Fprintln(v, line1)
	fmt.Fprint(v, line2)

	if err := drawFooter(g, "esc - back to main menu | q - quit"); err != nil {
		return err
	}
	return nil
}

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
	fmt.Fprint(tv, title)

	inputWidth := 30
	iv, err := g.SetView("game_id_input", centerX-inputWidth/2, centerY-1, centerX+inputWidth/2, centerY+1, 0)
	if err != nil && err != gocui.ErrUnknownView {
		return err
	}
	iv.Title = "Game ID"
	iv.Editable = true

	if _, err := g.SetCurrentView("game_id_input"); err != nil {
		return err
	}

	if err := drawFooter(g, "enter - join | esc - back to main menu"); err != nil {
		return err
	}
	return nil
}

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

func menuSelect(g *gocui.Gui, v *gocui.View) error {
	if currentScene != sceneLobby {
		return nil
	}
	g.DeleteView("welcome")
	for i := range menuItems {
		g.DeleteView(fmt.Sprintf("menu_%d", i))
	}
	switch focusedItem {
	case 0:
		currentGameID = generateGameID()
		currentScene = sceneCreateGame
	case 1:
		currentScene = sceneJoinGame
	}
	return nil
}

func menuUp(g *gocui.Gui, v *gocui.View) error {
	if currentScene == sceneLobby && focusedItem > 0 {
		focusedItem--
	}
	return nil
}

func menuDown(g *gocui.Gui, v *gocui.View) error {
	if currentScene == sceneLobby && focusedItem < len(menuItems)-1 {
		focusedItem++
	}
	return nil
}

func quit(g *gocui.Gui, v *gocui.View) error {
	if currentScene == sceneJoinGame {
		return nil
	}
	return gocui.ErrQuit
}

func goBack(g *gocui.Gui, v *gocui.View) error {
	switch currentScene {
	case sceneCreateGame:
		g.DeleteView("create_game")
	case sceneJoinGame:
		g.DeleteView("join_title")
		g.DeleteView("game_id_input")
	default:
		return nil
	}
	currentScene = sceneLobby
	return nil
}

func generateGameID() string {
	const letters = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	b := make([]byte, 4)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func submitGameID(g *gocui.Gui, v *gocui.View) error {
	gameID := strings.TrimSpace(v.Buffer())
	if gameID == "" {
		return nil
	}
	// TODO: connect to server with gameID
	_ = gameID
	return nil
}
