"use client";

import { useState } from "react";
import LoginGate from "@/components/LoginGate";
import AddTorrentForm from "@/components/AddTorrentForm";
import TorrentCard from "@/components/TorrentCard";
import { clearTokens } from "@/lib/api";
import { useTorrents } from "@/lib/useTorrents";

export default function Home() {
  const [authed, setAuthed] = useState(false);
  // Forces re-mount of the SSE hook when the user signs out and back in.
  const [sessionKey, setSessionKey] = useState(0);

  const handleAuthError = () => {
    clearTokens();
    setAuthed(false);
  };

  const { torrents, connected, refresh } = useTorrents(
    authed,
    handleAuthError,
    );

  if (!authed) {
    return (
      <LoginGate
        key={sessionKey}
        onReady={() => {
          setAuthed(true);
          setSessionKey((k) => k + 1);
        }}
      />
    );
  }

  const logout = () => {
    clearTokens();
    setAuthed(false);
  };

  return (
    <main className="mx-auto max-w-3xl px-4 py-8">
      <header className="mb-8 flex items-center justify-between">
        <h1 className="text-2xl font-bold text-white">GoFlix</h1>
        <div className="flex items-center gap-4">
          <span
            className={`flex items-center gap-1.5 text-xs ${
              connected ? "text-emerald-400" : "text-gray-500"
            }`}
          >
            <span
              className={`h-2 w-2 rounded-full ${
                connected ? "bg-emerald-400" : "bg-gray-600"
              }`}
            />
            {connected ? "Live" : "Polling"}
          </span>
          <button
            onClick={logout}
            className="text-xs text-gray-400 underline-offset-2 hover:text-gray-200 hover:underline"
          >
            Sign out
          </button>
        </div>
      </header>

      <div className="mb-8">
        <AddTorrentForm onAdded={refresh} />
      </div>

      <section>
        <div className="mb-3 flex items-center justify-between">
          <h2 className="text-sm font-medium uppercase tracking-wide text-gray-500">
            Torrents
          </h2>
          <span className="text-xs text-gray-500">{torrents.length} total</span>
        </div>

        {torrents.length === 0 ? (
          <div className="rounded-2xl border border-dashed border-white/10 p-12 text-center text-gray-500">
            No torrents yet. Add one above.
          </div>
        ) : (
          <div className="grid gap-4">
            {torrents.map((t) => (
              <TorrentCard key={t.id} torrent={t} onChanged={refresh} />
            ))}
          </div>
        )}
      </section>
    </main>
  );
}
