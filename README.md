# TicTac

A multiplayer Tic Tac Toe game with WebSocket server.

## Architecture

- **Server** (`cmd/server`): WebSocket game server handling two-player matches
- **TUI** (`cmd/tui`): Terminal user interface client (in development)
- **Game Logic** (`internal/game`): Board state, move validation, win detection
- **Transport** (`internal/transport`): WebSocket message protocol

## Running the Server

### Local Development
```bash
go run ./cmd/server
```

### Docker
```bash
docker build -t tictac-server:latest .
docker run -p 8080:8080 tictac-server:latest
```

### Kubernetes (Production)
Deployed to k8s cluster in `dev` namespace. Manifests in separate homelab repo.

## Connecting to the Game

### Via TUI Client
```bash
# Connect to server (ensure port-forward is running)
go run ./cmd/tui/ -server 100.103.112.50:8080/game
```

### Via Tailscale (for remote play)
```bash
# Port forward the service on your server
kubectl port-forward -n dev svc/tictac-server 8080:8080 --address=0.0.0.0

# Then connect with TUI from any Tailscale device
go run ./cmd/tui/ -server 100.103.112.50:8080/game
```

### Manual Testing with websocat
```bash
websocat ws://100.103.112.50:8080/game
# or local: ws://localhost:8080/game
```

## Game Protocol

### Create a game (Player 1)
```json
{"type":"create_game"}
```
Response: `{"type":"create_game","gameId":"ABCD","yourSymbol":"X",...}`

### Join a game (Player 2)
```json
{"type":"join_game","gameId":"ABCD"}
```

### Make a move
```json
{"type":"make_move","row":0,"col":1}
```

### Receive game state
```json
{
  "type":"state_update",
  "board":[["X","",""],["","O",""],["","",""]],
  "currentTurn":"X"
}
```

## Development

### Build
```bash
go build ./cmd/server
```

### Test
```bash
go test ./internal/server -v
go test ./internal/game -v
```

### Docker Image
Published to: `ghcr.io/droggokid/tictac-server:latest`
