import { hashSync, compareSync } from "bcryptjs";
import { randomUUID } from "crypto";
import { getSetting, setSetting } from "./db";
import { cookies } from "next/headers";

const SESSION_COOKIE = "dlm_session";
const SESSION_MAX_AGE = 60 * 60 * 24 * 7; // 7 days

export function isSetupDone(): boolean {
  return getSetting("dashboard_password") !== null;
}

export function setupPassword(password: string) {
  const hash = hashSync(password, 10);
  setSetting("dashboard_password", hash);
  const secret = randomUUID();
  setSetting("session_secret", secret);
}

export function verifyPassword(password: string): boolean {
  const hash = getSetting("dashboard_password");
  if (!hash) return false;
  return compareSync(password, hash);
}

export function createSession(): string {
  const token = randomUUID();
  setSetting("session_token", token);
  return token;
}

export function validateSession(token: string): boolean {
  const stored = getSetting("session_token");
  return stored !== null && stored === token;
}

export async function requireAuth(): Promise<boolean> {
  const cookieStore = await cookies();
  const token = cookieStore.get(SESSION_COOKIE)?.value;
  if (!token) return false;
  return validateSession(token);
}

export { SESSION_COOKIE, SESSION_MAX_AGE };
