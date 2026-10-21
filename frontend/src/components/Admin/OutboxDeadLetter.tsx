"use client";

import React, { useCallback, useEffect, useState } from "react";
import { RefreshCw } from "lucide-react";
import { api, DeadOutboxEvent } from "@/lib/api";

export function OutboxDeadLetter() {
  const [events, setEvents] = useState<DeadOutboxEvent[]>([]);
  const [error, setError] = useState(false);
  const [loading, setLoading] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    const result = await api.getDeadOutbox();
    setEvents(result || []);
    setError(!result);
    setLoading(false);
  }, []);

  useEffect(() => {
    load();
  }, [load]);

  return (
    <div className="rounded-3xl border border-slate-200 dark:border-white/10 bg-white dark:bg-white/5 p-6 mb-6">
      <div className="flex items-center justify-between mb-4">
        <div>
          <h3 className="font-semibold text-lg">Outbox: недоставленные события</h3>
          <p className="text-xs text-slate-500">События после исчерпания retry. Можно повторить вручную.</p>
        </div>
        <button onClick={load} className="inline-flex items-center gap-1 rounded-lg bg-emerald-600 text-white px-3 py-1 text-sm">
          <RefreshCw className={`w-3.5 h-3.5 ${loading ? "animate-spin" : ""}`} /> Обновить
        </button>
      </div>
      {error && <p className="text-sm text-red-500">Не удалось загрузить outbox.</p>}
      {!error && events.length === 0 && <p className="text-sm text-slate-500">Недоставленных событий нет.</p>}
      {events.length > 0 && (
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="text-left text-slate-500 border-b">
                <th className="py-2">ID</th>
                <th className="py-2">Тип</th>
                <th className="py-2">Попытки</th>
                <th className="py-2">Ошибка</th>
                <th className="py-2">Создано</th>
                <th className="py-2" />
              </tr>
            </thead>
            <tbody>
              {events.map((event) => (
                <tr key={event.id} className="border-b last:border-0 align-top">
                  <td className="py-2 font-mono">{event.id}</td>
                  <td className="py-2">{event.event_type}</td>
                  <td className="py-2 font-mono">{event.retry_count}/{event.max_retries}</td>
                  <td className="py-2 max-w-xs truncate text-xs text-rose-500" title={event.last_error}>{event.last_error || "—"}</td>
                  <td className="py-2 text-xs">{new Date(event.created_at).toLocaleString("ru-RU")}</td>
                  <td className="py-2">
                    <button
                      onClick={async () => {
                        if (await api.retryOutbox(event.id)) load();
                      }}
                      className="rounded-lg border px-2 py-1 text-xs"
                    >
                      Повторить
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
