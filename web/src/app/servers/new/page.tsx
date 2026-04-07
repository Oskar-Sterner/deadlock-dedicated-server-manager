"use client";

import { useState, useEffect } from "react";
import { useRouter } from "next/navigation";
import { motion } from "framer-motion";

export default function NewServerPage() {
  const [form, setForm] = useState({
    name: "",
    port: 27015,
    map: "dl_streets",
    password: "",
    steam_login: "",
    steam_pass: "",
    steam_2fa: "",
  });
  const [error, setError] = useState("");
  const [creating, setCreating] = useState(false);
  const router = useRouter();

  useEffect(() => {
    fetch("/api/servers").then(async (r) => {
      if (r.status === 401) { router.push("/login"); return; }
      const data = await r.json();
      setForm((f) => ({ ...f, port: data.nextPort }));
    });
  }, []);

  function update(field: string, value: string | number) {
    setForm((f) => ({ ...f, [field]: value }));
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    setCreating(true);

    const res = await fetch("/api/servers", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(form),
    });

    if (res.ok) {
      const data = await res.json();
      router.push(`/servers/${data.server.id}`);
    } else {
      const data = await res.json();
      setError(data.error || "Failed to create server");
      setCreating(false);
    }
  }

  const inputClass = "w-full px-4 py-2 bg-neutral-800 border border-neutral-700 rounded text-neutral-100 placeholder-neutral-500 focus:outline-none focus:border-[#eb3449]";
  const labelClass = "block text-sm font-medium text-neutral-300 mb-1";

  return (
    <motion.div
      className="max-w-lg mx-auto"
      initial={{ opacity: 0, y: 24 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ type: "spring", stiffness: 300, damping: 28 }}
    >
      <h1 className="text-2xl font-bold mb-6">Create Server</h1>

      <form onSubmit={handleSubmit} className="space-y-4">
        <div>
          <label className={labelClass}>Server Name</label>
          <input className={inputClass} value={form.name} onChange={(e) => update("name", e.target.value)} placeholder="My Deadlock Server" required />
        </div>

        <div className="grid grid-cols-2 gap-4">
          <div>
            <label className={labelClass}>Port</label>
            <input className={inputClass} type="number" value={form.port} onChange={(e) => update("port", parseInt(e.target.value))} required />
          </div>
          <div>
            <label className={labelClass}>Map</label>
            <select className={inputClass} value={form.map} onChange={(e) => update("map", e.target.value)}>
              <option value="dl_streets">dl_streets</option>
              <option value="dl_midtown">dl_midtown</option>
              <option value="dl_hideout">dl_hideout</option>
            </select>
          </div>
        </div>

        <div>
          <label className={labelClass}>Server Password (optional)</label>
          <input className={inputClass} value={form.password} onChange={(e) => update("password", e.target.value)} placeholder="Leave empty for no password" />
        </div>

        <hr className="border-neutral-800" />

        <div>
          <label className={labelClass}>Steam Login</label>
          <input className={inputClass} value={form.steam_login} onChange={(e) => update("steam_login", e.target.value)} required />
        </div>

        <div>
          <label className={labelClass}>Steam Password</label>
          <input className={inputClass} type="password" value={form.steam_pass} onChange={(e) => update("steam_pass", e.target.value)} required />
        </div>

        <div>
          <label className={labelClass}>Steam 2FA Code</label>
          <input className={inputClass} value={form.steam_2fa} onChange={(e) => update("steam_2fa", e.target.value)} placeholder="From Steam Guard app" />
        </div>

        {error && <p className="text-[#f05c6a] text-sm">{error}</p>}

        <motion.button
          whileTap={{ scale: 0.97 }}
          type="submit"
          disabled={creating}
          className="cursor-pointer w-full py-2.5 bg-gradient-to-r from-[#eb3449] to-[#c42a3b] hover:from-[#f05c6a] hover:to-[#eb3449] disabled:bg-none disabled:bg-neutral-700 disabled:text-neutral-400 rounded font-medium transition-all"
        >
          {creating ? "Creating..." : "Create Server"}
        </motion.button>
      </form>
    </motion.div>
  );
}
