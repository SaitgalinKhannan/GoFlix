import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: "GoFlix",
  description: "Download torrents and stream to Jellyfin",
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en">
      <body className="min-h-screen antialiased">
        {children}
      </body>
    </html>
  );
}
