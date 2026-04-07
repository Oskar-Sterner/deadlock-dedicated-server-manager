"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { motion } from "framer-motion";
import { Play, CircleStop, RotateCcw, Settings, Copy, ExternalLink, Loader2 } from "lucide-react";

interface StatsSidebarProps {
  serverId: string;
  onAction: (action: string) => void;
}

interface ServerDetail {
  name: string;
  port: number;
  map: string;
  status: string;
  startedAt: string | null;
  container_id: string | null;
  stats: { cpuPercent: number; memoryMb: number; memoryLimitMb: number } | null;
  players: number | null;
  maxPlayers: number | null;
  serverIp: string;
}

export function StatsSidebar({ serverId, onAction }: StatsSidebarProps) {
  const [data, setData] = useState<ServerDetail | null>(null);
  const [copied, setCopied] = useState(false);

  async function fetchData() {
    const res = await fetch(`/api/servers/${serverId}`);
    if (res.ok) setData(await res.json());
  }

  useEffect(() => {
    fetchData();
    const interval = setInterval(fetchData, 5000);
    return () => clearInterval(interval);
  }, [serverId]);

  if (!data) return <div className="text-neutral-500">Loading...</div>;

  const uptime = data.startedAt && data.status === "running"
    ? formatUptime(new Date(data.startedAt))
    : "\u2014";

  function handleCopy() {
    const text = `connect ${data!.serverIp}:${data!.port}`;
    try {
      const textarea = document.createElement("textarea");
      textarea.value = text;
      textarea.style.position = "fixed";
      textarea.style.opacity = "0";
      document.body.appendChild(textarea);
      textarea.select();
      document.execCommand("copy");
      document.body.removeChild(textarea);
    } catch {}
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  }

  return (
    <motion.div
      className="space-y-6"
      initial={{ opacity: 0, x: 16 }}
      animate={{ opacity: 1, x: 0 }}
      transition={{ type: "spring", stiffness: 300, damping: 28, delay: 0.1 }}
    >
      <div>
        <h2 className="font-semibold text-lg mb-1">{data.name}</h2>
        <div className="flex items-center gap-2">
          {data.status === "running" ? (
            <motion.span
              className="w-2 h-2 rounded-full bg-emerald-500"
              animate={{ opacity: [1, 0.4, 1] }}
              transition={{ duration: 1.5, repeat: Infinity, ease: "easeInOut" }}
            />
          ) : (
            <span className="w-2 h-2 rounded-full bg-neutral-500" />
          )}
          <span className="text-sm text-neutral-400 capitalize">{data.status}</span>
        </div>
      </div>

      <div className="space-y-3 text-sm">
        <div className="flex justify-between">
          <span className="text-neutral-500">Port</span>
          <span>{data.port}</span>
        </div>
        <div className="flex justify-between">
          <span className="text-neutral-500">Map</span>
          <span>{data.map}</span>
        </div>
        <div className="flex justify-between">
          <span className="text-neutral-500">Players</span>
          <span>{data.players ?? 0} / {data.maxPlayers ?? "?"}</span>
        </div>
        <div className="flex justify-between">
          <span className="text-neutral-500">Uptime</span>
          <span>{uptime}</span>
        </div>
        {data.stats && (
          <>
            <div className="flex justify-between">
              <span className="text-neutral-500">CPU</span>
              <span>{data.stats.cpuPercent}%</span>
            </div>
            <div className="flex justify-between">
              <span className="text-neutral-500">Memory</span>
              <span>{data.stats.memoryMb} MB</span>
            </div>
          </>
        )}
        {data.container_id && (
          <div className="flex justify-between">
            <span className="text-neutral-500">Container</span>
            <span className="font-mono text-xs">{String(data.container_id).slice(0, 12)}</span>
          </div>
        )}
      </div>

      <div className="space-y-2">
        {data.status === "sleeping" ? (
          <motion.button
            whileTap={{ scale: 0.97 }}
            onClick={() => onAction("wake")}
            className="cursor-pointer w-full flex items-center justify-center gap-1.5 py-2 text-sm bg-indigo-600 hover:bg-indigo-700 rounded font-medium transition-colors"
          >
            <Play size={14} fill="currentColor" /> Wake Up
          </motion.button>
        ) : data.status === "waking" ? (
          <div className="w-full flex items-center justify-center gap-1.5 py-2 text-sm text-amber-400">
            <Loader2 size={14} className="animate-spin" /> Waking up...
          </div>
        ) : data.status !== "running" ? (
          <motion.button
            whileTap={{ scale: 0.97 }}
            onClick={() => onAction("start")}
            className="cursor-pointer w-full flex items-center justify-center gap-1.5 py-2 text-sm bg-emerald-600 hover:bg-emerald-700 rounded font-medium transition-colors"
          >
            <Play size={14} fill="currentColor" /> Start
          </motion.button>
        ) : (
          <>
            <motion.button
              whileTap={{ scale: 0.97 }}
              onClick={() => onAction("stop")}
              className="cursor-pointer w-full flex items-center justify-center gap-1.5 py-2 text-sm bg-[#eb3449]/80 hover:bg-[#eb3449] text-white rounded font-medium transition-colors"
            >
              <CircleStop size={14} fill="currentColor" /> Stop
            </motion.button>
            <motion.button
              whileTap={{ scale: 0.97 }}
              onClick={() => onAction("restart")}
              className="cursor-pointer w-full flex items-center justify-center gap-1.5 py-2 text-sm bg-amber-600 hover:bg-amber-700 rounded font-medium transition-colors"
            >
              <RotateCcw size={14} /> Restart
            </motion.button>
          </>
        )}
        <Link href={`/servers/${serverId}/settings`} className="cursor-pointer flex items-center justify-center gap-1.5 w-full py-2 text-sm text-center bg-neutral-800 hover:bg-neutral-700 rounded font-medium transition-colors">
          <Settings size={14} /> Settings
        </Link>
      </div>

      <div className="space-y-2">
        <p className="text-xs text-neutral-500 uppercase tracking-wide">Connect</p>
        <div className="flex gap-2">
          <motion.button
            whileTap={{ scale: 0.97 }}
            onClick={handleCopy}
            className="cursor-pointer flex-1 flex items-center justify-center gap-1.5 py-2 text-sm bg-neutral-800 hover:bg-neutral-700 rounded font-medium transition-colors"
          >
            <Copy size={14} /> {copied ? <span className="text-emerald-400">Copied!</span> : "Copy IP"}
          </motion.button>
          <a
            href={`steam://connect/${data.serverIp}:${data.port}`}
            className="cursor-pointer flex items-center justify-center gap-1.5 px-3 py-2 text-sm bg-neutral-800 hover:bg-neutral-700 rounded font-medium transition-colors"
          >
            <ExternalLink size={14} /> Steam
          </a>
        </div>
      </div>
    </motion.div>
  );
}

function formatUptime(started: Date): string {
  const diff = Math.floor((Date.now() - started.getTime()) / 1000);
  const hours = Math.floor(diff / 3600);
  const mins = Math.floor((diff % 3600) / 60);
  const secs = diff % 60;
  if (hours > 0) return `${hours}h ${mins}m`;
  if (mins > 0) return `${mins}m ${secs}s`;
  return `${secs}s`;
}
