"use client";

import React, { useCallback, useEffect, useMemo, useState } from "react";
import { api } from "@/lib/api";
import type { CampaignLink, CampaignLinkWithClicks, LinkReport } from "@/lib/api-types";
import { CAMPAIGN_LINKS_CHANGED_EVENT, shortLinkUrl } from "@/lib/campaign-links";
import {
  AlertTriangle,
  BarChart3,
  Check,
  Copy,
  ExternalLink,
  Link2,
  Power,
  PowerOff,
  RefreshCw,
  Trash2,
} from "lucide-react";

const PERIODS = [7, 14, 30] as const;
type Period = (typeof PERIODS)[number];

// The backend reads date-only from/to in Moscow time, so the browser must not
// derive the window from its own timezone.
const MSK_OFFSET_MS = 3 * 60 * 60 * 1000;

function moscowDate(date: Date): string {
  return new Date(date.getTime() + MSK_OFFSET_MS).toISOString().slice(0, 10);
}

function reportWindow(days: Period): { from: string; to: string } {
  const now = new Date();
  const start = new Date(now.getTime() - (days - 1) * 24 * 60 * 60 * 1000);
  return { from: moscowDate(start), to: moscowDate(now) };
}

function formatNumber(value: number): string {
  return value.toLocaleString("ru-RU");
}

function formatMoney(value: number, maximumFractionDigits = 0): string {
  return `${value.toLocaleString("ru-RU", { maximumFractionDigits })} ₽`;
}

function activeBadge(active: boolean) {
  return active ? (
    <span className="inline-flex items-center gap-1 rounded-full bg-emerald-500/15 px-2 py-0.5 text-[10px] font-bold text-emerald-600 dark:text-emerald-400">
      <span className="h-1.5 w-1.5 rounded-full bg-emerald-500" />
      активна
    </span>
  ) : (
    <span className="inline-flex items-center gap-1 rounded-full bg-slate-200 px-2 py-0.5 text-[10px] font-bold text-slate-500 dark:bg-white/10 dark:text-slate-400">
      <span className="h-1.5 w-1.5 rounded-full bg-slate-400" />
      выключена
    </span>
  );
}

export function LinksReport() {
  const [days, setDays] = useState<Period>(7);
  const [report, setReport] = useState<LinkReport | null>(null);
  const [links, setLinks] = useState<CampaignLinkWithClicks[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(false);
  const [copied, setCopied] = useState<string | null>(null);
  const [busyId, setBusyId] = useState<number | null>(null);
  const [actionError, setActionError] = useState<string | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<CampaignLink | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    setError(false);
    const [linksResult, reportResult] = await Promise.all([
      api.getLinks(),
      api.getLinksReport(reportWindow(days)),
    ]);
    setLinks(linksResult);
    setReport(reportResult);
    setError(!reportResult);
    setLoading(false);
  }, [days]);

  useEffect(() => {
    load();
  }, [load]);

  // A link created in the generator above should appear without a manual
  // refresh; both components stay independent through a DOM event.
  useEffect(() => {
    const onChanged = () => {
      load();
    };
    window.addEventListener(CAMPAIGN_LINKS_CHANGED_EVENT, onChanged);
    return () => window.removeEventListener(CAMPAIGN_LINKS_CHANGED_EVENT, onChanged);
  }, [load]);

  const linksBySlug = useMemo(
    () => new Map(links.map((link) => [link.slug, link])),
    [links]
  );

  const rows = report?.rows ?? [];
  const totals = rows.reduce(
    (acc, row) => ({
      clicks: acc.clicks + row.clicks,
      payments: acc.payments + row.payments,
      revenue: acc.revenue + row.revenue,
    }),
    { clicks: 0, payments: 0, revenue: 0 }
  );

  const handleCopy = (slug: string) => {
    if (!navigator?.clipboard) return;
    navigator.clipboard.writeText(shortLinkUrl(slug)).catch(() => {});
    setCopied(slug);
    window.setTimeout(() => setCopied(null), 2000);
  };

  const handleToggle = async (link: CampaignLink) => {
    setBusyId(link.id);
    setActionError(null);
    const res = await api.updateLink(link.id, { is_active: !link.is_active });
    setBusyId(null);
    if (!res.success) {
      setActionError(res.error || "Не удалось изменить статус ссылки");
      return;
    }
    await load();
  };

  const handleDelete = async () => {
    if (!deleteTarget) return;
    setBusyId(deleteTarget.id);
    setActionError(null);
    const res = await api.deleteLink(deleteTarget.id);
    setBusyId(null);
    if (!res.success) {
      setActionError(res.error || "Не удалось удалить ссылку");
      return;
    }
    setDeleteTarget(null);
    await load();
  };

  const handleDeactivateInstead = async () => {
    const target = deleteTarget;
    setDeleteTarget(null);
    if (target?.is_active) {
      await handleToggle(target);
    }
  };

  return (
    <div className="rounded-3xl border border-slate-200 dark:border-white/10 bg-white dark:bg-white/5 p-6 mb-6">
      <div className="flex flex-wrap items-start justify-between gap-3 mb-4">
        <div className="flex items-start gap-2">
          <span className="p-2 rounded-xl bg-emerald-500/10 text-emerald-600 dark:text-emerald-400">
            <BarChart3 className="w-5 h-5" />
          </span>
          <div>
            <h3 className="font-semibold text-lg text-slate-900 dark:text-white">
              Отчёт по коротким ссылкам
            </h3>
            <p className="text-xs text-slate-500 dark:text-slate-400">
              Переходы, визиты, лиды и оплаты по каждой ссылке. Деньги — только успешные оплаты, атрибуция по первому
              клику.
            </p>
          </div>
        </div>

        <div className="flex flex-wrap items-center gap-2">
          <div className="inline-flex rounded-xl border border-slate-200 dark:border-white/10 p-0.5 text-xs font-mono bg-slate-50 dark:bg-white/[0.03]">
            {PERIODS.map((period) => (
              <button
                key={period}
                type="button"
                onClick={() => setDays(period)}
                className={`px-3 py-1.5 rounded-lg transition-colors ${
                  days === period
                    ? "bg-slate-900 text-white dark:bg-white dark:text-slate-900 font-bold"
                    : "text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-white"
                }`}
              >
                {period} дн.
              </button>
            ))}
          </div>
          <button
            type="button"
            onClick={load}
            disabled={loading}
            className="inline-flex items-center gap-1.5 rounded-xl bg-emerald-600 hover:bg-emerald-500 text-white px-3 py-1.5 text-xs font-medium transition-colors disabled:opacity-60"
            title="Обновить отчёт"
          >
            <RefreshCw className={`w-3.5 h-3.5 ${loading ? "animate-spin" : ""}`} />
            <span>{loading ? "Загрузка..." : "Обновить"}</span>
          </button>
        </div>
      </div>

      {actionError && (
        <div className="mb-4 flex items-center justify-between gap-3 rounded-2xl border border-rose-500/30 bg-rose-500/10 px-4 py-3 text-xs text-rose-600 dark:text-rose-300">
          <span>{actionError}</span>
          <button type="button" onClick={() => setActionError(null)} className="font-bold text-rose-500">
            ✕
          </button>
        </div>
      )}

      {error && <p className="text-sm text-red-500">Не удалось загрузить отчёт по ссылкам.</p>}

      {!error && loading && !report && <p className="text-sm text-slate-500">Загрузка отчёта...</p>}

      {!error && report && rows.length === 0 && (
        <div className="py-10 text-center">
          <Link2 className="w-8 h-8 mx-auto mb-3 text-slate-300 dark:text-slate-600" />
          <p className="text-sm font-medium text-slate-600 dark:text-slate-300">Пока нет переходов.</p>
          <p className="mt-1 text-xs text-slate-500">
            Создайте короткую ссылку в генераторе выше — она появится здесь с первого клика.
          </p>
        </div>
      )}

      {!error && report && rows.length > 0 && (
        <>
          <div className="mb-4 flex flex-wrap items-center gap-x-5 gap-y-1 text-xs text-slate-500">
            <span>
              Ссылок:{" "}
              <b className="font-mono text-slate-800 dark:text-slate-200">{formatNumber(rows.length)}</b>
            </span>
            <span>
              Переходов:{" "}
              <b className="font-mono text-slate-800 dark:text-slate-200">{formatNumber(totals.clicks)}</b>
            </span>
            <span>
              Оплат: <b className="font-mono text-slate-800 dark:text-slate-200">{formatNumber(totals.payments)}</b>
            </span>
            <span>
              Выручка по ссылкам:{" "}
              <b className="font-mono text-emerald-600 dark:text-emerald-400">{formatMoney(totals.revenue)}</b>
            </span>
          </div>

          {totals.clicks === 0 && (
            <p className="mb-3 rounded-xl bg-slate-50 px-3 py-2 text-xs text-slate-500 dark:bg-white/[0.03]">
              Пока нет переходов за выбранный период. Ниже — все ссылки с нулевыми показателями.
            </p>
          )}

          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-slate-200 dark:border-white/10 text-left text-xs text-slate-500">
                  <th className="py-2 pr-3 font-medium">Ссылка</th>
                  <th className="py-2 px-3 text-right font-medium">Переходы</th>
                  <th className="py-2 px-3 text-right font-medium">Уникальные</th>
                  <th className="py-2 px-3 text-right font-medium">Визиты на сайт</th>
                  <th className="py-2 px-3 text-right font-medium">Лиды</th>
                  <th className="py-2 px-3 text-right font-medium">Оплаты</th>
                  <th className="py-2 px-3 text-right font-medium">Выручка</th>
                  <th
                    className="py-2 px-3 text-right font-medium"
                    title="Выручка на переход; ниже — выручка на уникального посетителя (RPV)"
                  >
                    EPC
                  </th>
                  <th className="py-2 pl-3 text-right font-medium">Действия</th>
                </tr>
              </thead>
              <tbody>
                {rows.map((row) => {
                  const link = linksBySlug.get(row.slug);
                  const inactive = Boolean(link && !link.is_active);
                  return (
                    <tr
                      key={row.slug}
                      className={`border-b border-slate-100 last:border-0 dark:border-white/5 ${
                        inactive ? "opacity-60" : ""
                      }`}
                    >
                      <td className="py-3 pr-3">
                        <div className="flex flex-wrap items-center gap-2">
                          <span className="font-medium text-slate-900 dark:text-white">
                            {row.title || link?.title || row.slug}
                          </span>
                          {link && activeBadge(link.is_active)}
                        </div>
                        <div className="mt-1 flex min-w-0 items-center gap-1.5 text-xs font-mono text-slate-500">
                          <span className="truncate" title={shortLinkUrl(row.slug)}>/r/{row.slug}</span>
                          <button
                            type="button"
                            onClick={() => handleCopy(row.slug)}
                            className="rounded p-0.5 text-slate-400 hover:text-slate-700 dark:hover:text-white transition-colors"
                            title="Скопировать короткую ссылку"
                            aria-label={`Скопировать короткую ссылку /r/${row.slug}`}
                          >
                            {copied === row.slug ? (
                              <Check className="w-3.5 h-3.5 text-emerald-500" />
                            ) : (
                              <Copy className="w-3.5 h-3.5" />
                            )}
                          </button>
                          <a
                            href={shortLinkUrl(row.slug)}
                            target="_blank"
                            rel="noopener noreferrer"
                            className="rounded p-0.5 text-slate-400 hover:text-slate-700 dark:hover:text-white transition-colors"
                            title="Открыть короткую ссылку"
                            aria-label={`Открыть короткую ссылку /r/${row.slug}`}
                          >
                            <ExternalLink className="w-3.5 h-3.5" />
                          </a>
                        </div>
                      </td>
                      <td className="py-3 px-3 text-right font-mono">{formatNumber(row.clicks)}</td>
                      <td className="py-3 px-3 text-right font-mono">{formatNumber(row.unique_visitors)}</td>
                      <td className="py-3 px-3 text-right font-mono">{formatNumber(row.site_visits)}</td>
                      <td className="py-3 px-3 text-right font-mono">{formatNumber(row.leads)}</td>
                      <td className="py-3 px-3 text-right font-mono font-semibold">{formatNumber(row.payments)}</td>
                      <td className="py-3 px-3 text-right font-mono font-semibold text-emerald-600 dark:text-emerald-400">
                        {formatMoney(row.revenue)}
                      </td>
                      <td className="py-3 px-3 text-right font-mono">
                        <div>{row.epc > 0 ? formatMoney(row.epc, 2) : "—"}</div>
                        <div className="text-[10px] text-slate-400">
                          RPV {row.rpv > 0 ? formatMoney(row.rpv, 2) : "—"}
                        </div>
                      </td>
                      <td className="py-3 pl-3 text-right">
                        {link ? (
                          <div className="flex items-center justify-end gap-1.5">
                            <button
                              type="button"
                              onClick={() => handleToggle(link)}
                              disabled={busyId === link.id}
                              className={`inline-flex items-center gap-1 rounded-lg border px-2.5 py-1 text-[11px] font-bold transition-colors disabled:opacity-50 ${
                                link.is_active
                                  ? "border-amber-500/30 bg-amber-500/10 text-amber-700 hover:bg-amber-500/20 dark:text-amber-300"
                                  : "border-emerald-500/30 bg-emerald-500/10 text-emerald-700 hover:bg-emerald-500/20 dark:text-emerald-300"
                              }`}
                              title={
                                link.is_active
                                  ? "Ссылка перестанет работать, история сохранится"
                                  : "Ссылка снова начнёт работать"
                              }
                            >
                              {link.is_active ? (
                                <PowerOff className="w-3.5 h-3.5" />
                              ) : (
                                <Power className="w-3.5 h-3.5" />
                              )}
                              <span>{link.is_active ? "Деактивировать" : "Включить"}</span>
                            </button>
                            <button
                              type="button"
                              onClick={() => setDeleteTarget(link)}
                              disabled={busyId === link.id}
                              className="rounded-lg border border-rose-500/30 p-1.5 text-rose-600 hover:bg-rose-500/10 disabled:opacity-50 dark:text-rose-300"
                              title="Удалить навсегда"
                              aria-label={`Удалить ссылку /r/${row.slug} навсегда`}
                            >
                              <Trash2 className="w-3.5 h-3.5" />
                            </button>
                          </div>
                        ) : (
                          <span className="text-xs text-slate-400">—</span>
                        )}
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        </>
      )}

      {!error && report && (
        <div className="mt-4 flex items-start gap-2 rounded-2xl border border-amber-500/25 bg-amber-500/5 px-4 py-3 text-xs text-slate-600 dark:text-slate-300">
          <AlertTriangle className="mt-0.5 w-4 h-4 shrink-0 text-amber-500" />
          <p>
            Вне ссылок:{" "}
            <span className="font-mono font-semibold text-amber-700 dark:text-amber-300">
              {formatMoney(report.unattributed_revenue)}
            </span>{" "}
            — оплаты посетителей без клика по короткой ссылке за период. Эта сумма не входит в строки отчёта и не
            делится между ссылками.
          </p>
        </div>
      )}

      {deleteTarget && (
        <div
          className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 p-4 backdrop-blur-sm"
          role="dialog"
          aria-modal="true"
          aria-labelledby="delete-campaign-link-title"
        >
          <div className="w-full max-w-md space-y-4 rounded-2xl border border-slate-200 bg-white p-6 text-slate-900 shadow-2xl dark:border-white/20 dark:bg-[#12161D] dark:text-white">
            <div className="flex items-center gap-2">
              <AlertTriangle className="w-5 h-5 text-rose-500" />
              <h3 id="delete-campaign-link-title" className="text-lg font-bold">
                Удалить ссылку навсегда?
              </h3>
            </div>
            <p className="text-xs text-slate-600 dark:text-slate-400">
              Ссылка <span className="font-mono">/r/{deleteTarget.slug}</span> и все её переходы будут удалены. Это
              необратимо и задним числом изменит прошлые периоды отчёта.
            </p>

            <div className="rounded-xl border border-amber-500/30 bg-amber-500/10 p-3 text-xs text-amber-800 dark:text-amber-200">
              <p className="font-semibold">Обычно удаление не нужно.</p>
              <p className="mt-1">
                Деактивация останавливает ссылку, но сохраняет историю переходов, лидов и оплат. Используйте её, если
                кампания просто закончилась.
              </p>
              <button
                type="button"
                onClick={handleDeactivateInstead}
                className="mt-2 inline-flex items-center gap-1.5 rounded-lg border border-amber-500/40 bg-white/60 px-2.5 py-1 text-[11px] font-bold text-amber-800 hover:bg-white dark:bg-black/20 dark:text-amber-200 dark:hover:bg-black/30"
              >
                <PowerOff className="w-3.5 h-3.5" />
                <span>{deleteTarget.is_active ? "Деактивировать вместо удаления" : "Оставить выключенной"}</span>
              </button>
            </div>

            <div className="flex gap-3 text-xs font-mono">
              <button
                type="button"
                onClick={() => setDeleteTarget(null)}
                className="flex-1 rounded-xl border border-slate-200 px-4 py-2 dark:border-white/10"
              >
                Отмена
              </button>
              <button
                type="button"
                onClick={handleDelete}
                disabled={busyId === deleteTarget.id}
                className="flex-1 rounded-xl bg-rose-500 px-4 py-2 font-bold text-white hover:bg-rose-400 disabled:opacity-50"
              >
                {busyId === deleteTarget.id ? "Удаление..." : "Всё равно удалить"}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
