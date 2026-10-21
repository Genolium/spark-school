"use client";

import React, { useCallback, useEffect, useState } from "react";
import { api, FunnelData } from "@/lib/api";

const STEP_LABELS: Record<string, string> = {
  landing_view: "Визит лендинга",
  diagnostic_start: "Начало диагностики",
  diagnostic_complete: "Диагностика завершена",
  pricing_select_tier: "Выбор тарифа",
  bot_handoff: "Переход в Telegram",
  bot_lead: "Лид в боте",
  checkout_started: "Начал оформление",
  order_created: "Заказ создан",
  payment_confirmed: "Оплата подтверждена",
  access_delivered: "Доступ выдан",
};

export function FunnelAnalytics() {
  const [days, setDays] = useState(30);
  const [source, setSource] = useState("");
  const [tier, setTier] = useState("");
  const [promo, setPromo] = useState("");
  const [campaign, setCampaign] = useState("");
  const [data, setData] = useState<FunnelData | null>(null);
  const [error, setError] = useState(false);
  const [loading, setLoading] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    const to = new Date();
    const from = new Date();
    from.setDate(from.getDate() - (days - 1));
    const result = await api.getFunnel({
      from: from.toISOString().slice(0, 10),
      to: to.toISOString().slice(0, 10),
      source: source || undefined,
      campaign: campaign || undefined,
      tier: tier || undefined,
      promo: promo || undefined,
    });
    setData(result);
    setError(!result);
    setLoading(false);
  }, [days, source, tier, promo, campaign]);

  useEffect(() => {
    load();
  }, [load]);

  return (
    <div className="rounded-3xl border border-slate-200 dark:border-white/10 bg-white dark:bg-white/5 p-6 mb-6">
      <div className="flex flex-wrap items-center justify-between gap-3 mb-5">
        <div>
          <h3 className="font-semibold text-lg">Воронка привлечения</h3>
          <p className="text-xs text-slate-500">
            События: analytics_events. Деньги: только payment_transactions.
          </p>
        </div>
        <div className="flex flex-wrap gap-2">
          <select value={days} onChange={(e) => setDays(Number(e.target.value))} className="rounded-lg border px-2 py-1 text-sm bg-transparent">
            <option value={7}>7 дней</option>
            <option value={30}>30 дней</option>
            <option value={90}>90 дней</option>
          </select>
          <select value={tier} onChange={(e) => setTier(e.target.value)} className="rounded-lg border px-2 py-1 text-sm bg-transparent">
            <option value="">Все тарифы</option>
            <option value="basic">Basic</option>
            <option value="accelerator">Accelerator</option>
            <option value="vip">VIP</option>
          </select>
          <input value={source} onChange={(e) => setSource(e.target.value)} placeholder="Источник" className="w-28 rounded-lg border px-2 py-1 text-sm bg-transparent" />
          <input value={campaign} onChange={(e) => setCampaign(e.target.value)} placeholder="Кампания" className="w-28 rounded-lg border px-2 py-1 text-sm bg-transparent" />
          <input value={promo} onChange={(e) => setPromo(e.target.value)} placeholder="Промокод" className="w-28 rounded-lg border px-2 py-1 text-sm bg-transparent" />
          <button onClick={load} className="rounded-lg bg-emerald-600 text-white px-3 py-1 text-sm">
            {loading ? "Загрузка..." : "Обновить"}
          </button>
        </div>
      </div>

      {error && <p className="text-sm text-red-500">Не удалось загрузить воронку.</p>}

      {data && (
        <>
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-5">
            <div className="rounded-2xl bg-slate-50 dark:bg-white/5 p-4">
              <p className="text-xs text-slate-500">Подтверждённых оплат</p>
              <p className="text-2xl font-semibold">{data.payments.count}</p>
            </div>
            <div className="rounded-2xl bg-slate-50 dark:bg-white/5 p-4">
              <p className="text-xs text-slate-500">Выручка</p>
              <p className="text-2xl font-semibold">{data.payments.revenue.toLocaleString("ru-RU")} ₽</p>
            </div>
            <div className="rounded-2xl bg-slate-50 dark:bg-white/5 p-4">
              <p className="text-xs text-slate-500">Средний чек</p>
              <p className="text-2xl font-semibold">{data.payments.average.toLocaleString("ru-RU")} ₽</p>
            </div>
          </div>

          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="text-left text-slate-500 border-b">
                  <th className="py-2">Этап</th>
                  <th className="py-2">Уникальных</th>
                  <th className="py-2">Конверсия с прошлого</th>
                  <th className="py-2">Доля от визита</th>
                </tr>
              </thead>
              <tbody>
                {data.steps.map((step, index) => {
                  const first = data.steps[0]?.count || 0;
                  return (
                    <tr key={step.step} className="border-b last:border-0">
                      <td className="py-2">{STEP_LABELS[step.step] || step.step}</td>
                      <td className="py-2 font-mono">{step.count}</td>
                      <td className="py-2">{index === 0 ? "—" : `${step.conversion_from_previous}%`}</td>
                      <td className="py-2">{first > 0 ? `${((step.count / first) * 100).toFixed(1)}%` : "—"}</td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        </>
      )}
    </div>
  );
}
