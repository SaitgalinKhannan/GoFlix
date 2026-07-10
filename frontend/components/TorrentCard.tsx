"use client";

import { useState } from "react";
import { deleteTorrent, startSeeding, stopSeeding } from "@/lib/api";
import { formatBytes, formatEta, formatPercent, formatSpeed } from "@/lib/format";
import type { TorrentStatus } from "@/lib/types";

const STATUS_COLORS: Record<string, string> = {
  metadata: "text-amber-400 bg-amber-400/10",
  queued: "text-gray-400 bg-white/10",
  downloading: "text-blue-400 bg-blue-400/10",
  seeding: "text-emerald-400 bg-emerald-400/10",
  completed: "text-emerald-400 bg-emerald-400/10",
  paused: "text-gray-400 bg-white/10",
  error: "text-red-400 bg-red-400/10",
};

export default function TorrentCard({
  torrent,
  onChanged,
}: {
  torrent: TorrentStatus;
  onChanged: () => void;
}) {
  const [busy, setBusy] = useState(false);
  const [confirmOpen, setConfirmOpen] = useState(false);
  const [deleteFiles, setDeleteFiles] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const active = torrent.status === "downloading" || torrent.status === "seeding";

  const act = async (fn: () => Promise<void>) => {
    setBusy(true);
    setError(null);
    try {
      await fn();
      onChanged();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Action failed");
    } finally {
      setBusy(false);
    }
  };

  const doDelete = () => {
    act(() => deleteTorrent(torrent.id, deleteFiles).then(() => setConfirmOpen(false)));
  };

  return (
    <div className="rounded-2xl border border-white/10 bg-white/5 p-5">
      <div className="flex items-start justify-between gap-4">
        <div className="min-w-0">
          <h3 className="truncate font-medium text-white" title={torrent.name}>
            {torrent.name || torrent.infoHash.slice(0, 12)}
          </h3>
          <p className="mt-0.5 font-mono text-xs text-gray-500">{torrent.infoHash}</p>
        </div>
        <span
          className={`shrink-0 rounded-full px-2.5 py-1 text-xs font-medium capitalize ${
            STATUS_COLORS[torrent.status] ?? "text-gray-400 bg-white/10"
          }`}
        >
          {torrent.status}
        </span>
      </div>

      {/* Progress */}
      <div className="mt-4">
        <div className="mb-1 flex justify-between text-xs text-gray-400">
          <span>
            {formatBytes(torrent.downloaded)} / {formatBytes(torrent.size)}
          </span>
          <span>{formatPercent(torrent.progress)}</span>
        </div>
        <div className="h-2 overflow-hidden rounded-full bg-white/10">
          <div
            className="h-full bg-brand transition-all"
            style={{ width: `${Math.min(100, torrent.progress * 100)}%` }}
          />
        </div>
      </div>

      {/* Speeds */}
      <div className="mt-3 flex gap-6 text-xs text-gray-400">
        <span>
          ↓ <span className="text-gray-200">{formatSpeed(torrent.downloadSpeed)}</span>
        </span>
        <span>
          ↑ <span className="text-gray-200">{formatSpeed(torrent.uploadSpeed)}</span>
        </span>
        {torrent.status === "downloading" && torrent.eta > 0 && (
          <span>
            ⏳ <span className="text-gray-200">{formatEta(torrent.eta)}</span>
          </span>
        )}
      </div>

      {torrent.error && (
        <p className="mt-2 text-xs text-red-400">{torrent.error}</p>
      )}

      {/* Actions */}
      <div className="mt-4 flex flex-wrap gap-2">
        {active ? (
          <button
            onClick={() => act(() => stopSeeding(torrent.id))}
            disabled={busy}
            className="rounded-lg border border-white/15 px-3 py-1.5 text-xs text-gray-200 transition hover:bg-white/10 disabled:opacity-50"
          >
            {torrent.status === "seeding" || torrent.status === "completed" ? "Stop seeding" : "Stop"}
          </button>
        ) : (
          <button
            onClick={() => act(() => startSeeding(torrent.id))}
            disabled={busy}
            className="rounded-lg border border-white/15 px-3 py-1.5 text-xs text-gray-200 transition hover:bg-white/10 disabled:opacity-50"
          >
            Start
          </button>
        )}
        <button
          onClick={() => {
            setDeleteFiles(false);
            setConfirmOpen(true);
          }}
          disabled={busy}
          className="rounded-lg border border-red-500/30 px-3 py-1.5 text-xs text-red-400 transition hover:bg-red-500/10 disabled:opacity-50"
        >
          Delete
        </button>
      </div>

      {error && <p className="mt-2 text-xs text-red-400">{error}</p>}

      {/* Delete confirmation modal */}
      {confirmOpen && (
        <div
          className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 p-4"
          onClick={() => setConfirmOpen(false)}
        >
          <div
            className="w-full max-w-sm rounded-2xl border border-white/10 bg-[#12121a] p-6"
            onClick={(e) => e.stopPropagation()}
          >
            <h3 className="text-lg font-semibold text-white">Delete torrent?</h3>
            <p className="mt-2 text-sm text-gray-400">
              Remove <span className="text-gray-200">{torrent.name}</span> from the list.
            </p>
            <label className="mt-4 flex items-center gap-2 text-sm text-gray-300">
              <input
                type="checkbox"
                checked={deleteFiles}
                onChange={(e) => setDeleteFiles(e.target.checked)}
                className="h-4 w-4 accent-brand"
              />
              Also delete downloaded files from disk
            </label>
            <div className="mt-6 flex justify-end gap-2">
              <button
                onClick={() => setConfirmOpen(false)}
                className="rounded-lg border border-white/15 px-4 py-2 text-sm text-gray-200 hover:bg-white/10"
              >
                Cancel
              </button>
              <button
                onClick={doDelete}
                disabled={busy}
                className="rounded-lg bg-red-600 px-4 py-2 text-sm font-medium text-white hover:bg-red-700 disabled:opacity-50"
              >
                {busy ? "Deleting…" : "Delete"}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
