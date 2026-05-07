import type { ServerRow } from "./servers";
import type { ContainerInfo, ContainerStats } from "./docker";
import type { ServerQueryResult } from "./a2s";

export type PublicServerStatus =
  | "running" | "exited" | "sleeping" | "waking"
  | "created" | "restarting" | "dead" | "unknown";

export interface PublicServer {
  id: string;
  name: string;
  port: number;
  map: string;
  status: PublicServerStatus;
  players: number;
  maxPlayers: number;
  cpuPercent: number;
  memoryMb: number;
  memoryLimitMb: number;
  memoryPercent: number;
  startedAt: string | null;
  uptimeSeconds: number;
}

export interface BuildArgs {
  row: ServerRow;
  containerInfo: ContainerInfo | null;
  stats: ContainerStats | null;
  a2s: ServerQueryResult | null;
  sleeping: boolean;
  waking: boolean;
  now: Date;
}

const KNOWN_CONTAINER_STATES = new Set([
  "running", "exited", "created", "restarting", "dead",
]);

export function buildPublicServerPayload(args: BuildArgs): PublicServer {
  const { row, containerInfo, stats, a2s, sleeping, waking, now } = args;

  const baseStatus: PublicServerStatus = sleeping
    ? "sleeping"
    : waking
      ? "waking"
      : containerInfo
        ? (KNOWN_CONTAINER_STATES.has(containerInfo.state)
            ? (containerInfo.state as PublicServerStatus)
            : "unknown")
        : "unknown";

  const startedAt = containerInfo?.startedAt && containerInfo.startedAt !== ""
    ? containerInfo.startedAt
    : null;

  const uptimeSeconds = startedAt
    ? Math.max(0, Math.floor((now.getTime() - new Date(startedAt).getTime()) / 1000))
    : 0;

  const memoryMb = stats?.memoryMb ?? 0;
  const memoryLimitMb = stats?.memoryLimitMb ?? 0;
  const memoryPercent = memoryLimitMb > 0
    ? Math.round((memoryMb / memoryLimitMb) * 1000) / 10
    : 0;

  return {
    id: row.id,
    name: row.name,
    port: row.port,
    map: row.map,
    status: baseStatus,
    players: a2s?.players ?? 0,
    maxPlayers: a2s?.maxPlayers ?? 0,
    cpuPercent: stats?.cpuPercent ?? 0,
    memoryMb,
    memoryLimitMb,
    memoryPercent,
    startedAt,
    uptimeSeconds,
  };
}
