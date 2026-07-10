"use client";

import { useEffect, useState } from "react";
import type { LucideIcon } from "lucide-react";
import {
  ArrowDownToLine,
  ArrowUpFromLine,
  Check,
  CircleCheckBig,
  Clock3,
  Copy,
  FileStack,
  ListEnd,
  LoaderCircle,
  Pause,
  Play,
  Radio,
  RadioTower,
  RefreshCw,
  ShieldCheck,
  Square,
  Trash2,
  TriangleAlert,
  X,
} from "lucide-react";
import { deleteTorrent, recheckTorrent, startSeeding, stopSeeding } from "@/lib/api";
import { formatBytes, formatEta, formatPercent, formatSpeed } from "@/lib/format";
import type { TorrentStatus, TorrentStatusValue } from "@/lib/types";

type StatusPresentation = {
  label: string;
  icon: LucideIcon;
  tone: string;
  dot: string;
  progress: string;
};

const STATUS: Record<TorrentStatusValue, StatusPresentation> = {
  metadata: {
    label: "Metadata",
    icon: RadioTower,
    tone: "border-amber-300/20 bg-amber-300/[0.07] text-amber-200",
    dot: "bg-amber-300",
    progress: "#e9b949",
  },
  checking: {
    label: "Checking",
    icon: ShieldCheck,
    tone: "border-cyan-300/20 bg-cyan-300/[0.07] text-cyan-200",
    dot: "bg-cyan-300",
    progress: "#54bfe8",
  },
  queued: {
    label: "Queued",
    icon: ListEnd,
    tone: "border-slate-300/15 bg-slate-300/[0.06] text-slate-300",
    dot: "bg-slate-400",
    progress: "#7d9097",
  },
  downloading: {
    label: "Downloading",
    icon: ArrowDownToLine,
    tone: "border-mint/20 bg-mint/[0.07] text-mint",
    dot: "bg-mint",
    progress: "#36e0b5",
  },
  seeding: {
    label: "Seeding",
    icon: Radio,
    tone: "border-sky-300/20 bg-sky-300/[0.07] text-sky-200",
    dot: "bg-sky-300",
    progress: "#54bfe8",
  },
  completed: {
    label: "Complete",
    icon: CircleCheckBig,
    tone: "border-emerald-300/20 bg-emerald-300/[0.07] text-emerald-200",
    dot: "bg-emerald-300",
    progress: "#65d6a6",
  },
  paused: {
    label: "Paused",
    icon: Pause,
    tone: "border-slate-300/15 bg-slate-300/[0.05] text-slate-400",
    dot: "bg-slate-500",
    progress: "#77898f",
  },
  error: {
    label: "Error",
    icon: TriangleAlert,
    tone: "border-red-300/20 bg-red-300/[0.07] text-red-300",
    dot: "bg-red-400",
    progress: "#ff7474",
  },
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
  const [copied, setCopied] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!confirmOpen) return;
    const closeOnEscape = (event: KeyboardEvent) => {
      if (event.key === "Escape") setConfirmOpen(false);
    };
    window.addEventListener("keydown", closeOnEscape);
    return () => window.removeEventListener("keydown", closeOnEscape);
  }, [confirmOpen]);

  const checking = torrent.status === "checking";
  const active = ["downloading", "seeding", "metadata"].includes(torrent.status);
  const presentation = STATUS[torrent.status];
  const StatusIcon = presentation.icon;
  const progress = Math.min(100, Math.max(0, torrent.progress * 100));

  const act = async (action: () => Promise<void>) => {
    setBusy(true);
    setError(null);
    try {
      await action();
      onChanged();
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : "Action failed");
    } finally {
      setBusy(false);
    }
  };

  const doDelete = () => {
    void act(() => deleteTorrent(torrent.id, deleteFiles).then(() => setConfirmOpen(false)));
  };

  const copyHash = async () => {
    await navigator.clipboard.writeText(torrent.infoHash);
    setCopied(true);
    window.setTimeout(() => setCopied(false), 1400);
  };

  return (
    <article className="group relative overflow-hidden rounded-lg border border-white/[0.09] bg-panel transition duration-200 hover:border-white/[0.16] hover:bg-panel-raised">
      <div className="absolute inset-y-0 left-0 w-[2px]" style={{ backgroundColor: presentation.progress }} />

      <div className="p-4 sm:p-5">
        <div className="flex items-start justify-between gap-4">
          <div className="min-w-0 flex-1">
            <h3 className="truncate text-sm font-semibold text-slate-100 sm:text-[15px]" title={torrent.name}>
              {torrent.name || torrent.infoHash.slice(0, 12)}
            </h3>
            <div className="mt-1.5 flex min-w-0 items-center gap-1.5">
              <span className="truncate font-mono text-[10px] text-slate-600 sm:text-[11px]">{torrent.infoHash}</span>
              <button
                type="button"
                onClick={() => void copyHash()}
                className="shrink-0 text-slate-600 transition hover:text-slate-300"
                title="Copy info hash"
                aria-label="Copy info hash"
              >
                {copied ? <Check className="h-3.5 w-3.5 text-mint" /> : <Copy className="h-3.5 w-3.5" />}
              </button>
            </div>
          </div>

          <div className={`inline-flex h-7 shrink-0 items-center gap-2 rounded-full border px-2.5 text-[11px] font-semibold ${presentation.tone}`}>
            <span className={`h-1.5 w-1.5 rounded-full ${presentation.dot} ${active || checking ? "status-pulse" : ""}`} />
            <StatusIcon className="h-3.5 w-3.5" />
            <span className="hidden sm:inline">{presentation.label}</span>
          </div>
        </div>

        <div className="mt-5">
          <div className="mb-2 flex items-center justify-between gap-4 text-xs tabular-nums">
            <span className="text-slate-400">
              <strong className="font-medium text-slate-200">{formatBytes(torrent.downloaded)}</strong>
              <span className="mx-1 text-slate-600">/</span>
              {formatBytes(torrent.size)}
            </span>
            <span className="font-semibold text-slate-200">{formatPercent(torrent.progress)}</span>
          </div>
          <div className="h-1.5 overflow-hidden rounded-full bg-black/35">
            <div
              className="h-full rounded-full transition-[width] duration-700 ease-out"
              style={{ width: `${progress}%`, backgroundColor: presentation.progress }}
            />
          </div>
        </div>

        <div className="mt-5 flex flex-col gap-4 border-t border-white/[0.07] pt-4 lg:flex-row lg:items-center lg:justify-between">
          <div className="grid min-w-0 grid-cols-2 gap-x-6 gap-y-3 sm:flex sm:items-center sm:gap-7">
            <Metric icon={ArrowDownToLine} label="Down" value={formatSpeed(torrent.downloadSpeed)} active={torrent.downloadSpeed > 0} />
            <Metric icon={ArrowUpFromLine} label="Up" value={formatSpeed(torrent.uploadSpeed)} />
            <Metric icon={Clock3} label="ETA" value={formatEta(torrent.eta)} />
            <Metric icon={FileStack} label="Size" value={formatBytes(torrent.size)} />
          </div>

          <div className="flex shrink-0 items-center gap-2">
            {active || checking ? (
              <ActionButton
                icon={Square}
                label={torrent.status === "seeding" ? "Stop seeding" : "Stop"}
                onClick={() => void act(() => stopSeeding(torrent.id))}
                disabled={busy}
              />
            ) : (
              <ActionButton
                icon={Play}
                label="Start"
                onClick={() => void act(() => startSeeding(torrent.id))}
                disabled={busy}
              />
            )}
            <ActionButton
              icon={checking ? LoaderCircle : RefreshCw}
              label={checking ? "Checking" : "Recheck"}
              onClick={() => void act(() => recheckTorrent(torrent.id))}
              disabled={busy || checking}
              accent
              spin={checking}
            />
            <button
              type="button"
              onClick={() => {
                setDeleteFiles(false);
                setConfirmOpen(true);
              }}
              disabled={busy}
              className="icon-button ml-auto h-8 w-8 hover:border-red-400/30 hover:bg-red-400/[0.07] hover:text-red-300 lg:ml-0"
              title="Delete torrent"
              aria-label="Delete torrent"
            >
              <Trash2 className="h-3.5 w-3.5" />
            </button>
          </div>
        </div>

        {(torrent.error || error) && (
          <div className="mt-4 flex items-start gap-2 rounded-md border border-red-400/20 bg-red-400/[0.06] px-3 py-2 text-xs text-red-300">
            <TriangleAlert className="mt-0.5 h-3.5 w-3.5 shrink-0" />
            <span>{torrent.error || error}</span>
          </div>
        )}
      </div>

      {confirmOpen && (
        <div
          className="fixed inset-0 z-50 flex items-center justify-center bg-black/75 p-4 backdrop-blur-sm"
          onClick={() => setConfirmOpen(false)}
          role="presentation"
        >
          <div
            role="dialog"
            aria-modal="true"
            aria-labelledby={`delete-${torrent.id}`}
            className="w-full max-w-md rounded-lg border border-white/[0.12] bg-[#101719] shadow-2xl"
            onClick={(event) => event.stopPropagation()}
          >
            <div className="flex items-start justify-between border-b border-white/[0.08] p-5">
              <div className="flex gap-3">
                <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-md bg-red-400/[0.08] text-red-300">
                  <Trash2 className="h-4 w-4" />
                </div>
                <div>
                  <h3 id={`delete-${torrent.id}`} className="font-semibold text-slate-100">Delete torrent?</h3>
                  <p className="mt-1 line-clamp-2 text-xs leading-5 text-slate-500">{torrent.name}</p>
                </div>
              </div>
              <button type="button" onClick={() => setConfirmOpen(false)} className="icon-button h-8 w-8" title="Close" aria-label="Close dialog">
                <X className="h-4 w-4" />
              </button>
            </div>

            <div className="p-5">
              <label className="flex cursor-pointer items-start gap-3 rounded-md border border-white/[0.08] bg-black/15 p-3.5">
                <input
                  type="checkbox"
                  checked={deleteFiles}
                  onChange={(event) => setDeleteFiles(event.target.checked)}
                  className="mt-0.5 h-4 w-4 rounded border-white/20 accent-[#36e0b5]"
                />
                <span>
                  <span className="block text-sm font-medium text-slate-200">Delete downloaded files</span>
                  <span className="mt-1 block text-xs text-slate-500">This cannot be undone.</span>
                </span>
              </label>

              <div className="mt-5 flex justify-end gap-2">
                <button type="button" onClick={() => setConfirmOpen(false)} className="h-9 rounded-md border border-white/10 px-4 text-sm text-slate-300 transition hover:bg-white/[0.05]">
                  Cancel
                </button>
                <button
                  type="button"
                  onClick={doDelete}
                  disabled={busy}
                  className="inline-flex h-9 items-center gap-2 rounded-md bg-red-500 px-4 text-sm font-semibold text-white transition hover:bg-red-400 disabled:opacity-50"
                >
                  {busy ? <LoaderCircle className="h-4 w-4 animate-spin" /> : <Trash2 className="h-4 w-4" />}
                  Delete
                </button>
              </div>
            </div>
          </div>
        </div>
      )}
    </article>
  );
}

function Metric({ icon: Icon, label, value, active = false }: { icon: LucideIcon; label: string; value: string; active?: boolean }) {
  return (
    <div className="flex min-w-0 items-center gap-2">
      <Icon className={`h-3.5 w-3.5 shrink-0 ${active ? "text-mint" : "text-slate-600"}`} />
      <div className="min-w-0">
        <p className="text-[9px] font-semibold uppercase text-slate-600">{label}</p>
        <p className={`mt-0.5 truncate text-[11px] font-medium tabular-nums ${active ? "text-mint" : "text-slate-300"}`}>{value}</p>
      </div>
    </div>
  );
}

function ActionButton({
  icon: Icon,
  label,
  onClick,
  disabled,
  accent = false,
  spin = false,
}: {
  icon: LucideIcon;
  label: string;
  onClick: () => void;
  disabled: boolean;
  accent?: boolean;
  spin?: boolean;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      disabled={disabled}
      className={`inline-flex h-8 items-center justify-center gap-1.5 rounded-md border px-3 text-[11px] font-semibold transition disabled:cursor-not-allowed disabled:opacity-40 ${
        accent
          ? "border-mint/20 bg-mint/[0.05] text-mint hover:bg-mint/[0.1]"
          : "border-white/[0.1] bg-white/[0.025] text-slate-300 hover:border-white/20 hover:bg-white/[0.06] hover:text-white"
      }`}
    >
      <Icon className={`h-3.5 w-3.5 ${spin ? "animate-spin" : ""}`} />
      {label}
    </button>
  );
}
