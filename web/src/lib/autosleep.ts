import net from "net";
import dgram from "dgram";
import { listServers, getServer } from "./servers";
import { getContainerInfo, startContainer, stopContainer } from "./docker";
import { queryServer } from "./a2s";

const IDLE_TIMEOUT_MS = 5 * 60 * 1000; // 5 minutes
const POLL_INTERVAL_MS = 15 * 1000; // check every 15 seconds

interface SleepState {
  lastSeenPlayers: number;
  emptyTimestamp: number | null; // when the server was first seen empty
  sleeping: boolean;
  wakeListener: { tcp: net.Server; udp: dgram.Socket } | null;
  waking: boolean; // currently booting up
}

const state = new Map<string, SleepState>();
let pollTimer: ReturnType<typeof setInterval> | null = null;

function getState(serverId: string): SleepState {
  if (!state.has(serverId)) {
    state.set(serverId, {
      lastSeenPlayers: -1,
      emptyTimestamp: null,
      sleeping: false,
      wakeListener: null,
      waking: false,
    });
  }
  return state.get(serverId)!;
}

export function isServerSleeping(serverId: string): boolean {
  return getState(serverId).sleeping;
}

export function isServerWaking(serverId: string): boolean {
  return getState(serverId).waking;
}

async function pollServers() {
  const servers = listServers();

  for (const server of servers) {
    if (!server.container_id) continue;

    const s = getState(server.id);

    // Skip if currently waking up
    if (s.waking) continue;

    // If sleeping, the wake listener handles it
    if (s.sleeping) continue;

    // Check if container is running
    const info = await getContainerInfo(server.container_id);
    if (!info || info.state !== "running") continue;

    // Query player count via RCON
    const result = await queryServer("127.0.0.1", server.port);
    const players = result?.players ?? -1;

    if (players === -1) {
      // RCON failed, server might still be booting — skip
      continue;
    }

    if (players > 0) {
      // Players online — reset idle timer
      s.lastSeenPlayers = players;
      s.emptyTimestamp = null;
    } else {
      // Empty server
      if (s.emptyTimestamp === null) {
        s.emptyTimestamp = Date.now();
        console.log(`[autosleep] ${server.name} (port ${server.port}): empty, starting idle timer`);
      } else if (Date.now() - s.emptyTimestamp >= IDLE_TIMEOUT_MS) {
        // Idle timeout reached — put server to sleep
        console.log(`[autosleep] ${server.name} (port ${server.port}): idle for 5 min, sleeping`);
        await sleepServer(server.id, server.port, server.container_id);
      }
    }
  }
}

async function sleepServer(serverId: string, port: number, containerId: string) {
  const s = getState(serverId);

  try {
    // Stop the container (don't remove — just stop)
    await stopContainer(containerId);
  } catch (err) {
    console.error(`[autosleep] Failed to stop container:`, err);
    return;
  }

  s.sleeping = true;
  s.emptyTimestamp = null;

  // Start wake listeners on the server's port
  startWakeListener(serverId, port, containerId);
}

function startWakeListener(serverId: string, port: number, containerId: string) {
  const s = getState(serverId);

  // TCP listener
  const tcp = net.createServer((socket) => {
    console.log(`[autosleep] TCP connection on port ${port} — waking server ${serverId}`);
    socket.destroy();
    wakeServer(serverId, port, containerId);
  });

  tcp.on("error", (err: NodeJS.ErrnoException) => {
    if (err.code === "EADDRINUSE") {
      console.log(`[autosleep] Port ${port} TCP still in use, retrying in 5s`);
      setTimeout(() => {
        tcp.close();
        startWakeListener(serverId, port, containerId);
      }, 5000);
    }
  });

  // UDP listener
  const udp = dgram.createSocket("udp4");

  udp.on("message", () => {
    console.log(`[autosleep] UDP packet on port ${port} — waking server ${serverId}`);
    wakeServer(serverId, port, containerId);
  });

  udp.on("error", (err: NodeJS.ErrnoException) => {
    if (err.code === "EADDRINUSE") {
      console.log(`[autosleep] Port ${port} UDP still in use, retrying in 5s`);
      setTimeout(() => {
        udp.close();
        startWakeListener(serverId, port, containerId);
      }, 5000);
    }
  });

  // Wait a moment for Docker to release the port, then bind
  setTimeout(() => {
    try {
      tcp.listen(port, "0.0.0.0", () => {
        console.log(`[autosleep] Wake listener TCP on port ${port}`);
      });
      udp.bind(port, "0.0.0.0", () => {
        console.log(`[autosleep] Wake listener UDP on port ${port}`);
      });
      s.wakeListener = { tcp, udp };
    } catch (err) {
      console.error(`[autosleep] Failed to start wake listener:`, err);
    }
  }, 3000);
}

async function wakeServer(serverId: string, port: number, containerId: string) {
  const s = getState(serverId);

  // Prevent multiple simultaneous wakes
  if (s.waking) return;
  s.waking = true;

  // Close wake listeners first to free the port
  if (s.wakeListener) {
    s.wakeListener.tcp.close();
    s.wakeListener.udp.close();
    s.wakeListener = null;
  }

  console.log(`[autosleep] Waking server ${serverId} on port ${port}`);

  // Small delay to ensure port is released
  await new Promise((r) => setTimeout(r, 1000));

  try {
    await startContainer(containerId);
    console.log(`[autosleep] Server ${serverId} started`);
  } catch (err) {
    console.error(`[autosleep] Failed to start container:`, err);
  }

  s.sleeping = false;
  s.waking = false;
  s.emptyTimestamp = null;
  s.lastSeenPlayers = -1;
}

// Manually wake a sleeping server (from the dashboard)
export async function manualWake(serverId: string) {
  const server = getServer(serverId);
  if (!server?.container_id) return;

  const s = getState(serverId);
  if (!s.sleeping) return;

  await wakeServer(serverId, server.port, server.container_id);
}

// Manually reset sleep state (e.g. when user manually starts/stops)
export function resetSleepState(serverId: string) {
  const s = getState(serverId);
  if (s.wakeListener) {
    s.wakeListener.tcp.close();
    s.wakeListener.udp.close();
    s.wakeListener = null;
  }
  s.sleeping = false;
  s.waking = false;
  s.emptyTimestamp = null;
  s.lastSeenPlayers = -1;
}

export function startAutoSleep() {
  if (pollTimer) return;
  console.log("[autosleep] Started — polling every 15s, idle timeout 5 min");
  pollTimer = setInterval(pollServers, POLL_INTERVAL_MS);
}

export function stopAutoSleep() {
  if (pollTimer) {
    clearInterval(pollTimer);
    pollTimer = null;
  }
  // Clean up all listeners
  for (const [, s] of state) {
    if (s.wakeListener) {
      s.wakeListener.tcp.close();
      s.wakeListener.udp.close();
    }
  }
  state.clear();
}
