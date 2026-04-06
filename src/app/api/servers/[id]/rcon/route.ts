import { NextRequest, NextResponse } from "next/server";
import { requireAuth } from "@/lib/auth";
import { getServer } from "@/lib/servers";
import { sendRconCommand } from "@/lib/rcon";

export async function POST(req: NextRequest, { params }: { params: Promise<{ id: string }> }) {
  if (!(await requireAuth())) {
    return NextResponse.json({ error: "Unauthorized" }, { status: 401 });
  }

  const { id } = await params;
  const server = getServer(id);
  if (!server) return NextResponse.json({ error: "Not found" }, { status: 404 });

  const { command } = await req.json();
  if (!command) return NextResponse.json({ error: "No command" }, { status: 400 });

  try {
    const response = await sendRconCommand(server.port, command, server.password);
    return NextResponse.json({ response });
  } catch (err: any) {
    return NextResponse.json({ error: err.message }, { status: 500 });
  }
}
