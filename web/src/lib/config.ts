export const SERVER_IP = process.env.DDSM_SERVER_IP || "0.0.0.0";
export const RCON_PASSWORD = process.env.DDSM_RCON_PASSWORD || "ddsm_rcon_secret";
export const SERVERS_DIR = process.env.DDSM_SERVERS_DIR || "/opt/deadlock-servers";
export const DEADLOCK_IMAGE = process.env.DDSM_DOCKER_IMAGE || "deadlock-server";

/**
 * When true, the route at /api/public/servers becomes available.
 * The endpoint is read-only, returns a payload stripped of all credentials,
 * and is rate-limited to 60 req/min per IP.
 *
 * Off by default — when off, the route returns 404 (no leak that it exists).
 */
export const PUBLIC_API_ENABLED = process.env.DDSM_PUBLIC_API_ENABLED === "true";
