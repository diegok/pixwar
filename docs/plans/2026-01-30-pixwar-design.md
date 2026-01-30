# Pixwar - Multiplayer Territory Capture Game

A Qix-inspired multiplayer TUI game where players compete to capture territory on a shared board.

## Overview

**Binary**: `pixwar`
**Language**: Go
**Platform**: Terminal (TUI)
**Players**: 1-8

Single binary operates in two modes:
- `pixwar --server` - Host a game (also plays as client)
- `pixwar --join <ip>` - Connect to existing server

## Architecture

### Network Model

TCP-based client-server architecture. Server is authoritative:
- Clients send only input (direction changes)
- Server processes all game logic
- Server broadcasts state at fixed tick rate (~20/sec)

### Project Structure

```
pixwar/
├── cmd/pixwar/main.go      # CLI entry point, flag parsing
├── internal/
│   ├── server/             # Game server, lobby, networking
│   ├── client/             # Client connection, input handling
│   ├── game/               # Game logic, rules, collision
│   ├── protocol/           # Message types, serialization
│   └── ui/                 # TUI rendering
└── go.mod
```

## Game Mechanics

### Movement

- Continuous Snake-style movement (press direction, keep moving until changed)
- Arrow keys or WASD supported
- Constant speed (configurable by host)

### Trails & Territory

- Players outside their territory leave a **trail**
- Trails are vulnerable - another player touching your trail kills you
- Territory captured when trail connects back to:
  - Your own existing territory, OR
  - A board edge (edges are universal safe zones)
- Smaller side of the divided area is captured (Qix rule)

### Elimination

Players are eliminated when:
1. **Trail hit** - Another player crosses their trail
2. **Full enclosure** - Their entire territory gets surrounded (no edge access)

Eliminated players enter spectator mode with final rank/score displayed.

### Scoring

- Points accumulate based on territory owned (cells × time held)
- Elimination bonus: victim's points transfer to killer
- Final ranking by score, ties broken by survival time

### Starting Conditions

- **Center rush**: All players start at edges with NO initial territory
- Spawn positions distributed evenly along board edges
- 2-second spawn protection (invulnerability)

### Game End

Game ends when ANY of these conditions are met:
- Time limit reached (configurable, default: 5 minutes)
- Territory threshold reached (configurable, default: 95%)
- Only one player remains

## Server Configuration

```bash
pixwar --server [options]

Options:
  --port <port>        Listen port (default: 7777)
  --time <minutes>     Game duration (default: 5)
  --threshold <pct>    Territory % to end early (default: 95)
  --powerups           Enable power-ups (default: off)
```

Server displays local IP on startup for players to share.

## Screen Negotiation

1. Each client sends terminal dimensions on connect
2. Server tracks minimum width and height across all clients
3. Board size calculated as: `min(all_clients) × player_scaling`
4. Ensures every player sees the full board

## Connection Flow

1. Client connects via TCP to server IP:port
2. Client sends: player name + terminal dimensions
3. Server adds to lobby, broadcasts player list
4. Host presses Enter to start game
5. Server calculates board, assigns spawns/colors
6. Game loop begins

## Client Screens

### Lobby
- Connected players list with assigned colors
- Host sees "Press ENTER to start"
- Others see "Waiting for host..."

### Game
- Board fills terminal
- Bottom bar: score, time remaining, players alive
- Territory shown in player colors
- Trails shown with distinct pattern
- Player heads marked (● or @)

### Spectator (after elimination)
- Banner: "ELIMINATED - Rank #X - Score: XXXX"
- Tab to cycle between players
- Q to quit

### Results
- Final standings
- All scores and stats

## Visual Rendering

- **Primary**: Unicode blocks (█ territory, ░ trail, ● head)
- **Fallback**: ASCII (# territory, . trail, @ head)
- Auto-detect terminal capabilities
- 8 distinct player colors
- 30fps rendering with interpolation

## Power-ups (Optional)

Enabled with `--powerups` flag.

### Types

| Power-up | Symbol | Duration | Effect |
|----------|--------|----------|--------|
| Speed    | ⚡ / > | 3 sec    | 1.5× movement speed |
| Shield   | 🛡 / + | 4 sec    | Trail cannot be cut |
| Freeze   | ❄ / * | 2 sec    | Others move at 0.5× |

### Spawn Rules

- First spawns at 30 seconds
- New spawn every 15-20 seconds
- Maximum 2 on board
- Despawn after 10 seconds uncollected
- Only spawn on unclaimed cells

## Error Handling

### Network

- **Client disconnect**: Player eliminated, territory becomes unclaimed
- **Host disconnect**: Game ends, clients notified
- **Lag**: Server authoritative, late inputs applied to current state
- **No reconnection**: Disconnected = eliminated (v1 simplicity)

### Game

- **Simultaneous kills**: Both players die
- **Self-collision**: Not possible (always at head of own trail)
- **Minimum terminal**: 40×20 required, smaller clients rejected

### Shutdown

Server handles SIGINT/SIGTERM gracefully, notifies clients before exit.

## Out of Scope (v1)

Explicitly not building:
- Reconnection support
- Game replays
- Persistence / leaderboards
- Authentication
- In-game chat
- AI bots
- Custom maps / obstacles
- Team mode
- Web client
- Server discovery / matchmaking

## Future Considerations

If expanding later:
1. AI bots for practice/filling slots
2. Game replays
3. Team mode (2v2, 4v4)
4. Custom power-ups
