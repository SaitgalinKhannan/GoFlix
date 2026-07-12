"use client";

import { useEffect, useState } from "react";
import {
  ArrowDownToLine,
  ArrowUpFromLine,
  Gauge,
  LoaderCircle,
  Save,
  X,
} from "lucide-react";
import { getSpeedLimits, updateSpeedLimits } from "@/lib/api";
import type { SpeedLimits } from "@/lib/types";

const MEBIBYTE = 1024 * 1024;

function toMiB(bytes: number, fallback: string): string {
  if (bytes <= 0) return fallback;
  return String(Number((bytes / MEBIBYTE).toFixed(2)));
}

export default function SpeedLimitControl() {
  const [open, setOpen] = useState(false);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [limited, setLimited] = useState(false);
  const [downloadEnabled, setDownloadEnabled] = useState(false);
  const [uploadEnabled, setUploadEnabled] = useState(false);
  const [download, setDownload] = useState("10");
  const [upload, setUpload] = useState("5");
  const [error, setError] = useState<string | null>(null);

  const applyLimitsToForm = (limits: SpeedLimits) => {
    setDownloadEnabled(limits.download > 0);
    setUploadEnabled(limits.upload > 0);
    setDownload(toMiB(limits.download, "10"));
    setUpload(toMiB(limits.upload, "5"));
    setLimited(limits.download > 0 || limits.upload > 0);
  };

  useEffect(() => {
    let active = true;
    void getSpeedLimits()
      .then((limits) => {
        if (active) applyLimitsToForm(limits);
      })
      .catch((caught) => {
        if (active) setError(caught instanceof Error ? caught.message : "Failed to load limits");
      })
      .finally(() => {
        if (active) setLoading(false);
      });
    return () => {
      active = false;
    };
  }, []);

  useEffect(() => {
    if (!open) return;
    const closeOnEscape = (event: KeyboardEvent) => {
      if (event.key === "Escape") setOpen(false);
    };
    window.addEventListener("keydown", closeOnEscape);
    return () => window.removeEventListener("keydown", closeOnEscape);
  }, [open]);

  const save = async () => {
    const downloadValue = Number(download);
    const uploadValue = Number(upload);
    if ((downloadEnabled && (!Number.isFinite(downloadValue) || downloadValue <= 0)) ||
        (uploadEnabled && (!Number.isFinite(uploadValue) || uploadValue <= 0))) {
      setError("Enabled limits must be greater than zero");
      return;
    }

    setSaving(true);
    setError(null);
    try {
      const limits = await updateSpeedLimits({
        download: downloadEnabled ? Math.round(downloadValue * MEBIBYTE) : 0,
        upload: uploadEnabled ? Math.round(uploadValue * MEBIBYTE) : 0,
      });
      applyLimitsToForm(limits);
      setOpen(false);
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : "Failed to save limits");
    } finally {
      setSaving(false);
    }
  };

  return (
    <>
      <button
        type="button"
        onClick={() => setOpen(true)}
        className={`icon-button relative ${limited ? "border-mint/25 text-mint" : ""}`}
        title="Global speed limits"
        aria-label="Global speed limits"
      >
        {loading ? <LoaderCircle className="h-4 w-4 animate-spin" /> : <Gauge className="h-4 w-4" />}
        {limited && <span className="absolute right-1 top-1 h-1.5 w-1.5 rounded-full bg-mint" />}
      </button>

      {open && (
        <div
          className="fixed inset-0 z-50 flex items-center justify-center bg-black/75 p-4 backdrop-blur-sm"
          onClick={() => setOpen(false)}
          role="presentation"
        >
          <div
            role="dialog"
            aria-modal="true"
            aria-labelledby="speed-limits-title"
            className="w-full max-w-md rounded-lg border border-white/[0.12] bg-[#101719] shadow-2xl"
            onClick={(event) => event.stopPropagation()}
          >
            <div className="flex items-start justify-between border-b border-white/[0.08] p-5">
              <div className="flex gap-3">
                <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-md border border-mint/20 bg-mint/[0.06] text-mint">
                  <Gauge className="h-4 w-4" />
                </div>
                <div>
                  <h2 id="speed-limits-title" className="font-semibold text-slate-100">Global speed limits</h2>
                  <p className="mt-1 text-xs text-slate-500">Shared by every active torrent</p>
                </div>
              </div>
              <button type="button" onClick={() => setOpen(false)} className="icon-button h-8 w-8" title="Close" aria-label="Close speed limits">
                <X className="h-4 w-4" />
              </button>
            </div>

            <div className="space-y-3 p-5">
              <LimitRow
                icon={ArrowDownToLine}
                label="Download"
                enabled={downloadEnabled}
                onEnabledChange={setDownloadEnabled}
                value={download}
                onValueChange={setDownload}
              />
              <LimitRow
                icon={ArrowUpFromLine}
                label="Upload"
                enabled={uploadEnabled}
                onEnabledChange={setUploadEnabled}
                value={upload}
                onValueChange={setUpload}
              />

              {error && (
                <p className="rounded-md border border-red-400/20 bg-red-400/[0.06] px-3 py-2 text-xs text-red-300">{error}</p>
              )}

              <div className="flex justify-end gap-2 pt-2">
                <button type="button" onClick={() => setOpen(false)} className="h-9 rounded-md border border-white/10 px-4 text-sm text-slate-300 transition hover:bg-white/[0.05]">
                  Cancel
                </button>
                <button
                  type="button"
                  onClick={() => void save()}
                  disabled={saving || loading}
                  className="inline-flex h-9 items-center gap-2 rounded-md bg-mint px-4 text-sm font-semibold text-[#04251d] transition hover:bg-[#58ebc6] disabled:opacity-40"
                >
                  {saving ? <LoaderCircle className="h-4 w-4 animate-spin" /> : <Save className="h-4 w-4" />}
                  Save limits
                </button>
              </div>
            </div>
          </div>
        </div>
      )}
    </>
  );
}

function LimitRow({
  icon: Icon,
  label,
  enabled,
  onEnabledChange,
  value,
  onValueChange,
}: {
  icon: typeof ArrowDownToLine;
  label: string;
  enabled: boolean;
  onEnabledChange: (enabled: boolean) => void;
  value: string;
  onValueChange: (value: string) => void;
}) {
  return (
    <div className="rounded-md border border-white/[0.08] bg-black/15 p-3.5">
      <div className="flex items-center justify-between gap-4">
        <div className="flex items-center gap-2.5">
          <Icon className={`h-4 w-4 ${enabled ? "text-mint" : "text-slate-600"}`} />
          <span className="text-sm font-medium text-slate-200">{label}</span>
        </div>
        <label className="relative inline-flex cursor-pointer items-center">
          <input
            type="checkbox"
            checked={enabled}
            onChange={(event) => onEnabledChange(event.target.checked)}
            className="peer sr-only"
            aria-label={`Limit ${label.toLowerCase()} speed`}
          />
          <span className="h-5 w-9 rounded-full border border-white/10 bg-slate-700 transition peer-checked:border-mint/30 peer-checked:bg-mint/25" />
          <span className="absolute left-0.5 h-4 w-4 rounded-full bg-slate-400 transition peer-checked:translate-x-4 peer-checked:bg-mint" />
        </label>
      </div>

      <div className={`mt-3 flex items-center gap-2 transition ${enabled ? "opacity-100" : "pointer-events-none opacity-35"}`}>
        <input
          type="number"
          min="0.01"
          max="1048576"
          step="0.1"
          value={value}
          onChange={(event) => onValueChange(event.target.value)}
          className="h-10 min-w-0 flex-1 rounded-md border border-white/[0.09] bg-[#0a0f11] px-3 text-sm tabular-nums text-slate-100 outline-none focus:border-mint/60"
          aria-label={`${label} limit in MiB per second`}
        />
        <span className="w-12 shrink-0 text-xs font-medium text-slate-500">MiB/s</span>
      </div>
    </div>
  );
}
