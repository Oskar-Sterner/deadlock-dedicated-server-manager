"use client";

import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { motion } from "framer-motion";

const letters = ["D", "D", "S", "M"];

export function Nav() {
  const pathname = usePathname();
  const router = useRouter();

  if (pathname === "/login") return null;

  async function logout() {
    await fetch("/api/auth", { method: "DELETE" });
    router.push("/login");
  }

  return (
    <nav className="bg-neutral-900/50 backdrop-blur-sm sticky top-0 z-50"
      style={{
        borderBottom: "1px solid",
        borderImageSource: "linear-gradient(to right, transparent, #eb344960, transparent)",
        borderImageSlice: 1,
      }}
    >
      <div className="max-w-6xl mx-auto px-6 h-14 flex items-center justify-between">
        <Link href="/" className="cursor-pointer" aria-label="Home">
          <div className="font-nabla text-2xl tracking-wider flex">
            {letters.map((letter, i) => (
              <motion.span
                key={i}
                initial={{ opacity: 0, x: 20 }}
                animate={{ opacity: 1, x: 0 }}
                transition={{
                  type: "spring",
                  stiffness: 300,
                  damping: 20,
                  delay: (letters.length - 1 - i) * 0.1,
                }}
                className="inline-block"
              >
                {letter}
              </motion.span>
            ))}
          </div>
        </Link>

        <div className="flex items-center gap-4">
          <Link
            href="/servers/new"
            className="cursor-pointer px-3 py-1.5 text-sm bg-gradient-to-r from-[#eb3449] to-[#c42a3b] hover:from-[#f05c6a] hover:to-[#eb3449] rounded font-medium transition-all"
          >
            + New Server
          </Link>
          <button
            onClick={logout}
            className="cursor-pointer text-sm text-neutral-500 hover:text-neutral-300 transition-colors"
          >
            Logout
          </button>
        </div>
      </div>
    </nav>
  );
}
