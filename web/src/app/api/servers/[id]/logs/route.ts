import { NextRequest } from "next/server";
import { requireAuth } from "@/lib/auth";
import { getServer } from "@/lib/servers";
import { docker } from "@/lib/docker";

export async function GET(_req: NextRequest, { params }: { params: Promise<{ id: string }> }) {
  if (!(await requireAuth())) {
    return new Response("Unauthorized", { status: 401 });
  }

  const { id } = await params;
  const server = getServer(id);
  if (!server?.container_id) {
    return new Response("Not found", { status: 404 });
  }

  const container = docker.getContainer(server.container_id);

  const logStream = await container.logs({
    follow: true,
    stdout: true,
    stderr: true,
    tail: 200,
    timestamps: false,
  });

  const stream = new ReadableStream({
    start(controller) {
      const encoder = new TextEncoder();

      logStream.on("data", (chunk: Buffer) => {
        let text = "";
        let offset = 0;
        while (offset < chunk.length) {
          if (offset + 8 > chunk.length) break;
          const size = chunk.readUInt32BE(offset + 4);
          offset += 8;
          if (offset + size > chunk.length) {
            text += chunk.slice(offset).toString("utf-8");
            break;
          }
          text += chunk.slice(offset, offset + size).toString("utf-8");
          offset += size;
        }

        const lines = text.split("\n").filter(Boolean);
        for (const line of lines) {
          controller.enqueue(encoder.encode(`data: ${JSON.stringify(line)}\n\n`));
        }
      });

      logStream.on("end", () => {
        controller.enqueue(encoder.encode("data: [STREAM_END]\n\n"));
        controller.close();
      });

      logStream.on("error", () => {
        controller.close();
      });
    },
    cancel() {
      (logStream as NodeJS.ReadableStream & { destroy: () => void }).destroy();
    },
  });

  return new Response(stream, {
    headers: {
      "Content-Type": "text/event-stream",
      "Cache-Control": "no-cache",
      "Connection": "keep-alive",
    },
  });
}
