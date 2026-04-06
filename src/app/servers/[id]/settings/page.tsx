"use client";

import { use, useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { motion } from "framer-motion";

export default function SettingsPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = use(params);
  const router = useRouter();
  const [form, setForm] = useState({
    name: "", port: 27015, map: "dl_streets", password: "",
    steam_login: "", steam_pass: "", steam_2fa: "", skip_update: 1,
  });
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const [deleteConfirm, setDeleteConfirm] = useState(false);

  useEffect(() => {
    fetch(`/api/servers/${id}`).then(async (r) => {
      if (r.status === 401) { router.push("/login"); return; }
      if (!r.ok) { router.push("/"); return; }
      const data = await r.json();
      setForm({
        name: data.name, port: data.port, map: data.map, password: data.password,
        steam_login: data.steam_login, steam_pass: "", steam_2fa: data.steam_2fa,
        skip_update: data.skip_update,
      });
    });
  }, [id]);

  function update(field: string, value: string | number) {
    setForm((f) => ({ ...f, [field]: value }));
  }

  async function handleSave(e: React.FormEvent) {
    e.preventDefault();
    setSaving(true);
    setError("");

    const res = await fetch(`/api/servers/${id}`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ action: "update", ...form }),
    });

    if (res.ok) {
      router.push(`/servers/${id}`);
    } else {
      const data = await res.json();
      setError(data.error || "Failed to save");
      setSaving(false);
    }
  }

  async function handleDelete() {
    await fetch(`/api/servers/${id}`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ action: "delete", deleteFiles: true }),
    });
    router.push("/");
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
      <h1 className="text-2xl font-bold mb-6">Server Settings</h1>

      <form onSubmit={handleSave} className="space-y-4">
        <div>
          <label className={labelClass}>Server Name</label>
          <input className={inputClass} value={form.name} onChange={(e) => update("name", e.target.value)} required />
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
          <label className={labelClass}>Server Password</label>
          <input className={inputClass} value={form.password} onChange={(e) => update("password", e.target.value)} placeholder="Leave empty for no password" />
        </div>

        <div className="flex items-center gap-3">
          <input type="checkbox" id="skip" checked={form.skip_update === 1} onChange={(e) => update("skip_update", e.target.checked ? 1 : 0)} className="cursor-pointer rounded" />
          <label htmlFor="skip" className="cursor-pointer text-sm text-neutral-300">Skip SteamCMD update on restart</label>
        </div>

        <hr className="border-neutral-800" />

        <div>
          <label className={labelClass}>Steam Login</label>
          <input className={inputClass} value={form.steam_login} onChange={(e) => update("steam_login", e.target.value)} required />
        </div>
        <div>
          <label className={labelClass}>Steam Password</label>
          <input className={inputClass} type="password" value={form.steam_pass} onChange={(e) => update("steam_pass", e.target.value)} placeholder="Enter to change" />
        </div>
        <div>
          <label className={labelClass}>Steam 2FA Code</label>
          <input className={inputClass} value={form.steam_2fa} onChange={(e) => update("steam_2fa", e.target.value)} placeholder="From Steam Guard app" />
        </div>

        {error && <p className="text-[#f05c6a] text-sm">{error}</p>}

        <motion.button
          whileTap={{ scale: 0.97 }}
          type="submit"
          disabled={saving}
          className="cursor-pointer w-full py-2.5 bg-gradient-to-r from-[#eb3449] to-[#c42a3b] hover:from-[#f05c6a] hover:to-[#eb3449] disabled:bg-none disabled:bg-neutral-700 rounded font-medium transition-all"
        >
          {saving ? "Saving..." : "Save & Restart Server"}
        </motion.button>
      </form>

      <div className="mt-8 pt-6 border-t border-neutral-800">
        {!deleteConfirm ? (
          <motion.button
            whileTap={{ scale: 0.97 }}
            onClick={() => setDeleteConfirm(true)}
            className="cursor-pointer w-full py-2.5 bg-[#eb3449]/10 hover:bg-[#eb3449]/20 text-[#f05c6a] rounded font-medium transition-colors"
          >
            Delete Server
          </motion.button>
        ) : (
          <div className="space-y-2">
            <p className="text-sm text-[#f05c6a]">This will stop the container and delete all game files. Are you sure?</p>
            <div className="flex gap-2">
              <motion.button
                whileTap={{ scale: 0.97 }}
                onClick={handleDelete}
                className="cursor-pointer flex-1 py-2 bg-gradient-to-r from-[#eb3449] to-[#c42a3b] hover:from-[#f05c6a] hover:to-[#eb3449] rounded font-medium transition-all"
              >
                Yes, Delete
              </motion.button>
              <motion.button
                whileTap={{ scale: 0.97 }}
                onClick={() => setDeleteConfirm(false)}
                className="cursor-pointer flex-1 py-2 bg-neutral-800 hover:bg-neutral-700 rounded font-medium transition-colors"
              >
                Cancel
              </motion.button>
            </div>
          </div>
        )}
      </div>
    </motion.div>
  );
}
