import { v4 as uuid } from "uuid";
import { getDb } from "./db";
import { createContainer, removeContainer, startContainer } from "./docker";
import { SERVERS_DIR } from "./config";
import { resetSleepState } from "./autosleep";
import fs from "fs";
import path from "path";
const START_SCRIPT_SRC = path.join(process.cwd(), "start.sh");

export interface ServerRow {
  id: string;
  name: string;
  port: number;
  map: string;
  password: string;
  steam_login: string;
  steam_pass: string;
  steam_2fa: string;
  skip_update: number;
  container_id: string | null;
  created_at: string;
}

export function listServers(): ServerRow[] {
  return getDb().prepare("SELECT * FROM servers ORDER BY created_at DESC").all() as ServerRow[];
}

export function getServer(id: string): ServerRow | null {
  return (getDb().prepare("SELECT * FROM servers WHERE id = ?").get(id) as ServerRow) ?? null;
}

export function getNextPort(): number {
  const row = getDb().prepare("SELECT MAX(port) as maxPort FROM servers").get() as { maxPort: number | null };
  return (row?.maxPort ?? 27014) + 1;
}

// Where the Deadlock game directory lives for this server. Servers created
// via the CLI use overlayfs and the game dir is under merged/, while plain
// installs put it directly under the server volume.
function deadlockGameDir(id: string): string {
  const overlayed = path.join(SERVERS_DIR, id, "merged", "Deadlock", "game");
  if (fs.existsSync(overlayed)) return overlayed;
  return path.join(SERVERS_DIR, id, "Deadlock", "game");
}

// Mirror the dashboard "Server Name" into the in-game hostname convar by
// writing citadel/cfg/server.cfg, which Source 2 auto-execs on map load.
// Embedded double quotes are stripped — Source's quoting can't escape them.
export function writeHostnameCfg(id: string, hostname: string): void {
  const cfgDir = path.join(deadlockGameDir(id), "citadel", "cfg");
  if (!fs.existsSync(path.dirname(cfgDir))) return; // game dir not provisioned yet
  fs.mkdirSync(cfgDir, { recursive: true });
  const sanitized = hostname.replace(/"/g, "");
  fs.writeFileSync(path.join(cfgDir, "server.cfg"), `hostname "${sanitized}"\n`);
}

export async function createServer(data: {
  name: string;
  port: number;
  map: string;
  password: string;
  steam_login: string;
  steam_pass: string;
  steam_2fa: string;
}): Promise<ServerRow> {
  const id = uuid();
  const volumePath = path.join(SERVERS_DIR, id);

  fs.mkdirSync(volumePath, { recursive: true });
  if (fs.existsSync(START_SCRIPT_SRC)) {
    fs.copyFileSync(START_SCRIPT_SRC, path.join(volumePath, "start.sh"));
    fs.chmodSync(path.join(volumePath, "start.sh"), 0o755);
  }

  const containerName = `deadlock-${id.slice(0, 8)}`;
  const containerId = await createContainer({
    name: containerName,
    port: data.port,
    env: {
      PORT: String(data.port),
      MAP: data.map,
      SERVER_PASSWORD: data.password,
      STEAM_LOGIN: data.steam_login,
      STEAM_PASSWORD: data.steam_pass,
      STEAM_2FA_CODE: data.steam_2fa,
      SKIP_UPDATE: "0",
    },
    volumePath,
  });

  getDb().prepare(`
    INSERT INTO servers (id, name, port, map, password, steam_login, steam_pass, steam_2fa, skip_update, container_id)
    VALUES (?, ?, ?, ?, ?, ?, ?, ?, 0, ?)
  `).run(id, data.name, data.port, data.map, data.password, data.steam_login, data.steam_pass, data.steam_2fa, containerId);

  writeHostnameCfg(id, data.name);

  await startContainer(containerId);
  return getServer(id)!;
}

export async function updateServer(id: string, data: {
  name: string;
  port: number;
  map: string;
  password: string;
  steam_login: string;
  steam_pass: string;
  steam_2fa: string;
  skip_update: number;
}): Promise<ServerRow> {
  const server = getServer(id);
  if (!server) throw new Error("Server not found");

  // Free any wake-listener bound to the server's port; otherwise the
  // recreate below fails to bind with "address already in use".
  resetSleepState(id);

  if (server.container_id) {
    try { await removeContainer(server.container_id); } catch { /* ok */ }
  }

  // Brief pause so the kernel fully releases the port (TCP TIME_WAIT,
  // listener teardown, etc.) before the new container tries to bind.
  await new Promise((r) => setTimeout(r, 500));

  const volumePath = path.join(SERVERS_DIR, id);
  const containerName = `deadlock-${id.slice(0, 8)}`;

  const containerId = await createContainer({
    name: containerName,
    port: data.port,
    env: {
      PORT: String(data.port),
      MAP: data.map,
      SERVER_PASSWORD: data.password,
      STEAM_LOGIN: data.steam_login,
      STEAM_PASSWORD: data.steam_pass,
      STEAM_2FA_CODE: data.steam_2fa,
      SKIP_UPDATE: String(data.skip_update),
    },
    volumePath,
  });

  getDb().prepare(`
    UPDATE servers SET name=?, port=?, map=?, password=?, steam_login=?, steam_pass=?, steam_2fa=?, skip_update=?, container_id=?
    WHERE id=?
  `).run(data.name, data.port, data.map, data.password, data.steam_login, data.steam_pass, data.steam_2fa, data.skip_update, containerId, id);

  writeHostnameCfg(id, data.name);

  await startContainer(containerId);
  return getServer(id)!;
}

export async function deleteServer(id: string, deleteFiles: boolean = false) {
  const server = getServer(id);
  if (!server) throw new Error("Server not found");

  if (server.container_id) {
    try { await removeContainer(server.container_id); } catch { /* ok */ }
  }

  if (deleteFiles) {
    const volumePath = path.join(SERVERS_DIR, id);
    fs.rmSync(volumePath, { recursive: true, force: true });
  }

  getDb().prepare("DELETE FROM servers WHERE id = ?").run(id);
}
