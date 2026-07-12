"use client";

import { FolderCog, FolderOpen, LoaderCircle } from "lucide-react";
import { useEffect, useState } from "react";

declare global {
  interface Window {
    __TAURI__?: {
      core?: {
        invoke<T>(command: string, args?: Record<string, unknown>): Promise<T>;
      };
    };
  }
}

type DataDirectory = { path: string; projectDir: string };

function invoke<T>(command: string, args?: Record<string, unknown>): Promise<T> {
  const nativeInvoke = window.__TAURI__?.core?.invoke;
  if (!nativeInvoke) return Promise.reject(new Error("Desktop API is unavailable"));
  return nativeInvoke<T>(command, args);
}

export default function DesktopDataDirectory() {
  const [directory, setDirectory] = useState<DataDirectory | null>(null);
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState<string | null>(null);
  const [available, setAvailable] = useState(false);

  useEffect(() => {
    if (!window.__TAURI__?.core?.invoke) return;
    setAvailable(true);
    void invoke<DataDirectory>("get_data_dir")
      .then(setDirectory)
      .catch((error) => setMessage(error instanceof Error ? error.message : "Could not read DATA_DIR"));
  }, []);

  const chooseDirectory = async () => {
    setMessage(null);
    setBusy(true);
    try {
      const selected = await invoke<string | null>("pick_data_dir", { current: directory?.path });
      if (!selected) return;
      if (!window.confirm("GoFlix will restart its containers. Existing downloads are not moved to the new folder. Continue?")) {
        return;
      }
      const saved = await invoke<DataDirectory>("save_data_dir", { path: selected });
      await invoke<void>("restart_compose");
      setDirectory(saved);
      setMessage("Download folder updated and containers restarted.");
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "Could not update DATA_DIR");
    } finally {
      setBusy(false);
    }
  };

  if (!available) return null;

  return (
    <section className="mb-6 rounded-lg border border-white/[0.09] bg-panel px-4 py-3.5 sm:px-5">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div className="flex min-w-0 items-center gap-3">
          <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-md border border-mint/20 bg-mint/[0.07] text-mint">
            <FolderCog className="h-4 w-4" />
          </div>
          <div className="min-w-0">
            <p className="text-sm font-medium text-slate-200">Shared download folder</p>
            <p className="truncate text-xs text-slate-500" title={directory?.path}>
              {directory?.path ?? "Loading DATA_DIR…"}
            </p>
          </div>
        </div>
        <button
          type="button"
          onClick={() => void chooseDirectory()}
          disabled={busy || !directory}
          className="inline-flex h-9 shrink-0 items-center justify-center gap-2 rounded-md border border-white/[0.12] bg-white/[0.04] px-3 text-xs font-medium text-slate-200 transition hover:border-mint/40 hover:text-mint disabled:cursor-not-allowed disabled:opacity-45"
        >
          {busy ? <LoaderCircle className="h-3.5 w-3.5 animate-spin" /> : <FolderOpen className="h-3.5 w-3.5" />}
          Choose folder
        </button>
      </div>
      {message && <p className="mt-3 text-xs text-slate-400">{message}</p>}
    </section>
  );
}
