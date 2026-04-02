package main

import (
	"flag"

	"tictac/internal/tui"
)

func main() {
	addr := flag.String("server", "localhost:8080", "game server address (host:port)")
	flag.Parse()
	tui.Start(*addr)
}
