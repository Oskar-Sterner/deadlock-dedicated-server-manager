"use client";

import { use } from "react";
import { motion } from "framer-motion";
import { Console } from "@/components/console";
import { StatsSidebar } from "@/components/stats-sidebar";

export default function ServerPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = use(params);

  async function handleAction(action: string) {
    await fetch(`/api/servers/${id}`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ action }),
    });
  }

  return (
    <motion.div
      className="flex gap-6"
      initial={{ opacity: 0 }}
      animate={{ opacity: 1 }}
      transition={{ duration: 0.3 }}
    >
      <div className="flex-1 min-w-0">
        <Console serverId={id} />
      </div>
      <div className="w-64 shrink-0">
        <StatsSidebar serverId={id} onAction={handleAction} />
      </div>
    </motion.div>
  );
}
