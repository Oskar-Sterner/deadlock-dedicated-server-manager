import { NextRequest, NextResponse } from "next/server";
import { requireAuth } from "@/lib/auth";
import { getServer, updateServer, deleteServer } from "@/lib/servers";
import { getContainerInfo, getContainerStats, startContainer, stopContainer, restartContainer } from "@/lib/docker";
import { queryServer } from "@/lib/a2s";
import { isServerSleeping, isServerWaking, manualWake, resetSleepState } from "@/lib/autosleep";
import { SERVER_IP } from "@/lib/config";

export async function GET(_req: NextRequest, { params }: { params: Promise<{ id: string }> }) {
  if (!(await requireAuth())) {
    return NextResponse.json({ error: "Unauthorized" }, { status: 401 });
  }

  const { id } = await params;
  const server = getServer(id);
  if (!server) return NextResponse.json({ error: "Not found" }, { status: 404 });

  const info = server.container_id ? await getContainerInfo(server.container_id) : null;
  const stats = server.container_id && info?.state === "running" ? await getContainerStats(server.container_id) : null;

  const a2s = info?.state === "running" ? await queryServer("127.0.0.1", server.port) : null;

  const sleeping = isServerSleeping(server.id);
  const waking = isServerWaking(server.id);

  return NextResponse.json({
    ...server,
    steam_pass: "***",
    status: sleeping ? "sleeping" : waking ? "waking" : (info?.state ?? "unknown"),
    startedAt: info?.startedAt ?? null,
    stats,
    players: a2s?.players ?? null,
    maxPlayers: a2s?.maxPlayers ?? null,
    serverIp: SERVER_IP,
  });
}

export async function POST(req: NextRequest, { params }: { params: Promise<{ id: string }> }) {
  if (!(await requireAuth())) {
    return NextResponse.json({ error: "Unauthorized" }, { status: 401 });
  }

  const { id } = await params;
  const server = getServer(id);
  if (!server) return NextResponse.json({ error: "Not found" }, { status: 404 });

  const { action, ...data } = await req.json();

  try {
    if (!server.container_id) {
      return NextResponse.json({ error: "No container" }, { status: 400 });
    }

    switch (action) {
      case "start":
        resetSleepState(id);
        await startContainer(server.container_id);
        break;
      case "stop":
        resetSleepState(id);
        await stopContainer(server.container_id);
        break;
      case "restart":
        resetSleepState(id);
        await restartContainer(server.container_id);
        break;
      case "wake":
        await manualWake(id);
        break;
      case "delete":
        await deleteServer(id, data.deleteFiles ?? false);
        return NextResponse.json({ ok: true, deleted: true });
      case "update":
        const updated = await updateServer(id, data);
        return NextResponse.json({ server: updated });
      default:
        return NextResponse.json({ error: "Unknown action" }, { status: 400 });
    }

    return NextResponse.json({ ok: true });
  } catch (err: any) {
    return NextResponse.json({ error: err.message }, { status: 500 });
  }
}
