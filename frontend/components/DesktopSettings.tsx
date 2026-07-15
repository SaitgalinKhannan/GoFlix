"use client";

import { Settings, SlidersHorizontal, X } from "lucide-react";
import { useEffect, useState } from "react";
import DesktopDataDirectory from "@/components/DesktopDataDirectory";
import DesktopServices from "@/components/DesktopServices";
import { isDesktop } from "@/lib/desktop";

export default function DesktopSettings() {
  const [available, setAvailable] = useState(false);
  const [open, setOpen] = useState(false);

  useEffect(() => {
    setAvailable(isDesktop());
  }, []);

  useEffect(() => {
    if (!open) return;
    const closeOnEscape = (event: KeyboardEvent) => {
      if (event.key === "Escape") setOpen(false);
    };
    window.addEventListener("keydown", closeOnEscape);
    return () => window.removeEventListener("keydown", closeOnEscape);
  }, [open]);

  if (!available) return null;

  return (
    <>
      <button
        type="button"
        onClick={() => setOpen(true)}
        className="icon-button"
        title="Desktop settings"
        aria-label="Desktop settings"
      >
        <Settings className="h-4 w-4" />
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
            aria-labelledby="desktop-settings-title"
            className="flex max-h-[calc(100vh-2rem)] w-full max-w-2xl flex-col overflow-hidden rounded-lg border border-white/[0.12] bg-[#101719] shadow-2xl"
            onClick={(event) => event.stopPropagation()}
          >
            <div className="flex shrink-0 items-start justify-between border-b border-white/[0.08] p-5">
              <div className="flex gap-3">
                <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-md border border-mint/20 bg-mint/[0.06] text-mint">
                  <SlidersHorizontal className="h-4 w-4" />
                </div>
                <div>
                  <h2 id="desktop-settings-title" className="font-semibold text-slate-100">Desktop settings</h2>
                  <p className="mt-1 text-xs text-slate-500">Storage and local services</p>
                </div>
              </div>
              <button
                type="button"
                onClick={() => setOpen(false)}
                className="icon-button h-8 w-8"
                title="Close"
                aria-label="Close desktop settings"
              >
                <X className="h-4 w-4" />
              </button>
            </div>

            <div className="scrollbar-thin overflow-y-auto p-5 pb-0">
              <DesktopDataDirectory />
              <DesktopServices />
            </div>
          </div>
        </div>
      )}
    </>
  );
}
