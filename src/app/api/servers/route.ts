import { NextRequest, NextResponse } from "next/server";
import { requireAuth } from "@/lib/auth";
import { listServers, createServer, getNextPort } from "@/lib/servers";
import { getContainerInfo, getContainerStats } from "@/lib/docker";
import { queryServer } from "@/lib/a2s";
import { ensureInitialized } from "@/lib/init";
import { isServerSleeping, isServerWaking } from "@/lib/autosleep";
import { SERVER_IP } from "@/lib/config";

export async function GET() {
  ensureInitialized();
  if (!(await requireAuth())) {
    return NextResponse.json({ error: "Unauthorized" }, { status: 401 });
  }

  const servers = listServers();
  const results = await Promise.all(
    servers.map(async (s) => {
      const info = s.container_id ? await getContainerInfo(s.container_id) : null;
      const stats = s.container_id && info?.state === "running" ? await getContainerStats(s.container_id) : null;
      const a2s = info?.state === "running" ? await queryServer("127.0.0.1", s.port) : null;
      const sleeping = isServerSleeping(s.id);
      const waking = isServerWaking(s.id);
      return {
        ...s,
        steam_pass: "***",
        status: sleeping ? "sleeping" : waking ? "waking" : (info?.state ?? "unknown"),
        startedAt: info?.startedAt ?? null,
        stats,
        players: a2s?.players ?? null,
        maxPlayers: a2s?.maxPlayers ?? null,
      };
    })
  );

  return NextResponse.json({ servers: results, nextPort: getNextPort(), serverIp: SERVER_IP });
}

export async function POST(req: NextRequest) {
  if (!(await requireAuth())) {
    return NextResponse.json({ error: "Unauthorized" }, { status: 401 });
  }

  const body = await req.json();
  const { name, port, map, password, steam_login, steam_pass, steam_2fa } = body;

  if (!name || !port || !steam_login || !steam_pass) {
    return NextResponse.json({ error: "Missing required fields" }, { status: 400 });
  }

  try {
    const server = await createServer({
      name, port, map: map || "dl_streets", password: password || "",
      steam_login, steam_pass, steam_2fa: steam_2fa || "",
    });
    return NextResponse.json({ server });
  } catch (err: any) {
    return NextResponse.json({ error: err.message }, { status: 500 });
  }
}
