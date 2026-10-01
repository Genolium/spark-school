"use client";

import React, { useEffect } from "react";
import { motion, AnimatePresence } from "framer-motion";
import { X, Send } from "lucide-react";

interface LeadMagnetModalProps {
  isOpen: boolean;
  onClose: () => void;
}

export function LeadMagnetModal({ isOpen, onClose }: LeadMagnetModalProps) {
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === "Escape") onClose();
    };
    if (isOpen) {
      document.body.style.overflow = "hidden";
      window.addEventListener("keydown", handleKeyDown);
    }
    return () => {
      document.body.style.overflow = "unset";
      window.removeEventListener("keydown", handleKeyDown);
    };
  }, [isOpen, onClose]);

  if (!isOpen) return null;

  return (
    <AnimatePresence>
      <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
        {/* Backdrop */}
        <motion.div
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          exit={{ opacity: 0 }}
          onClick={onClose}
          className="absolute inset-0 bg-black/75 backdrop-blur-md"
        />

        {/* Modal Window */}
        <motion.div
          initial={{ opacity: 0, scale: 0.95, y: 15 }}
          animate={{ opacity: 1, scale: 1, y: 0 }}
          exit={{ opacity: 0, scale: 0.95, y: 15 }}
          transition={{ duration: 0.25 }}
          className="relative w-full max-w-lg bento-card-dark p-6 sm:p-8 rounded-2xl border border-emerald-500/20 dark:border-white/20 shadow-2xl z-10 overflow-hidden bg-white dark:bg-[#12161f]"
        >
          {/* Close Button */}
          <button
            onClick={onClose}
            className="absolute top-5 right-5 w-8 h-8 rounded-full bg-emerald-500/10 dark:bg-white/10 flex items-center justify-center text-slate-500 dark:text-slate-300 hover:text-slate-900 dark:hover:text-white hover:bg-emerald-500/20 dark:hover:bg-white/20 transition-all"
            aria-label="Закрыть"
          >
            <X className="w-4 h-4" />
          </button>

          <div>
            <div className="flex items-center gap-2 mb-3">
              <span className="bento-pill px-3 py-1 rounded-full text-xs font-mono font-medium flex items-center gap-1.5 text-slate-700 dark:text-slate-200">
                <span className="w-2 h-2 rounded-full bg-emerald-500 dark:bg-slate-400" />
                Бесплатный разбор
              </span>
            </div>

            <h3 className="text-2xl sm:text-3xl font-bold text-slate-900 dark:text-white mb-3 leading-tight">
              Анатомия заявки на $20,000
            </h3>
            <p className="text-sm sm:text-base text-slate-600 dark:text-slate-300 leading-relaxed mb-8 font-normal">
              Видеоразбор заявки финалиста SPARK 2026 + PDF-чеклист по эссе
            </p>

            {/* Launch in Bot CTA */}
            <div className="space-y-3">
              <a
                href="https://t.me/spark_prep_bot?start=free_audit"
                target="_blank"
                rel="noreferrer"
                className="btn-primary-mono w-full py-4 text-sm sm:text-base font-bold flex items-center justify-center gap-2.5 active:scale-95 shadow-xl"
              >
                <Send className="w-4 h-4" />
                <span>Забрать видео и PDF в Telegram-боте</span>
              </a>

              <div className="text-center text-[11px] font-mono text-slate-500 dark:text-slate-400">
                Бот мгновенно пришлёт материалы в личные сообщения без спама.
              </div>
            </div>
          </div>
        </motion.div>
      </div>
    </AnimatePresence>
  );
}
