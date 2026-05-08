const BUCKET_SIZE = 60;          // max tokens per IP
const REFILL_INTERVAL_MS = 1000; // 1 token per second

interface Bucket {
  tokens: number;
  lastRefill: number;
}

const buckets = new Map<string, Bucket>();

/**
 * Consume one token for `ip`. Returns true if allowed, false if over the limit.
 * `now` is injected for testability; pass `Date.now()` in production callers.
 */
export function takeToken(ip: string, now: number = Date.now()): boolean {
  let b = buckets.get(ip);
  if (!b) {
    b = { tokens: BUCKET_SIZE, lastRefill: now };
    buckets.set(ip, b);
  }
  const elapsed = now - b.lastRefill;
  if (elapsed >= REFILL_INTERVAL_MS) {
    const refill = Math.floor(elapsed / REFILL_INTERVAL_MS);
    b.tokens = Math.min(BUCKET_SIZE, b.tokens + refill);
    b.lastRefill += refill * REFILL_INTERVAL_MS;
  }
  if (b.tokens <= 0) return false;
  b.tokens -= 1;
  return true;
}

/** Test-only reset. */
export function _resetForTests() {
  buckets.clear();
}
