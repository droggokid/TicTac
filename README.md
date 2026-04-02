# TicTac

A multiplayer Tic Tac Toe game that runs entirely in the terminal. Two players connect to a shared server over WebSocket and play in real time, each from their own terminal.

```
  __    __    ___  _        __   ___   ___ ___    ___      ______   ___       ______  ____   __ ______
 |  |__|  |  /  _]| |      /  ] /   \ |   |   |  /  _]    |      | /   \     |      ||    | /  ]      |
 |  |  |  | /  [_ | |     /  / |     || _   _ | /  [_     |      ||     |    |      | |  | /  /|      |
 |  |  |  ||    _]| |___ /  /  |  O  ||  \_/  ||    _]    |_|  |_||  O  |    |_|  |_| |  |/  / |_|  |_|
```

## How to Play

### 1. Start the server

```bash
go run ./cmd/server/
# Custom port:
go run ./cmd/server/ -addr localhost:9090
```

### 2. Player 1 — Create a game

```bash
go run ./cmd/tui/ -server <host>:8080
```

Select **Create Game**. A 4-letter game ID will appear on screen — share it with your friend.

### 3. Player 2 — Join the game

```bash
go run ./cmd/tui/ -server <host>:8080
```

Select **Join Game**, type the game ID, and press Enter. Both terminals transition to the game board simultaneously.

---

## Controls

| Key | Action |
|-----|--------|
| `↑` `↓` `←` `→` | Move cursor |
| `Enter` | Place your mark |
| `Esc` | Go back / disconnect |
| `q` | Quit |

---

## Connecting to a Remote Server

Point the client at any reachable host — local network, Tailscale, or a VPS:

```bash
go run ./cmd/tui/ -server 100.103.112.50:8080
```

The server is also available as a Docker image:

```bash
docker run -p 8080:8080 ghcr.io/droggokid/tictac-server:latest
```

---

## Manual Testing with websocat

```bash
websocat ws://localhost:8080/game

# Create a game
{"type":"create_game"}

# Join a game
{"type":"join_game","gameId":"ABCD"}

# Make a move (row/col are 0-indexed)
{"type":"make_move","row":1,"col":2}
```

---

## Project Structure

```
cmd/
  server/   — WebSocket game server
  tui/      — Terminal UI client
internal/
  game/     — Board, move validation, win detection
  server/   — WebSocket hub and game room logic
  transport/ — Shared JSON message types
  tui/      — gocui views and WebSocket client
```

## Build

```bash
go build -o tictac-server ./cmd/server/
go build -o tictac       ./cmd/tui/
```
