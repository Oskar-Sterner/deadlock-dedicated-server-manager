import { describe, it, expect, beforeEach, afterEach, mock } from "bun:test";

// Mock the dependencies before importing the route module
mock.module("@/lib/servers", () => ({
  listServers: () => [
    {
      id: "21632714-0b94-44ce-abbc-90ea5edcc729",
      name: "DSpawn BHOP",
      port: 27015,
      map: "bhop_colour",
      password: "secret",
      steam_login: "tritan270",
      steam_pass: "9kfenz94)?+",
      steam_2fa: "ABCDE",
      skip_update: 0,
      container_id: "0123abcd",
      created_at: "2026-04-27T20:12:00Z",
    },
  ],
}));
mock.module("@/lib/docker", () => ({
  getContainerInfo: async () => ({
    id: "0123abcd",
    status: "running",
    state: "running",
    startedAt: "2026-05-07T18:04:11Z",
  }),
  getContainerStats: async () => ({ cpuPercent: 12.4, memoryMb: 1843, memoryLimitMb: 4096 }),
}));
mock.module("@/lib/a2s", () => ({
  queryServer: async () => ({ players: 3, maxPlayers: 24 }),
}));
mock.module("@/lib/autosleep", () => ({
  isServerSleeping: () => false,
  isServerWaking: () => false,
}));
mock.module("@/lib/init", () => ({ ensureInitialized: () => {} }));

import { _resetForTests } from "@/lib/rate-limit";

const origEnv = process.env.DDSM_PUBLIC_API_ENABLED;

afterEach(() => {
  process.env.DDSM_PUBLIC_API_ENABLED = origEnv;
});
beforeEach(() => {
  _resetForTests();
});

function makeRequest(opts: { ip?: string; method?: string } = {}) {
  return new Request("http://localhost:3000/api/public/servers", {
    method: opts.method ?? "GET",
    headers: { "x-forwarded-for": opts.ip ?? "1.2.3.4" },
  });
}

describe("GET /api/public/servers", () => {
  it("returns 404 when DDSM_PUBLIC_API_ENABLED is not set", async () => {
    delete process.env.DDSM_PUBLIC_API_ENABLED;
    const { GET } = await import("./route");
    const res = await GET(makeRequest());
    expect(res.status).toBe(404);
    const body = await res.json();
    expect(body).toEqual({ error: "Not Found" });
  });

  it("returns 200 with stripped payload when enabled", async () => {
    process.env.DDSM_PUBLIC_API_ENABLED = "true";
    const { GET } = await import("./route");
    const res = await GET(makeRequest());
    expect(res.status).toBe(200);
    const body = await res.json();
    expect(body.servers).toHaveLength(1);
    const s = body.servers[0];
    expect(s).toMatchObject({
      id: "21632714-0b94-44ce-abbc-90ea5edcc729",
      name: "DSpawn BHOP",
      port: 27015,
      map: "bhop_colour",
      status: "running",
      players: 3,
      maxPlayers: 24,
      cpuPercent: 12.4,
    });
    expect(s).not.toHaveProperty("password");
    expect(s).not.toHaveProperty("steam_login");
    expect(s).not.toHaveProperty("steam_pass");
    expect(s).not.toHaveProperty("container_id");
    expect(body).toHaveProperty("serverIp");
    expect(body).toHaveProperty("fetchedAt");
  });

  it("sets CORS and Cache-Control headers", async () => {
    process.env.DDSM_PUBLIC_API_ENABLED = "true";
    const { GET } = await import("./route");
    const res = await GET(makeRequest());
    expect(res.headers.get("access-control-allow-origin")).toBe("*");
    expect(res.headers.get("cache-control")).toContain("max-age=4");
  });

  it("returns 429 after 60 requests in a minute from one IP", async () => {
    process.env.DDSM_PUBLIC_API_ENABLED = "true";
    const { GET } = await import("./route");
    for (let i = 0; i < 60; i++) {
      const r = await GET(makeRequest({ ip: "9.9.9.9" }));
      expect(r.status).toBe(200);
    }
    const r = await GET(makeRequest({ ip: "9.9.9.9" }));
    expect(r.status).toBe(429);
    expect(r.headers.get("retry-after")).toBe("60");
    const body = await r.json();
    expect(body).toEqual({ error: "Rate limited" });
  });

  it("OPTIONS preflight returns 204 with CORS headers", async () => {
    process.env.DDSM_PUBLIC_API_ENABLED = "true";
    const { OPTIONS } = await import("./route");
    const res = await OPTIONS(makeRequest({ method: "OPTIONS" }));
    expect(res.status).toBe(204);
    expect(res.headers.get("access-control-allow-origin")).toBe("*");
    expect(res.headers.get("access-control-allow-methods")).toContain("GET");
  });
});
