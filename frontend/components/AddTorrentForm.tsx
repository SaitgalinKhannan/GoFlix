"use client";

import { useRef, useState } from "react";
import { addMagnet, addTorrentFile } from "@/lib/api";

export default function AddTorrentForm({
  onAdded,
}: {
  onAdded: () => void;
}) {
  const [magnet, setMagnet] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [dragOver, setDragOver] = useState(false);
  const fileInput = useRef<HTMLInputElement>(null);

  const submitMagnet = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    const value = magnet.trim();
    if (!value) return;
    setBusy(true);
    try {
      await addMagnet(value);
      setMagnet("");
      onAdded();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to add torrent");
    } finally {
      setBusy(false);
    }
  };

  const uploadFile = async (file: File) => {
    if (!file.name.endsWith(".torrent")) {
      setError("Please drop a .torrent file");
      return;
    }
    setError(null);
    setBusy(true);
    try {
      await addTorrentFile(file);
      onAdded();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to add torrent");
    } finally {
      setBusy(false);
    }
  };

  const onDrop = (e: React.DragEvent) => {
    e.preventDefault();
    setDragOver(false);
    const file = e.dataTransfer.files?.[0];
    if (file) uploadFile(file);
  };

  return (
    <div className="rounded-2xl border border-white/10 bg-white/5 p-5">
      <form onSubmit={submitMagnet} className="flex flex-col gap-3 sm:flex-row">
        <input
          type="text"
          value={magnet}
          onChange={(e) => setMagnet(e.target.value)}
          placeholder="magnet:?xt=urn:btih:…"
          className="flex-1 rounded-lg border border-white/10 bg-black/40 px-3 py-2 text-sm text-white outline-none focus:border-brand"
        />
        <button
          type="submit"
          disabled={busy || !magnet.trim()}
          className="rounded-lg bg-brand px-5 py-2 text-sm font-medium text-white transition hover:bg-brand-dark disabled:opacity-50"
        >
          {busy ? "Adding…" : "Add magnet"}
        </button>
      </form>

      <div className="my-3 flex items-center gap-3 text-xs text-gray-500">
        <div className="h-px flex-1 bg-white/10" />
        OR
        <div className="h-px flex-1 bg-white/10" />
      </div>

      <div
        onClick={() => fileInput.current?.click()}
        onDragOver={(e) => {
          e.preventDefault();
          setDragOver(true);
        }}
        onDragLeave={() => setDragOver(false)}
        onDrop={onDrop}
        className={`cursor-pointer rounded-lg border border-dashed px-4 py-6 text-center text-sm transition ${
          dragOver
            ? "border-brand bg-brand/10 text-white"
            : "border-white/15 text-gray-400 hover:border-white/30"
        }`}
      >
        Drop a <span className="font-mono">.torrent</span> file here, or click to browse
        <input
          ref={fileInput}
          type="file"
          accept=".torrent"
          className="hidden"
          onChange={(e) => {
            const f = e.target.files?.[0];
            if (f) uploadFile(f);
            e.target.value = "";
          }}
        />
      </div>

      {error && <p className="mt-3 text-sm text-red-400">{error}</p>}
    </div>
  );
}
