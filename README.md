# Pixwar

A multiplayer Qix-inspired territory capture game for the terminal.

## Overview

Pixwar is a fast-paced multiplayer game where players compete to capture territory on a shared board. Leave trails as you move, close them off to claim territory, but watch out - if another player crosses your trail, you're eliminated!

## Features

- **Multiplayer**: 1-8 players over TCP/IP
- **Terminal UI**: Runs in any terminal with color support
- **Simple controls**: Arrow keys or WASD to move
- **Qix-style gameplay**: Capture territory by enclosing areas with your trail

## Installation

```bash
go install github.com/diegok/pixwar/cmd/pixwar@latest
```

Or build from source:

```bash
git clone https://github.com/diegok/pixwar.git
cd pixwar
go build -o pixwar ./cmd/pixwar
```

## How to Play

### Start a Server

```bash
pixwar --server --name YourName
```

The server will display IP addresses that other players can use to connect.

### Join a Game

```bash
pixwar --join <server-ip> --name YourName
```

### Controls

| Key | Action |
|-----|--------|
| Arrow keys / WASD | Move |
| Enter | Start game (host only) |
| Tab | Cycle players (spectator mode) |
| Q / Escape | Quit |

## Game Rules

1. **Movement**: Your player moves continuously in the current direction
2. **Trails**: When outside your territory, you leave a vulnerable trail
3. **Capture**: Close your trail by returning to your territory or a board edge
4. **Elimination**: If another player crosses your trail, you're out
5. **Winning**: Game ends when time runs out, one player remains, or someone captures 95% of the board

## Server Options

```
pixwar --server [options]

Options:
  --port <port>       Server port (default: 7777)
  --time <minutes>    Game duration (default: 5)
  --threshold <pct>   Territory % to win (default: 95)
  --powerups          Enable power-ups
  --name <name>       Your player name
```

## Architecture

- **Single binary**: Same executable runs as server or client
- **Authoritative server**: Server handles all game logic at 20 ticks/sec
- **TCP networking**: Reliable message delivery with gob encoding
- **TUI**: Built with [tcell](https://github.com/gdamore/tcell) for cross-platform terminal support

## License

MIT
