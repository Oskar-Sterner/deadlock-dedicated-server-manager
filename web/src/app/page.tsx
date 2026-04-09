"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { motion } from "framer-motion";
import { ServerCard } from "@/components/server-card";

interface ServerData {
  id: string;
  name: string;
  port: number;
  map: string;
  deadworks: number;
  status: string;
  stats: { cpuPercent: number; memoryMb: number; memoryLimitMb: number } | null;
  players: number | null;
  maxPlayers: number | null;
}

const containerVariants = {
  hidden: { opacity: 0 },
  visible: {
    opacity: 1,
    transition: {
      staggerChildren: 0.1,
    },
  },
};

const cardVariants = {
  hidden: { opacity: 0, y: 20 },
  visible: { opacity: 1, y: 0, transition: { type: "spring" as const, stiffness: 300, damping: 25 } },
};

export default function Dashboard() {
  const [servers, setServers] = useState<ServerData[]>([]);
  const [serverIp, setServerIp] = useState("");
  const [loading, setLoading] = useState(true);
  const router = useRouter();

  async function fetchServers() {
    const res = await fetch("/api/servers");
    if (res.status === 401) { router.push("/login"); return; }
    const data = await res.json();
    setServers(data.servers);
    if (data.serverIp) setServerIp(data.serverIp);
    setLoading(false);
  }

  useEffect(() => {
    fetchServers();
    const interval = setInterval(fetchServers, 5000);
    return () => clearInterval(interval);
  }, []);

  async function handleAction(id: string, action: string) {
    await fetch(`/api/servers/${id}`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ action }),
    });
    fetchServers();
  }

  if (loading) {
    return (
      <motion.div
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}
        className="text-neutral-500 text-center mt-20"
      >
        Loading...
      </motion.div>
    );
  }

  return (
    <motion.div
      initial={{ opacity: 0 }}
      animate={{ opacity: 1 }}
      transition={{ duration: 0.3 }}
    >
      {servers.length === 0 ? (
        <div className="text-center mt-20">
          <p className="text-neutral-500 mb-4">No servers yet</p>
          <motion.button
            whileTap={{ scale: 0.95 }}
            onClick={() => router.push("/servers/new")}
            className="cursor-pointer px-4 py-2 bg-gradient-to-r from-[#eb3449] to-[#c42a3b] hover:from-[#f05c6a] hover:to-[#eb3449] rounded font-medium transition-all"
          >
            Create your first server
          </motion.button>
        </div>
      ) : (
        <motion.div
          className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4"
          variants={containerVariants}
          initial="hidden"
          animate="visible"
        >
          {servers.map((s) => (
            <motion.div key={s.id} variants={cardVariants}>
              <ServerCard {...s} serverIp={serverIp} onAction={handleAction} />
            </motion.div>
          ))}
        </motion.div>
      )}
    </motion.div>
  );
}
