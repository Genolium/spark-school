"use client";

import React, { useCallback, useEffect, useState } from "react";
import { api, ExperimentMetric } from "@/lib/api";

const EXPERIMENT_LABELS: Record<string, string> = {
  hero_cta_copy: "Главный CTA",
  hero_headline: "Заголовок первого экрана",
  pricing_cta_copy: "CTA в тарифах",
  pricing_price: "Цена Акселератора",
};

const VARIANT_LABELS: Record<string, string> = {
  control: "Контроль",
  benefit: "Выгода",
  diagnostic: "Диагностика",
  urgency: "Дедлайн",
  action: "Оплатить и начать",
  lower: "9 990 ₽",
};

export function ExperimentsReport() {
  const [rows, setRows] = useState<ExperimentMetric[]>([]);
  const [error, setError] = useState(false);

  const load = useCallback(async () => {
    const result = await api.getExperiments();
    setRows(result || []);
    setError(!result);
  }, []);

  useEffect(() => {
    load();
  }, [load]);

  return (
    <div className="rounded-3xl border border-slate-200 dark:border-white/10 bg-white dark:bg-white/5 p-6 mb-6">
      <div className="flex items-center justify-between mb-4">
        <div>
          <h3 className="font-semibold text-lg">A/B-эксперименты</h3>
          <p className="text-xs text-slate-500">
            Успех — только подтверждённая оплата в payment_transactions.
          </p>
        </div>
        <button onClick={load} className="rounded-lg bg-emerald-600 text-white px-3 py-1 text-sm">
          Обновить
        </button>
      </div>
      {error && <p className="text-sm text-red-500">Не удалось загрузить эксперименты.</p>}
      {!error && rows.length === 0 && <p className="text-sm text-slate-500">Пока нет назначений.</p>}
      {rows.length > 0 && (
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="text-left text-slate-500 border-b">
                <th className="py-2">Эксперимент</th>
                <th className="py-2">Вариант</th>
                <th className="py-2">Посетители</th>
                <th className="py-2">Лиды</th>
                <th className="py-2">Заказы</th>
                <th className="py-2">Оплаты</th>
                <th className="py-2">Конверсия</th>
                <th className="py-2">Uplift</th>
                <th className="py-2">p-value</th>
                <th className="py-2">Значимость</th>
                <th className="py-2">Выручка</th>
              </tr>
            </thead>
            <tbody>
              {rows.map((row) => (
                <tr key={`${row.experiment}:${row.variant}`} className="border-b last:border-0">
                  <td className="py-2">{EXPERIMENT_LABELS[row.experiment] || row.experiment}</td>
                  <td className="py-2">{VARIANT_LABELS[row.variant] || row.variant}</td>
                  <td className="py-2 font-mono">{row.visitors}</td>
                  <td className="py-2 font-mono">{row.bot_leads}</td>
                  <td className="py-2 font-mono">{row.orders}</td>
                  <td className="py-2 font-mono font-semibold">{row.payers}</td>
                  <td className="py-2 font-mono">{row.conversion}%</td>
                  <td className="py-2 font-mono">
                    {row.variant === "control" ? "—" : `${row.uplift_percent > 0 ? "+" : ""}${row.uplift_percent}%`}
                  </td>
                  <td className="py-2 font-mono">{row.variant === "control" ? "—" : row.p_value}</td>
                  <td className="py-2">
                    {row.variant === "control" ? (
                      <span className="text-slate-500">база</span>
                    ) : row.significant ? (
                      <span className="rounded-full bg-emerald-500/15 px-2 py-0.5 text-xs text-emerald-600 dark:text-emerald-400">
                        значимо
                      </span>
                    ) : (
                      <span className="text-xs text-amber-600 dark:text-amber-400">
                        {row.min_sample_reached ? "не значимо" : `мало данных · нужно ~${row.required_sample_per_variant}`}
                      </span>
                    )}
                  </td>
                  <td className="py-2 font-mono">{row.revenue.toLocaleString("ru-RU")} ₽</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
