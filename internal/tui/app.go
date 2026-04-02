package tui

import (
	"fmt"
	"log"

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

	if err := g.MainLoop(); err != nil && err != gocui.ErrQuit {
		log.Fatal(err)
	}
}

func layout(g *gocui.Gui) error {
	maxX, maxY := g.Size()

	// Center the welcome view
	width := 160
	height := 8
	x0 := (maxX - width) / 2
	y0 := (maxY - height) / 2

	v, err := g.SetView("welcome", x0, y0, x0+width, y0+height, 0)
	if err != nil && err != gocui.ErrUnknownView {
		return err
	}
	v.Clear()
	v.Frame = false
	fmt.Fprint(v, asciiArt)

	return nil
}

func quit(g *gocui.Gui, v *gocui.View) error {
	return gocui.ErrQuit
}
