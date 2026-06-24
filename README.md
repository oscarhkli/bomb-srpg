# Bomb Tactics
Turn-based Strategy Game

## Current Status

[Details here](/docs/roadmap.md)
- [x] Phase 1: CLI (legacy)
- [x] Phase 2: HTTP API
- [ ] Phase 3: Phaser UI
- [ ] Phase 4: Cloud deployment
- [ ] Phase 5: WebSocket
- [ ] Phase 6+: Other enhancements

## Prerequisites

To build and run this project, you need to install **Go** on your system. 

---

## Usage (backend only version)

Build the project and run the Web Server:

```bash
# Build
make build-server

# Run web server
./bin/srpg-web
```

Then go to http://localhost:8080/debug_board.html to view the debug board and play with the [test.http file](/scripts/test.http), or simply use `curl`.

[Visit oscarhkli.com for more](https://oscarhkli.com/)