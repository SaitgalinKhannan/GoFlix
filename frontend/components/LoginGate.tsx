"use client";

import { useEffect, useState } from "react";
import { ArrowRight, LoaderCircle, LockKeyhole, Radio, UserRound } from "lucide-react";
import DesktopDataDirectory from "@/components/DesktopDataDirectory";
import { clearTokens, hasTokens, listTorrents, login } from "@/lib/api";

export default function LoginGate({ onReady }: { onReady: () => void }) {
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [checking, setChecking] = useState(true);

  useEffect(() => {
    let active = true;
    void (async () => {
      if (!hasTokens()) {
        if (active) setChecking(false);
        return;
      }
      try {
        await listTorrents();
        if (active) onReady();
      } catch {
        clearTokens();
        if (active) setChecking(false);
      }
    })();
    return () => {
      active = false;
    };
  }, [onReady]);

  const submit = async (event: React.FormEvent) => {
    event.preventDefault();
    setError(null);
    setBusy(true);
    try {
      await login(username, password);
      onReady();
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : "Login failed");
    } finally {
      setBusy(false);
    }
  };

  if (checking) {
    return (
      <div className="app-shell flex min-h-screen items-center justify-center">
        <div className="flex items-center gap-3 text-sm text-slate-400">
          <LoaderCircle className="h-4 w-4 animate-spin text-mint" />
          Restoring session
        </div>
      </div>
    );
  }

  return (
    <main className="app-shell flex min-h-screen items-center justify-center px-4 py-10">
      <div className="w-full max-w-sm">
        <div className="mb-7 flex items-center justify-center gap-3">
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

        <DesktopDataDirectory />

        <form onSubmit={submit} className="rounded-lg border border-white/[0.1] bg-panel p-5 shadow-[0_24px_80px_rgba(0,0,0,0.42)] sm:p-6">
          <div className="mb-6 flex items-center justify-between border-b border-white/[0.07] pb-5">
            <div>
              <h2 className="text-base font-semibold text-slate-100">Welcome back</h2>
              <p className="mt-1 text-xs text-slate-500">Sign in to manage your downloads</p>
            </div>
            <div className="flex h-8 w-8 items-center justify-center rounded-md border border-mint/20 bg-mint/[0.06] text-mint">
              <Radio className="h-4 w-4" />
            </div>
          </div>

          <label className="block text-[11px] font-semibold text-slate-400" htmlFor="username">Username</label>
          <div className="group relative mt-2">
            <UserRound className="pointer-events-none absolute left-3.5 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-600 group-focus-within:text-mint" />
            <input
              id="username"
              autoFocus
              autoComplete="username"
              type="text"
              value={username}
              onChange={(event) => setUsername(event.target.value)}
              placeholder="Enter username"
              className="h-11 w-full rounded-md border border-white/[0.09] bg-[#0a0f11] pl-10 pr-3 text-sm text-slate-100 outline-none transition placeholder:text-slate-600 focus:border-mint/60"
            />
          </div>

          <label className="mt-4 block text-[11px] font-semibold text-slate-400" htmlFor="password">Password</label>
          <div className="group relative mt-2">
            <LockKeyhole className="pointer-events-none absolute left-3.5 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-600 group-focus-within:text-mint" />
            <input
              id="password"
              autoComplete="current-password"
              type="password"
              value={password}
              onChange={(event) => setPassword(event.target.value)}
              placeholder="Enter password"
              className="h-11 w-full rounded-md border border-white/[0.09] bg-[#0a0f11] pl-10 pr-3 text-sm text-slate-100 outline-none transition placeholder:text-slate-600 focus:border-mint/60"
            />
          </div>

          {error && (
            <p className="mt-4 rounded-md border border-red-400/20 bg-red-400/[0.06] px-3 py-2 text-xs text-red-300">{error}</p>
          )}

          <button
            type="submit"
            disabled={busy || !username || !password}
            className="mt-5 inline-flex h-11 w-full items-center justify-center gap-2 rounded-md bg-mint text-sm font-semibold text-[#04251d] transition hover:bg-[#58ebc6] disabled:cursor-not-allowed disabled:opacity-40"
          >
            {busy ? <LoaderCircle className="h-4 w-4 animate-spin" /> : <ArrowRight className="h-4 w-4" />}
            {busy ? "Signing in" : "Sign in"}
          </button>
        </form>

        <p className="mt-5 text-center text-[10px] text-slate-600">Private access · encrypted session</p>
      </div>
    </main>
  );
}
