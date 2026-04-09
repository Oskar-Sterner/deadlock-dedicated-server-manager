"use client";

import { useState } from "react";
import Link from "next/link";
import { motion, AnimatePresence } from "framer-motion";
import { Play, RotateCcw, Terminal, Settings, Copy, ExternalLink, CircleStop, Moon, Loader2 } from "lucide-react";

interface ServerCardProps {
  id: string;
  name: string;
  port: number;
  map: string;
  deadworks: number;
  status: string;
  stats: { cpuPercent: number; memoryMb: number; memoryLimitMb: number } | null;
  onAction: (id: string, action: string) => void;
  players?: number | null;
  maxPlayers?: number | null;
  serverIp?: string;
}

const STATUS_COLORS: Record<string, string> = {
  running: "bg-emerald-500",
  exited: "bg-neutral-500",
  created: "bg-amber-500",
  restarting: "bg-amber-500",
  sleeping: "bg-indigo-500",
  waking: "bg-amber-400",
  dead: "bg-[#eb3449]",
  unknown: "bg-neutral-600",
};

function copyToClipboard(text: string): boolean {
  // Fallback for non-HTTPS contexts
  try {
    const textarea = document.createElement("textarea");
    textarea.value = text;
    textarea.style.position = "fixed";
    textarea.style.opacity = "0";
    document.body.appendChild(textarea);
    textarea.select();
    document.execCommand("copy");
    document.body.removeChild(textarea);
    return true;
  } catch {
    return false;
  }
}

export function ServerCard({ id, name, port, map, deadworks, status, stats, onAction, players, maxPlayers, serverIp = "" }: ServerCardProps) {
  const [copied, setCopied] = useState(false);

  function handleCopy() {
    const connectStr = `connect ${serverIp}:${port}`;
    copyToClipboard(connectStr);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  }

  return (
    <motion.div
      className="bg-neutral-900 border border-neutral-800 rounded-lg p-5 transition-colors relative overflow-hidden group"
      whileHover={{ scale: 1.02 }}
      transition={{ type: "spring", stiffness: 400, damping: 25 }}
    >
      <div className="pointer-events-none absolute inset-0 rounded-lg opacity-0 group-hover:opacity-100 transition-opacity duration-300"
        style={{
          background: "linear-gradient(135deg, #eb344910 0%, transparent 60%)",
          border: "1px solid #eb344930",
        }}
      />

      <div className="flex items-start justify-between mb-3">
        <Link href={`/servers/${id}`} className="cursor-pointer hover:text-[#f05c6a] transition-colors">
          <h3 className="font-semibold text-lg">{name}</h3>
        </Link>
        <div className="flex items-center gap-2">
          {status === "running" ? (
            <motion.span
              className={`w-2.5 h-2.5 rounded-full ${STATUS_COLORS[status] || STATUS_COLORS.unknown}`}
              animate={{ opacity: [1, 0.4, 1] }}
              transition={{ duration: 1.5, repeat: Infinity, ease: "easeInOut" }}
            />
          ) : (
            <span className={`w-2.5 h-2.5 rounded-full ${STATUS_COLORS[status] || STATUS_COLORS.unknown}`} />
          )}
          <span className="text-sm text-neutral-400 capitalize">{status}</span>
        </div>
      </div>

      <div className="text-sm text-neutral-400 space-y-1 mb-4">
        <p>Port: <span className="text-neutral-200">{port}</span></p>
        <p>Map: <span className="text-neutral-200">{map}</span></p>
        <p>Deadworks: <span className={deadworks === 1 ? "text-emerald-400" : "text-red-400"}>{deadworks === 1 ? "Yes" : "No"}</span></p>
        <p>Players: <span className="text-neutral-200">{players ?? 0} / {maxPlayers ?? "?"}</span></p>
      </div>

      {stats && status === "running" && (
        <div className="space-y-2 mb-4">
          <div>
            <div className="flex justify-between text-xs text-neutral-500 mb-1">
              <span>CPU</span>
              <span>{stats.cpuPercent}%</span>
            </div>
            <div className="h-1.5 bg-neutral-800 rounded-full overflow-hidden">
              <div className="h-full bg-[#eb3449] rounded-full transition-all" style={{ width: `${Math.min(stats.cpuPercent, 100)}%` }} />
            </div>
          </div>
          <div>
            <div className="flex justify-between text-xs text-neutral-500 mb-1">
              <span>Memory</span>
              <span>{stats.memoryMb} MB</span>
            </div>
            <div className="h-1.5 bg-neutral-800 rounded-full overflow-hidden">
              <div className="h-full bg-[#eb3449] rounded-full transition-all" style={{ width: `${stats.memoryLimitMb > 0 ? (stats.memoryMb / stats.memoryLimitMb) * 100 : 0}%` }} />
            </div>
          </div>
        </div>
      )}

      {/* Action buttons */}
      <div className="flex items-center gap-2 mb-3">
        {status === "sleeping" ? (
          <motion.button
            whileTap={{ scale: 0.95 }}
            onClick={() => onAction(id, "wake")}
            className="cursor-pointer flex items-center gap-1.5 px-3 py-1.5 text-xs bg-indigo-600 hover:bg-indigo-700 rounded font-medium transition-colors"
          >
            <Play size={13} fill="currentColor" /> Wake Up
          </motion.button>
        ) : status === "waking" ? (
          <span className="flex items-center gap-1.5 px-3 py-1.5 text-xs text-amber-400">
            <Loader2 size={13} className="animate-spin" /> Waking up...
          </span>
        ) : status !== "running" ? (
          <motion.button
            whileTap={{ scale: 0.95 }}
            onClick={() => onAction(id, "start")}
            className="cursor-pointer flex items-center gap-1.5 px-3 py-1.5 text-xs bg-emerald-600 hover:bg-emerald-700 rounded font-medium transition-colors"
          >
            <Play size={13} fill="currentColor" /> Start
          </motion.button>
        ) : (
          <>
            <motion.button
              whileTap={{ scale: 0.95 }}
              onClick={() => onAction(id, "stop")}
              className="cursor-pointer flex items-center gap-1.5 px-3 py-1.5 text-xs bg-[#eb3449]/80 hover:bg-[#eb3449] text-white rounded font-medium transition-colors"
            >
              <CircleStop size={13} fill="currentColor" /> Stop
            </motion.button>
            <motion.button
              whileTap={{ scale: 0.95 }}
              onClick={() => onAction(id, "restart")}
              className="cursor-pointer flex items-center gap-1.5 px-3 py-1.5 text-xs bg-amber-600 hover:bg-amber-700 rounded font-medium transition-colors"
            >
              <RotateCcw size={13} /> Restart
            </motion.button>
          </>
        )}
      </div>

      {/* Navigation + connect row */}
      <div className="flex items-center gap-2">
        <Link href={`/servers/${id}`} className="cursor-pointer flex items-center gap-1.5 px-3 py-1.5 text-xs bg-neutral-800 hover:bg-neutral-700 rounded font-medium transition-colors">
          <Terminal size={13} /> Console
        </Link>
        <Link href={`/servers/${id}/settings`} className="cursor-pointer flex items-center gap-1.5 px-3 py-1.5 text-xs bg-neutral-800 hover:bg-neutral-700 rounded font-medium transition-colors">
          <Settings size={13} /> Settings
        </Link>
        <div className="ml-auto flex items-center gap-2 relative">
          <motion.button
            whileTap={{ scale: 0.95 }}
            onClick={handleCopy}
            className="cursor-pointer flex items-center gap-1.5 px-3 py-1.5 text-xs bg-neutral-800 hover:bg-neutral-700 rounded font-medium transition-colors relative"
          >
            <Copy size={13} /> Connect
            <AnimatePresence>
              {copied && (
                <motion.span
                  initial={{ opacity: 0, y: 4 }}
                  animate={{ opacity: 1, y: 0 }}
                  exit={{ opacity: 0, y: -4 }}
                  className="absolute -top-7 left-1/2 -translate-x-1/2 text-[10px] bg-emerald-600 text-white px-2 py-0.5 rounded whitespace-nowrap"
                >
                  Copied!
                </motion.span>
              )}
            </AnimatePresence>
          </motion.button>
          <a
            href={`steam://connect/${serverIp}:${port}`}
            className="cursor-pointer flex items-center justify-center p-1.5 text-xs bg-neutral-800 hover:bg-neutral-700 rounded transition-colors text-neutral-400 hover:text-neutral-200"
            title="Open in Steam"
          >
            <ExternalLink size={13} />
          </a>
        </div>
      </div>
    </motion.div>
  );
}
