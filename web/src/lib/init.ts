import { startAutoSleep } from "./autosleep";

let initialized = false;

export function ensureInitialized() {
  if (initialized) return;
  initialized = true;
  startAutoSleep();
}
