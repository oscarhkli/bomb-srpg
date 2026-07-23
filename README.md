# Bomb Tactics
Turn-based Strategy Game

## Current Status

Phase 3 (Phaser UI) — in progress. See [docs/roadmap.md](/docs/roadmap.md) for the full phase breakdown.

## Prerequisites

To build and run this project, you need to install **Go** and **Node.js** (with npm) on your system.

Install frontend dependencies once:

```bash
cd web && npm install
```

---

## Usage

Build the project and run the Web Server (must be running first — the frontend proxies API calls to it):

```bash
make run-server
```

Start the frontend dev server:

```bash
make web-dev
```

Then go to http://localhost:5173.

[Visit oscarhkli.com for more](https://oscarhkli.com/)