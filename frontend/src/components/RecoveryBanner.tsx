"use client";

import React, { useEffect, useState } from "react";
import { X, Compass } from "lucide-react";

interface RecoveryBannerProps {
  onOpenDiagnostic: () => void;
}

const DISMISS_KEY = "spark_diagnostic_banner_dismissed";
const MIN_AGE_MS = 30 * 60 * 1000;

export function RecoveryBanner({ onOpenDiagnostic }: RecoveryBannerProps) {
  const [visible, setVisible] = useState(false);

  useEffect(() => {
    try {
      const dismissedAt = Number(localStorage.getItem(DISMISS_KEY) || 0);
      if (dismissedAt && Date.now() - dismissedAt < 24 * 60 * 60 * 1000) return;

      const raw = localStorage.getItem("spark_funnel_events");
      if (!raw) return;
      const events = JSON.parse(raw) as Array<{ step?: string; timestamp?: string }>;
      const lastStart = [...events].reverse().find((event) => event.step === "diagnostic_start");
      const lastComplete = [...events].reverse().find((event) => event.step === "diagnostic_complete");
      if (!lastStart?.timestamp) return;
      if (lastComplete?.timestamp && new Date(lastComplete.timestamp) > new Date(lastStart.timestamp)) return;
      if (Date.now() - new Date(lastStart.timestamp).getTime() < MIN_AGE_MS) return;
      setVisible(true);
    } catch {
      // Ignore malformed local storage.
    }
  }, []);

  if (!visible) return null;

  const dismiss = () => {
    try {
      localStorage.setItem(DISMISS_KEY, String(Date.now()));
    } catch {
      // ignore
    }
    setVisible(false);
  };

  return (
    <div className="fixed bottom-4 left-4 right-4 sm:left-auto sm:right-6 sm:max-w-sm z-[80] rounded-2xl border border-emerald-500/30 bg-white dark:bg-[#12161f] shadow-2xl p-4">
      <button
        type="button"
        onClick={dismiss}
        aria-label="Закрыть"
        className="absolute top-2 right-2 p-1 text-slate-400 hover:text-slate-700 dark:hover:text-white"
      >
        <X className="w-4 h-4" />
      </button>
      <div className="flex items-start gap-3 pr-5">
        <div className="w-9 h-9 rounded-xl bg-emerald-500/15 text-emerald-600 dark:text-emerald-400 flex items-center justify-center shrink-0">
          <Compass className="w-5 h-5" />
        </div>
        <div>
          <p className="text-sm font-semibold text-slate-900 dark:text-white">Вы не завершили диагностику</p>
          <p className="text-xs text-slate-500 mt-1">Осталось несколько вопросов — результат покажет готовность и ближайшие шаги.</p>
          <button
            type="button"
            onClick={() => {
              dismiss();
              onOpenDiagnostic();
            }}
            className="mt-3 rounded-xl bg-emerald-600 px-3 py-1.5 text-xs font-semibold text-white"
          >
            Продолжить
          </button>
        </div>
      </div>
    </div>
  );
}
