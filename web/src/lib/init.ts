import { startAutoSleep } from "./autosleep";

let initialized = false;

export function ensureInitialized() {
  if (initialized) return;
  initialized = true;
  const enabled = process.env.DDSM_AUTOSLEEP_ENABLED;
  if (enabled === undefined || enabled === "true" || enabled === "1") {
    startAutoSleep();
  } else {
    console.log("[autosleep] Disabled via DDSM_AUTOSLEEP_ENABLED");
  }
}
