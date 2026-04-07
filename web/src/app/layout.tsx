import type { Metadata } from "next";
import { Nav } from "@/components/nav";
import "./globals.css";

export const metadata: Metadata = {
  title: "DDSM",
  description: "Manage Deadlock dedicated servers",
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en" className="dark">
      <head>
        <link
          href="https://fonts.googleapis.com/css2?family=Nabla&display=swap"
          rel="stylesheet"
        />
        <style dangerouslySetInnerHTML={{ __html: `
          @font-palette-values --custom {
            font-family: Nabla;
            override-colors: 0 #ff1438, 1 #5c0017, 2 #66001f, 3 #901d2f, 4 #ff142c, 5 #ff6b72, 6 #ff1438, 7 #ff6b8b, 8 #fe8ba2, 9 #ffffff;
          }
          .font-nabla {
            font-family: "Nabla", sans-serif;
            font-palette: --custom;
          }
        `}} />
      </head>
      <body className="min-h-screen bg-neutral-950 text-neutral-100 antialiased">
        <Nav />
        <main className="max-w-6xl mx-auto px-6 py-8">{children}</main>
      </body>
    </html>
  );
}
