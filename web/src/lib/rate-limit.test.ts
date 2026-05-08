import { describe, it, expect, beforeEach } from "bun:test";
import { takeToken, _resetForTests } from "./rate-limit";

describe("rate-limit", () => {
  beforeEach(() => _resetForTests());

  it("allows the first request from an IP", () => {
    expect(takeToken("1.2.3.4", 0)).toBe(true);
  });

  it("allows up to 60 requests in the same second from one IP", () => {
    for (let i = 0; i < 60; i++) {
      expect(takeToken("1.2.3.4", 0)).toBe(true);
    }
  });

  it("rejects the 61st request in the same second", () => {
    for (let i = 0; i < 60; i++) takeToken("1.2.3.4", 0);
    expect(takeToken("1.2.3.4", 0)).toBe(false);
  });

  it("refills 1 token per second", () => {
    for (let i = 0; i < 60; i++) takeToken("1.2.3.4", 0);
    expect(takeToken("1.2.3.4", 999)).toBe(false);
    expect(takeToken("1.2.3.4", 1000)).toBe(true);
    expect(takeToken("1.2.3.4", 1000)).toBe(false);
  });

  it("isolates buckets per IP", () => {
    for (let i = 0; i < 60; i++) takeToken("1.2.3.4", 0);
    expect(takeToken("1.2.3.4", 0)).toBe(false);
    expect(takeToken("5.6.7.8", 0)).toBe(true);
  });
});
