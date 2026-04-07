import { NextRequest, NextResponse } from "next/server";
import { isSetupDone, setupPassword, createSession, SESSION_COOKIE, SESSION_MAX_AGE } from "@/lib/auth";

export async function GET() {
  return NextResponse.json({ setup: !isSetupDone() });
}

export async function POST(req: NextRequest) {
  if (isSetupDone()) {
    return NextResponse.json({ error: "Already configured" }, { status: 400 });
  }

  const { password } = await req.json();
  if (!password || password.length < 4) {
    return NextResponse.json({ error: "Password must be at least 4 characters" }, { status: 400 });
  }

  setupPassword(password);
  const token = createSession();

  const res = NextResponse.json({ ok: true });
  res.cookies.set(SESSION_COOKIE, token, {
    httpOnly: true,
    sameSite: "lax",
    maxAge: SESSION_MAX_AGE,
    path: "/",
  });
  return res;
}
