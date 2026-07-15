"use client";

import { useRef, useState } from "react";
import {
  FileUp,
  Link2,
  LoaderCircle,
  Magnet,
  Plus,
  UploadCloud,
} from "lucide-react";
import { addMagnet, addTorrentFile } from "@/lib/api";

type AddMode = "magnet" | "file";

export default function AddTorrentForm({ onAdded }: { onAdded: () => void }) {
  const [mode, setMode] = useState<AddMode>("magnet");
  const [magnet, setMagnet] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [dragOver, setDragOver] = useState(false);
  const fileInput = useRef<HTMLInputElement>(null);

  const submitMagnet = async (event: React.FormEvent) => {
    event.preventDefault();
    setError(null);
    const value = magnet.trim();
    if (!value) return;
    setBusy(true);
    try {
      await addMagnet(value);
      setMagnet("");
      onAdded();
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : "Failed to add torrent");
    } finally {
      setBusy(false);
    }
  };

  const uploadFile = async (file: File) => {
    if (!file.name.toLowerCase().endsWith(".torrent")) {
      setError("Choose a valid .torrent file");
      return;
    }
    setError(null);
    setBusy(true);
    try {
      await addTorrentFile(file);
      onAdded();
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : "Failed to add torrent");
    } finally {
      setBusy(false);
    }
  };

  const onDrop = (event: React.DragEvent) => {
    event.preventDefault();
    setDragOver(false);
    const file = event.dataTransfer.files?.[0];
    if (file) void uploadFile(file);
  };

  return (
    <section className="overflow-hidden rounded-lg border border-white/[0.09] bg-panel shadow-[0_18px_60px_rgba(0,0,0,0.22)]">
      <div className="flex flex-col gap-4 border-b border-white/[0.07] px-4 py-4 sm:flex-row sm:items-center sm:justify-between sm:px-5">
        <div className="flex items-center gap-3">
          <div className="flex h-9 w-9 items-center justify-center rounded-md border border-mint/20 bg-mint/[0.07] text-mint">
            <Plus className="h-4 w-4" />
          </div>
          <div>
            <h2 className="text-sm font-semibold text-slate-100">Add torrent</h2>
            <p className="mt-0.5 text-xs text-slate-500">Start a new download</p>
          </div>
        </div>

        <div className="flex w-full rounded-md border border-white/[0.08] bg-[#0b1113] p-1 sm:w-auto">
          <ModeButton active={mode === "magnet"} onClick={() => setMode("magnet")} icon={Magnet}>
            Magnet
          </ModeButton>
          <ModeButton active={mode === "file"} onClick={() => setMode("file")} icon={FileUp}>
            Torrent file
          </ModeButton>
        </div>
      </div>

      <div className="p-4 sm:p-5">
        {mode === "magnet" ? (
          <form onSubmit={submitMagnet} className="flex flex-col gap-3 sm:flex-row">
            <label className="group relative min-w-0 flex-1">
              <span className="sr-only">Magnet link</span>
              <Link2 className="pointer-events-none absolute left-3.5 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-600 transition group-focus-within:text-mint" />
              <input
                type="text"
                value={magnet}
                onChange={(event) => setMagnet(event.target.value)}
                placeholder="Paste magnet link"
                className="h-11 w-full rounded-md border border-white/[0.09] bg-[#0a0f11] pl-10 pr-3 text-sm text-slate-100 outline-none transition placeholder:text-slate-600 focus:border-mint/60 focus:bg-[#0c1315]"
              />
            </label>
            <button
              type="submit"
              disabled={busy || !magnet.trim()}
              className="inline-flex h-11 shrink-0 items-center justify-center gap-2 rounded-md bg-mint px-5 text-sm font-semibold text-[#04251d] transition hover:bg-[#58ebc6] disabled:cursor-not-allowed disabled:opacity-40"
            >
              {busy ? <LoaderCircle className="h-4 w-4 animate-spin" /> : <Plus className="h-4 w-4" />}
              {busy ? "Adding" : "Add magnet"}
            </button>
          </form>
        ) : (
          <div
            role="button"
            tabIndex={0}
            onClick={() => fileInput.current?.click()}
            onKeyDown={(event) => {
              if (event.key === "Enter" || event.key === " ") fileInput.current?.click();
            }}
            onDragOver={(event) => {
              event.preventDefault();
              setDragOver(true);
            }}
            onDragLeave={() => setDragOver(false)}
            onDrop={onDrop}
            className={`flex min-h-28 cursor-pointer items-center justify-center gap-4 rounded-md border border-dashed px-5 py-6 transition ${
              dragOver
                ? "border-mint bg-mint/[0.07]"
                : "border-white/[0.13] bg-[#0b1113] hover:border-mint/40 hover:bg-[#0d1416]"
            }`}
          >
            <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-md border border-white/10 bg-white/[0.04] text-mint">
              {busy ? <LoaderCircle className="h-5 w-5 animate-spin" /> : <UploadCloud className="h-5 w-5" />}
            </div>
            <div>
              <p className="text-sm font-medium text-slate-200">Drop a .torrent file here</p>
              <p className="mt-1 text-xs text-slate-500">or click to choose from disk</p>
            </div>
            <input
              ref={fileInput}
              type="file"
              accept=".torrent,application/x-bittorrent"
              className="hidden"
              onChange={(event) => {
                const file = event.target.files?.[0];
                if (file) void uploadFile(file);
                event.target.value = "";
              }}
            />
          </div>
        )}

        {error && (
          <p className="mt-3 rounded-md border border-red-400/20 bg-red-400/[0.06] px-3 py-2 text-xs text-red-300">
            {error}
          </p>
        )}
      </div>
    </section>
  );
}

function ModeButton({
  active,
  onClick,
  icon: Icon,
  children,
}: {
  active: boolean;
  onClick: () => void;
  icon: typeof Magnet;
  children: React.ReactNode;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={`flex min-h-8 flex-1 items-center justify-center gap-2 rounded px-3 text-xs font-medium transition sm:flex-none ${
        active ? "bg-white/[0.09] text-slate-100 shadow-sm" : "text-slate-500 hover:text-slate-300"
      }`}
    >
      <Icon className={`h-3.5 w-3.5 ${active ? "text-mint" : ""}`} />
      {children}
    </button>
  );
}
