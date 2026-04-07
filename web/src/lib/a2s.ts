import { Rcon } from "rcon-client";
import { RCON_PASSWORD } from "./config";

export interface ServerQueryResult {
  players: number;
  maxPlayers: number;
}

export async function queryServer(host: string, port: number): Promise<ServerQueryResult | null> {
  try {
    const rcon = await Rcon.connect({
      host,
      port,
      password: RCON_PASSWORD,
      timeout: 3000,
    });

    try {
      const response = await rcon.send("status");
      return parseStatusResponse(response);
    } finally {
      rcon.end();
    }
  } catch {
    return null;
  }
}

function parseStatusResponse(response: string): ServerQueryResult {
  let players = 0;
  let maxPlayers = 0;

  const lines = response.split("\n");
  for (const line of lines) {
    // Match "players : 0 humans, 0 bots (0 max)"
    const match = line.match(/players\s*:\s*(\d+)\s*humans?,\s*(\d+)\s*bots?\s*\((\d+)\s*max\)/i);
    if (match) {
      players = parseInt(match[1], 10) + parseInt(match[2], 10);
      maxPlayers = parseInt(match[3], 10);
    }
  }

  return { players, maxPlayers };
}
