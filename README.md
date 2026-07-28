# Bomb Tactics
Turn-based Strategy Game

## Current Status

Phase 4 (Cloud Deployment and Pixel Art) — In progress. See [docs/roadmap.md](/docs/roadmap.md) for the full phase breakdown.

## Prerequisites

To build and run this project, you need to install **Go** and **Node.js** (with npm) on your system.

Install frontend dependencies once:

```bash
cd web && npm install
```

---

## Usage

### For Development

Build the project and run the Web Server (must be running first — the frontend proxies API calls to it):

```bash
make run-server
```

Start the frontend dev server:

```bash
make web-dev
```

Then go to http://localhost:5173.

### For Deployment

Execute the following:

```bash
make web-build
make run-server
```

Then go to http://localhost:8080.

---

[Visit oscarhkli.com for more](https://oscarhkli.com/)