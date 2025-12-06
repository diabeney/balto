import type { Metadata } from "next";
import { DM_Mono } from "next/font/google";
import "./globals.css";

const googleSansCode = DM_Mono({
weight: "400",
  variable: "--font-mono",
  subsets: ["latin"],
});

export const metadata: Metadata = {
  title: "Balto Dashboard",
  description: "Load balancer and reverse proxy management dashboard",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en" className="dark">
      <body
        className={`${googleSansCode.variable} font-mono antialiased bg-zinc-950 text-zinc-100`}
      >
        {children}
      </body>
    </html>
  );
}
