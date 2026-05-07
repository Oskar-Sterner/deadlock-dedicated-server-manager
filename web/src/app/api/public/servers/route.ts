import { NextResponse } from "next/server";
import { listServers } from "@/lib/servers";
import { getContainerInfo, getContainerStats } from "@/lib/docker";
import { queryServer } from "@/lib/a2s";
import { isServerSleeping, isServerWaking } from "@/lib/autosleep";
import { ensureInitialized } from "@/lib/init";
import { isPublicApiEnabled, SERVER_IP } from "@/lib/config";
import { takeToken } from "@/lib/rate-limit";
import { buildPublicServerPayload } from "@/lib/public-server-payload";

const CORS_HEADERS = {
  "access-control-allow-origin": "*",
  "access-control-allow-methods": "GET, OPTIONS",
  "access-control-max-age": "86400",
};

function clientIp(req: Request): string {
  const xff = req.headers.get("x-forwarded-for");
  if (xff) return xff.split(",")[0].trim();
  const real = req.headers.get("x-real-ip");
  if (real) return real;
  return "unknown";
}

function notFound() {
  return NextResponse.json({ error: "Not Found" }, { status: 404 });
}

export async function GET(req: Request) {
  if (!isPublicApiEnabled()) return notFound();

  if (!takeToken(clientIp(req))) {
    return NextResponse.json(
      { error: "Rate limited" },
      { status: 429, headers: { "retry-after": "60", ...CORS_HEADERS } },
    );
  }

  ensureInitialized();

  const servers = listServers();
  const now = new Date();

  const results = await Promise.all(
    servers.map(async (row) => {
      const containerInfo = row.container_id
        ? await getContainerInfo(row.container_id)
        : null;
      const stats = row.container_id && containerInfo?.state === "running"
        ? await getContainerStats(row.container_id)
        : null;
      const a2s = containerInfo?.state === "running"
        ? await queryServer("127.0.0.1", row.port)
        : null;
      const sleeping = isServerSleeping(row.id);
      const waking = isServerWaking(row.id);
      return buildPublicServerPayload({
        row, containerInfo, stats, a2s, sleeping, waking, now,
      });
    }),
  );

  return NextResponse.json(
    {
      servers: results,
      serverIp: SERVER_IP,
      fetchedAt: now.toISOString(),
    },
    {
      status: 200,
      headers: {
        "cache-control": "public, max-age=4, s-maxage=4",
        ...CORS_HEADERS,
      },
    },
  );
}

export async function OPTIONS(_req: Request) {
  return new NextResponse(null, { status: 204, headers: CORS_HEADERS });
}
