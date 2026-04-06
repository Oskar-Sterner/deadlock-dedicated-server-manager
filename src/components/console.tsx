"use client";

import { useEffect, useRef, useState } from "react";
import { motion } from "framer-motion";

interface ConsoleProps {
  serverId: string;
}

export function Console({ serverId }: ConsoleProps) {
  const [lines, setLines] = useState<string[]>([]);
  const [command, setCommand] = useState("");
  const [autoScroll, setAutoScroll] = useState(true);
  const bottomRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const eventSource = new EventSource(`/api/servers/${serverId}/logs`);

    eventSource.onmessage = (event) => {
      if (event.data === "[STREAM_END]") {
        eventSource.close();
        return;
      }
      try {
        const line = JSON.parse(event.data);
        setLines((prev) => [...prev.slice(-500), line]);
      } catch {
        setLines((prev) => [...prev.slice(-500), event.data]);
      }
    };

    eventSource.onerror = () => {
      eventSource.close();
    };

    return () => eventSource.close();
  }, [serverId]);

  useEffect(() => {
    if (autoScroll) bottomRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [lines, autoScroll]);

  async function sendCommand(e: React.FormEvent) {
    e.preventDefault();
    if (!command.trim()) return;

    setLines((prev) => [...prev, `> ${command}`]);

    const res = await fetch(`/api/servers/${serverId}/rcon`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ command }),
    });

    const data = await res.json();
    if (data.response) {
      setLines((prev) => [...prev, data.response]);
    } else if (data.error) {
      setLines((prev) => [...prev, `RCON Error: ${data.error}`]);
    }

    setCommand("");
  }

  return (
    <motion.div
      className="flex flex-col h-[calc(100vh-12rem)] bg-neutral-950 border border-neutral-800 rounded-lg overflow-hidden"
      initial={{ opacity: 0 }}
      animate={{ opacity: 1 }}
      transition={{ duration: 0.4 }}
    >
      <div className="flex items-center justify-between px-4 py-2 bg-neutral-900 border-b border-neutral-800">
        <span className="text-sm text-neutral-400 font-mono">Console</span>
        <motion.button
          whileTap={{ scale: 0.95 }}
          onClick={() => setAutoScroll(!autoScroll)}
          className={`cursor-pointer text-xs px-2 py-1 rounded ${autoScroll ? "bg-emerald-900 text-emerald-300" : "bg-neutral-800 text-neutral-500"}`}
        >
          Auto-scroll {autoScroll ? "ON" : "OFF"}
        </motion.button>
      </div>

      <div className="flex-1 overflow-y-auto p-4 font-mono text-sm leading-relaxed">
        {lines.map((line, i) => (
          <div key={i} className={`${line.startsWith(">") ? "text-amber-400" : line.includes("Error") || line.includes("ERROR") ? "text-[#f05c6a]" : "text-neutral-300"} whitespace-pre-wrap break-all`}>
            {line}
          </div>
        ))}
        <div ref={bottomRef} />
      </div>

      <form onSubmit={sendCommand} className="flex border-t border-neutral-800">
        <span className="px-3 py-2.5 text-neutral-600 font-mono text-sm">&gt;</span>
        <input
          value={command}
          onChange={(e) => setCommand(e.target.value)}
          placeholder="RCON command..."
          className="flex-1 py-2.5 bg-transparent text-neutral-100 font-mono text-sm placeholder-neutral-600 focus:outline-none"
        />
        <motion.button
          whileTap={{ scale: 0.95 }}
          type="submit"
          className="cursor-pointer px-4 py-2.5 text-sm text-neutral-400 hover:text-neutral-200 transition-colors"
        >
          Send
        </motion.button>
      </form>
    </motion.div>
  );
}
