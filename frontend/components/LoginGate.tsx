"use client";

import { useState } from "react";
import { clearTokens, hasTokens, listTorrents, login } from "@/lib/api";

export default function LoginGate({
  onReady,
}: {
  onReady: () => void;
}) {
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [checking, setChecking] = useState(true);

  // If a valid token pair is already stored, skip straight in.
  if (checking) {
    void (async () => {
      if (!hasTokens()) {
        setChecking(false);
        return;
      }
      try {
        await listTorrents();
        onReady();
      } catch {
        clearTokens();
        setChecking(false);
      }
    })();
    return (
      <div className="flex min-h-screen items-center justify-center">
        <p className="text-gray-400">Checking session…</p>
      </div>
    );
  }

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    setBusy(true);
    try {
      await login(username, password);
      onReady();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Login failed");
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="flex min-h-screen items-center justify-center px-4">
      <form
        onSubmit={submit}
        className="w-full max-w-sm rounded-2xl border border-white/10 bg-white/5 p-8 shadow-xl"
      >
        <h1 className="mb-1 text-2xl font-semibold text-white">GoFlix</h1>
        <p className="mb-6 text-sm text-gray-400">Sign in to continue.</p>

        <label className="mb-1 block text-xs text-gray-400">Username</label>
        <input
          autoFocus
          type="text"
          value={username}
          onChange={(e) => setUsername(e.target.value)}
          placeholder="username"
          className="mb-3 w-full rounded-lg border border-white/10 bg-black/40 px-3 py-2 text-white outline-none focus:border-brand"
        />

        <label className="mb-1 block text-xs text-gray-400">Password</label>
        <input
          type="password"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          placeholder="password"
          className="mb-3 w-full rounded-lg border border-white/10 bg-black/40 px-3 py-2 text-white outline-none focus:border-brand"
        />

        {error && <p className="mb-3 text-sm text-red-400">{error}</p>}
        <button
          type="submit"
          disabled={busy || !username || !password}
          className="w-full rounded-lg bg-brand px-4 py-2 font-medium text-white transition hover:bg-brand-dark disabled:opacity-50"
        >
          {busy ? "Signing in…" : "Sign in"}
        </button>
      </form>
    </div>
  );
}
