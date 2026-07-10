"use client";

import { useCallback, useMemo, useState } from "react";
import {
  Activity,
  ArrowUpFromLine,
  CircleCheckBig,
  Download,
  Layers3,
  LogOut,
  Radio,
} from "lucide-react";
import LoginGate from "@/components/LoginGate";
import AddTorrentForm from "@/components/AddTorrentForm";
import TorrentCard from "@/components/TorrentCard";
import { clearTokens } from "@/lib/api";
import { formatSpeed } from "@/lib/format";
import { useTorrents } from "@/lib/useTorrents";

export default function Home() {
  const [authed, setAuthed] = useState(false);
  const [sessionKey, setSessionKey] = useState(0);

  const handleAuthError = useCallback(() => {
    clearTokens();
    setAuthed(false);
  }, []);

  const handleReady = useCallback(() => {
    setAuthed(true);
    setSessionKey((key) => key + 1);
  }, []);

  const { torrents, connected, refresh } = useTorrents(authed, handleAuthError);

  const stats = useMemo(() => {
    const active = torrents.filter((torrent) =>
      ["downloading", "checking", "metadata"].includes(torrent.status),
    ).length;
    const complete = torrents.filter((torrent) =>
      ["seeding", "completed"].includes(torrent.status),
    ).length;
    const speed = torrents.reduce((sum, torrent) => sum + torrent.downloadSpeed, 0);
    const uploadSpeed = torrents.reduce((sum, torrent) => sum + torrent.uploadSpeed, 0);
    return { active, complete, speed, uploadSpeed };
  }, [torrents]);

  if (!authed) {
    return <LoginGate key={sessionKey} onReady={handleReady} />;
  }

  const logout = () => {
    clearTokens();
    setAuthed(false);
  };

  return (
    <main className="app-shell min-h-screen">
      <div className="mx-auto w-full max-w-6xl px-4 pb-16 pt-5 sm:px-6 lg:px-8">
        <header className="mb-8 flex min-h-14 items-center justify-between border-b border-white/[0.07] pb-5">
          <div className="flex items-center gap-3">
            <div className="brand-mark" aria-hidden="true">
              <span />
              <span />
              <span />
            </div>
            <div>
              <h1 className="text-xl font-semibold text-white">GoFlix</h1>
              <p className="text-[11px] text-slate-500">Torrent control center</p>
            </div>
          </div>

          <div className="flex items-center gap-2 sm:gap-4">
            <div className="hidden items-center gap-2 border-r border-white/10 pr-4 sm:flex">
              <Radio className={`h-3.5 w-3.5 ${connected ? "text-mint" : "text-amber-400"}`} />
              <span className="text-xs text-slate-400">{connected ? "Live sync" : "Polling"}</span>
            </div>
            <button onClick={logout} className="icon-button" title="Sign out" aria-label="Sign out">
              <LogOut className="h-4 w-4" />
            </button>
          </div>
        </header>

        <section className="mb-6 grid grid-cols-2 gap-px overflow-hidden rounded-lg border border-white/[0.08] bg-white/[0.08] sm:grid-cols-5">
          <Stat icon={Layers3} label="Torrents" value={String(torrents.length)} />
          <Stat icon={Activity} label="Active" value={String(stats.active)} accent />
          <Stat icon={CircleCheckBig} label="Ready" value={String(stats.complete)} />
          <Stat icon={Download} label="Down speed" value={formatSpeed(stats.speed)} />
          <Stat icon={ArrowUpFromLine} label="Up speed" value={formatSpeed(stats.uploadSpeed)} className="col-span-2 sm:col-span-1" />
        </section>

        <AddTorrentForm onAdded={refresh} />

        <section className="mt-9">
          <div className="mb-4 flex items-end justify-between gap-4">
            <div>
              <p className="section-kicker">Library</p>
              <h2 className="mt-1 text-lg font-semibold text-slate-100">Your torrents</h2>
            </div>
            <span className="text-xs tabular-nums text-slate-500">
              {torrents.length} {torrents.length === 1 ? "item" : "items"}
            </span>
          </div>

          {torrents.length === 0 ? (
            <div className="empty-state">
              <Download className="h-6 w-6 text-mint" />
              <h3 className="mt-4 font-medium text-slate-200">Nothing in the queue</h3>
              <p className="mt-1 text-sm text-slate-500">Add a magnet link or torrent file above.</p>
            </div>
          ) : (
            <div className="grid gap-3">
              {torrents.map((torrent) => (
                <TorrentCard key={torrent.id} torrent={torrent} onChanged={refresh} />
              ))}
            </div>
          )}
        </section>
      </div>
    </main>
  );
}

function Stat({
  icon: Icon,
  label,
  value,
  accent = false,
  className = "",
}: {
  icon: typeof Activity;
  label: string;
  value: string;
  accent?: boolean;
  className?: string;
}) {
  return (
    <div className={`flex min-h-20 items-center gap-3 bg-panel px-4 py-3 sm:px-5 ${className}`}>
      <Icon className={`h-4 w-4 shrink-0 ${accent ? "text-mint" : "text-slate-500"}`} />
      <div className="min-w-0">
        <p className="truncate text-[11px] text-slate-500">{label}</p>
        <p className={`mt-0.5 truncate text-sm font-semibold tabular-nums ${accent ? "text-mint" : "text-slate-100"}`}>
          {value}
        </p>
      </div>
    </div>
  );
}
