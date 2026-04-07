"use client";

import { useState, useEffect } from "react";
import { useRouter } from "next/navigation";
import { motion } from "framer-motion";

export default function LoginPage() {
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [isSetup, setIsSetup] = useState(false);
  const [loading, setLoading] = useState(true);
  const router = useRouter();

  useEffect(() => {
    fetch("/api/setup").then(async (r) => {
      const data = await r.json();
      setIsSetup(data.setup);
      setLoading(false);
    });
  }, []);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError("");

    const endpoint = isSetup ? "/api/setup" : "/api/auth";
    const res = await fetch(endpoint, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ password }),
    });

    if (res.ok) {
      router.push("/");
    } else {
      const data = await res.json();
      if (data.error === "Already configured") {
        setIsSetup(false);
        setError("Password already set. Please log in.");
      } else {
        setError(data.error || "Failed");
      }
    }
  }

  if (loading) {
    return (
      <div className="flex items-center justify-center min-h-screen">
        <div className="text-neutral-500">Loading...</div>
      </div>
    );
  }

  return (
    <div className="flex items-center justify-center min-h-screen">
      <motion.div
        initial={{ opacity: 0, y: 24 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ type: "spring", stiffness: 300, damping: 28 }}
        className="w-full max-w-sm bg-neutral-900 border border-neutral-700 rounded-lg p-8"
      >
        <h1 className="text-xl font-bold mb-1 text-center">DDSM</h1>
        <p className="text-neutral-400 text-sm text-center mb-6">
          {isSetup ? "Set your dashboard password" : "Enter your password"}
        </p>

        <form onSubmit={handleSubmit} className="space-y-4">
          <input
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            placeholder="Password"
            className="w-full px-4 py-2 bg-neutral-800 border border-neutral-700 rounded text-neutral-100 placeholder-neutral-500 focus:outline-none focus:border-[#eb3449]"
            autoFocus
          />

          {error && <p className="text-[#f05c6a] text-sm">{error}</p>}

          <motion.button
            whileTap={{ scale: 0.97 }}
            type="submit"
            className="cursor-pointer w-full py-2 bg-gradient-to-r from-[#eb3449] to-[#c42a3b] hover:from-[#f05c6a] hover:to-[#eb3449] rounded font-medium transition-all"
          >
            {isSetup ? "Set Password" : "Log In"}
          </motion.button>
        </form>

        {!isSetup && (
          <button
            onClick={() => setIsSetup(true)}
            className="cursor-pointer w-full mt-3 text-sm text-neutral-500 hover:text-neutral-300 transition-colors"
          >
            First time? Set up password
          </button>
        )}
      </motion.div>
    </div>
  );
}
