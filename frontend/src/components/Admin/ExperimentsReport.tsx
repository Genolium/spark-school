"use client";

import React, { useCallback, useEffect, useState } from "react";
import { api, ExperimentMetric } from "@/lib/api";

const EXPERIMENT_LABELS: Record<string, string> = {
  hero_cta_copy: "Главный CTA",
  pricing_cta_copy: "CTA в тарифах",
};

const VARIANT_LABELS: Record<string, string> = {
  control: "Контроль",
  benefit: "Выгода",
  diagnostic: "Диагностика",
  action: "Оплатить и начать",
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
