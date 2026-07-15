"use client";

import {
  CircleAlert,
  ExternalLink,
  LoaderCircle,
  Power,
  RefreshCw,
  Server,
  Square,
} from "lucide-react";
import { useCallback, useEffect, useState } from "react";
import { setDesktopApiBase } from "@/lib/api";
import { desktopInvoke, DesktopStatus, isDesktop } from "@/lib/desktop";

type JellyfinAction = "start_jellyfin" | "stop_jellyfin" | "restart_jellyfin";

export default function DesktopServices({ compact = false }: { compact?: boolean }) {
  const [available, setAvailable] = useState(false);
  const [status, setStatus] = useState<DesktopStatus | null>(null);
  const [busy, setBusy] = useState<JellyfinAction | null>(null);
  const [error, setError] = useState<string | null>(null);

  const refresh = useCallback(async () => {
    try {
      const next = await desktopInvoke<DesktopStatus>("get_desktop_status");
      setDesktopApiBase(next.apiUrl);
      setStatus(next);
      setError(null);
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : String(caught));
    }
  }, []);

  useEffect(() => {
    if (!isDesktop()) return;
    setAvailable(true);
    void refresh();
    const timer = window.setInterval(() => void refresh(), 3000);
    return () => window.clearInterval(timer);
  }, [refresh]);

  const run = async (action: JellyfinAction) => {
    setBusy(action);
    setError(null);
    try {
      await desktopInvoke<void>(action);
      await refresh();
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : String(caught));
    } finally {
      setBusy(null);
    }
  };

  if (!available) return null;

  const jellyfinRunning = status?.jellyfin === "running";
  const backendRunning = status?.backend === "running";

  return (
    <section className={`rounded-lg border border-white/[0.09] bg-panel ${compact ? "mb-5 p-4" : "mb-6 px-4 py-3.5 sm:px-5"}`}>
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div className="flex min-w-0 items-center gap-3">
          <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-md border border-mint/20 bg-mint/[0.07] text-mint">
            <Server className="h-4 w-4" />
          </div>
          <div className="min-w-0">
            <p className="text-sm font-medium text-slate-200">Desktop services</p>
            <div className="mt-1 flex flex-wrap gap-x-3 gap-y-1 text-[11px] text-slate-500">
              <ServiceLabel label="Backend" running={backendRunning} loading={!status} />
              <ServiceLabel label="Jellyfin" running={jellyfinRunning} loading={!status} />
            </div>
          </div>
        </div>

        <div className="flex flex-wrap items-center gap-2">
          {jellyfinRunning ? (
            <>
              <a
                href={status?.jellyfinUrl}
                target="_blank"
                rel="noreferrer"
                className="icon-button"
                title="Open Jellyfin"
                aria-label="Open Jellyfin"
              >
                <ExternalLink className="h-3.5 w-3.5" />
              </a>
              <button
                type="button"
                onClick={() => void run("restart_jellyfin")}
                disabled={!!busy}
                className="icon-button"
                title="Restart Jellyfin"
                aria-label="Restart Jellyfin"
              >
                {busy === "restart_jellyfin" ? <LoaderCircle className="h-3.5 w-3.5 animate-spin" /> : <RefreshCw className="h-3.5 w-3.5" />}
              </button>
              <button
                type="button"
                onClick={() => void run("stop_jellyfin")}
                disabled={!!busy}
                className="inline-flex h-9 items-center justify-center gap-2 rounded-md border border-red-400/20 bg-red-400/[0.05] px-3 text-xs font-medium text-red-300 transition hover:bg-red-400/[0.1] disabled:opacity-45"
              >
                {busy === "stop_jellyfin" ? <LoaderCircle className="h-3.5 w-3.5 animate-spin" /> : <Square className="h-3.5 w-3.5" />}
                Stop Jellyfin
              </button>
            </>
          ) : (
            <button
              type="button"
              onClick={() => void run("start_jellyfin")}
              disabled={!!busy || !status}
              className="inline-flex h-9 items-center justify-center gap-2 rounded-md bg-mint px-3 text-xs font-semibold text-[#04251d] transition hover:bg-[#58ebc6] disabled:opacity-45"
            >
              {busy === "start_jellyfin" ? <LoaderCircle className="h-3.5 w-3.5 animate-spin" /> : <Power className="h-3.5 w-3.5" />}
              Start Jellyfin
            </button>
          )}
        </div>
      </div>

      {(error || status?.startupError) && (
        <p className="mt-3 flex items-start gap-2 rounded-md border border-red-400/20 bg-red-400/[0.05] px-3 py-2 text-xs text-red-300">
          <CircleAlert className="mt-0.5 h-3.5 w-3.5 shrink-0" />
          <span>{error || status?.startupError}</span>
        </p>
      )}
    </section>
  );
}

function ServiceLabel({ label, running, loading }: { label: string; running: boolean; loading: boolean }) {
  return (
    <span className="inline-flex items-center gap-1.5">
      <span className={`h-1.5 w-1.5 rounded-full ${loading ? "bg-slate-600" : running ? "bg-mint" : "bg-slate-600"}`} />
      {label}: {loading ? "checking" : running ? "running" : "stopped"}
    </span>
  );
}
