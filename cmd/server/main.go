package main

import (
	"flag"
	"log"
	"net/http"

	"tictac/internal/server"
)

var addr = flag.String("addr", "localhost:8080", "http service address")

func main() {
	flag.Parse()
	log.SetFlags(0)

	http.HandleFunc("/game", server.GameHandler)

	log.Printf("Starting TicTac server on %s", *addr)
	log.Fatal(http.ListenAndServe(*addr, nil))
}
