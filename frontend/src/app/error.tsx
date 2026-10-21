"use client";

import { useEffect } from "react";
import { reportClientError } from "@/lib/errorTracking";

export default function GlobalError({ error, reset }: { error: Error & { digest?: string }; reset: () => void }) {
  useEffect(() => {
    reportClientError(error.message || "page error", error.stack);
  }, [error]);

  return (
    <main className="min-h-[60vh] flex items-center justify-center px-4">
      <div className="max-w-md text-center">
        <h1 className="text-2xl font-bold mb-2">Что-то пошло не так</h1>
        <p className="text-sm text-slate-500 mb-6">Мы уже получили сигнал об ошибке. Попробуйте обновить страницу.</p>
        <button onClick={reset} className="rounded-xl bg-emerald-600 px-5 py-2.5 text-sm font-semibold text-white">
          Попробовать снова
        </button>
      </div>
    </main>
  );
}
