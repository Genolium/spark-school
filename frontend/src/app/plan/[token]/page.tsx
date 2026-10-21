"use client";

import React, { useEffect, useState } from "react";
import { useParams } from "next/navigation";
import Link from "next/link";
import { CheckCircle2, Send, ArrowLeft, AlertTriangle } from "lucide-react";
import { DiagnosticResult, getDiagnosticPlan } from "@/lib/diagnostic";

export default function PlanPage() {
  const params = useParams<{ token: string }>();
  const [data, setData] = useState<(DiagnosticResult & { linked: boolean }) | null>(null);
  const [error, setError] = useState(false);

  useEffect(() => {
    if (!params?.token) return;
    getDiagnosticPlan(params.token).then((result) => {
      setData(result);
      setError(!result);
    });
  }, [params?.token]);

  if (error) {
    return (
      <main className="min-h-[70vh] flex items-center justify-center px-4">
        <div className="text-center max-w-md">
          <h1 className="text-2xl font-bold mb-3">План не найден</h1>
          <p className="text-sm text-slate-500 mb-6">Ссылка устарела. Пройди диагностику заново на главной странице.</p>
          <Link href="/" className="inline-flex items-center gap-2 rounded-xl bg-emerald-600 px-5 py-2.5 text-sm font-semibold text-white">
            <ArrowLeft className="w-4 h-4" /> На главную
          </Link>
        </div>
      </main>
    );
  }

  if (!data) {
    return <main className="min-h-[70vh] flex items-center justify-center text-sm text-slate-500">Загружаем план…</main>;
  }

  const { plan } = data;
  return (
    <main className="min-h-screen bg-[var(--bg-base)] text-[var(--text-main)] px-4 py-16">
      <div className="max-w-3xl mx-auto">
        <Link href="/" className="inline-flex items-center gap-2 text-xs font-mono text-slate-500 hover:text-emerald-600 mb-8">
          <ArrowLeft className="w-3.5 h-3.5" /> На главную
        </Link>
        <div className="rounded-3xl border border-emerald-500/25 bg-white dark:bg-[#12161f] p-6 sm:p-10 shadow-xl">
          <div className="flex flex-wrap items-center justify-between gap-3 mb-5">
            <span className="rounded-full bg-emerald-500/15 px-3 py-1 text-xs font-mono font-bold text-emerald-700 dark:text-emerald-300">
              Персональный план подготовки
            </span>
            <span className="text-2xl font-black font-mono text-emerald-600 dark:text-emerald-400">{plan.score}%</span>
          </div>
          <h1 className="text-2xl sm:text-4xl font-bold mb-3">Что делать дальше</h1>
          <p className="text-sm sm:text-base text-slate-600 dark:text-slate-300 leading-relaxed mb-6">{plan.summary}</p>

          {plan.warning && (
            <div className="mb-6 rounded-2xl border border-amber-500/30 bg-amber-500/10 p-4 text-sm text-amber-800 dark:text-amber-200 flex gap-2">
              <AlertTriangle className="w-5 h-5 shrink-0" />
              <span>{plan.warning}</span>
            </div>
          )}

          <div className="space-y-3 mb-8">
            {plan.steps.map((step, index) => (
              <div key={step.title} className="rounded-2xl border border-slate-200 dark:border-white/10 bg-slate-50 dark:bg-white/5 p-4">
                <div className="flex items-start gap-3">
                  <CheckCircle2 className="w-5 h-5 text-emerald-500 shrink-0 mt-0.5" />
                  <div>
                    <p className="font-semibold text-sm">{index + 1}. {step.title}</p>
                    <p className="text-xs sm:text-sm text-slate-600 dark:text-slate-300 mt-1">{step.description}</p>
                  </div>
                </div>
              </div>
            ))}
          </div>

          <div className="flex flex-col sm:flex-row gap-3">
            <a
              href={data.telegram_url}
              target="_blank"
              rel="noopener noreferrer"
              className="inline-flex items-center justify-center gap-2 rounded-xl bg-emerald-600 px-5 py-3 text-sm font-bold text-white"
            >
              <Send className="w-4 h-4" /> Открыть план в Telegram
            </a>
            <Link
              href="/#pricing"
              className="inline-flex items-center justify-center gap-2 rounded-xl border border-slate-300 dark:border-white/15 px-5 py-3 text-sm font-semibold"
            >
              Посмотреть тарифы
            </Link>
          </div>
          <p className="mt-5 text-xs text-slate-500">
            Это план подготовки, а не гарантия гранта. Персональный разбор — в личном чате с Илем.
          </p>
        </div>
      </div>
    </main>
  );
}
