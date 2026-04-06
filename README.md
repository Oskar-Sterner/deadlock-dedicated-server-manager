# DDSM — Deadlock Dedicated Server Manager

> **Work in progress.** This project is under active development and comes with no guarantees of stability or flawless operation. Use at your own risk.

A web dashboard for managing multiple [Deadlock](https://store.steampowered.com/app/1422450/Deadlock/) dedicated server instances on Linux.

Built on top of [deadlock-dedicated-proton-server](https://github.com/Oskar-Sterner/deadlock-dedicated-proton-server).

## Features

- **Multi-server management** — Create, start, stop, restart, and delete server instances from one dashboard
- **Live console** — Real-time log streaming via SSE with RCON command input
- **Server stats** — CPU, memory, uptime, and live player count via RCON
- **Auto-sleep** — Automatically stops idle servers after 5 minutes of no players. Wakes them when a player tries to connect
- **Connect button** — One-click copy of `connect IP:PORT` string and Steam deep-link
- **Password protected** — Single-password auth with HTTP-only session cookies

## Prerequisites

- Linux server (Ubuntu 22.04+ recommended)
- [Docker](https://docs.docker.com/engine/install/) and Docker Compose
- [Bun](https://bun.sh/) runtime
- [Node.js](https://nodejs.org/) 20+ (needed for `better-sqlite3` native compilation)
- The `deadlock-server` Docker image built from [deadlock-dedicated-proton-server](https://github.com/Oskar-Sterner/deadlock-dedicated-proton-server)
- A Steam account that owns Deadlock

## Quick Start

```bash
# 1. Make sure you've built the Deadlock server image first
# See: https://github.com/Oskar-Sterner/deadlock-dedicated-proton-server

# 2. Clone this repo
git clone https://github.com/Oskar-Sterner/deadlock-dedicated-server-manager.git
cd deadlock-dedicated-server-manager

# 3. Install dependencies
bun install

# 4. Build the native SQLite module
npm rebuild better-sqlite3

# 5. Configure
cp .env.example .env
nano .env  # Set your server's public IP

# 6. Create the data and servers directories
mkdir -p data /opt/deadlock-servers

# 7. Raise vm.max_map_count (required for Proton)
sudo sysctl -w vm.max_map_count=2147483642

# 8. Build and start
bun run build
bun run start -- -H 0.0.0.0 -p 3000
```

Visit `http://your-server-ip:3000` and set your dashboard password on first visit.

## Configuration

Copy `.env.example` to `.env` and configure:

```ini
# Public IP of your server (shown in connect strings)
DDSM_SERVER_IP=0.0.0.0

# RCON password for querying server status
DDSM_RCON_PASSWORD=ddsm_rcon_secret

# Directory for server instance data
DDSM_SERVERS_DIR=/opt/deadlock-servers

# Docker image name
DDSM_DOCKER_IMAGE=deadlock-server
```

## Creating a Server

1. Click **+ New Server** in the dashboard
2. Enter a name, port (auto-suggested), and map
3. Enter your Steam credentials (needed for SteamCMD to download game files)
4. Click **Create Server**

The first start downloads ~36 GB of game files. Subsequent starts are instant with `SKIP_UPDATE=1`.

## Auto-Sleep

DDSM automatically monitors player count via RCON. When a server has 0 players for 5 consecutive minutes:

1. The server container is stopped
2. A lightweight TCP/UDP listener takes over the port
3. When a player tries to connect, the listener detects it and starts the container
4. The player needs to reconnect after ~30-60 seconds while the server boots

This saves significant resources on CPU-intensive Proton-based servers.

## Tech Stack

- **Runtime:** [Bun](https://bun.sh/)
- **Framework:** [Next.js](https://nextjs.org/) 16 (App Router)
- **Styling:** [Tailwind CSS](https://tailwindcss.com/) v4
- **Animations:** [Framer Motion](https://www.framer.com/motion/)
- **Database:** SQLite via [better-sqlite3](https://github.com/WiseLibs/better-sqlite3)
- **Docker:** [Dockerode](https://github.com/apocas/dockerode)
- **RCON:** [rcon-client](https://github.com/gorcon/node-rcon)
- **Icons:** [Lucide](https://lucide.dev/)

## Project Structure

```
src/
├── app/                    # Next.js App Router pages + API routes
│   ├── api/
│   │   ├── auth/           # Login/logout
│   │   ├── setup/          # First-time password setup
│   │   └── servers/        # Server CRUD, logs (SSE), RCON
│   ├── login/              # Login page
│   ├── servers/
│   │   ├── new/            # Create server form
│   │   └── [id]/           # Console + settings pages
│   └── page.tsx            # Dashboard
├── components/             # React components
│   ├── console.tsx         # Live log terminal
│   ├── nav.tsx             # Navigation bar
│   ├── server-card.tsx     # Server card widget
│   └── stats-sidebar.tsx   # Stats + actions sidebar
└── lib/                    # Backend logic
    ├── a2s.ts              # RCON-based server querying
    ├── auth.ts             # Password + session management
    ├── autosleep.ts        # Auto-sleep/wake system
    ├── config.ts           # Environment variable config
    ├── db.ts               # SQLite database
    ├── docker.ts           # Docker container management
    ├── init.ts             # App initialization
    ├── rcon.ts             # RCON client
    └── servers.ts          # Server instance CRUD
```

## Known Limitations

- **Performance:** Deadlock has no native Linux server binary. Servers run through Proton + software Vulkan (llvmpipe), which is CPU-intensive. See the [proton server repo](https://github.com/Oskar-Sterner/deadlock-dedicated-proton-server#important-performance-expectations) for details.
- **Player count:** Queried via RCON `status` command. May show 0 during server initialization.
- **No HTTPS:** Runs on HTTP by default. Put behind a reverse proxy (nginx, Caddy) for HTTPS.
- **Single session:** Only one login session is active at a time.

## License

MIT
