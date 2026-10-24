"use client";

import React, { useState } from "react";
import { Link2, Copy, Check, ExternalLink, Sparkles, Send } from "lucide-react";

export function CampaignLinkGenerator() {
  const [targetType, setTargetType] = useState<"landing" | "pricing" | "calc" | "bot">("landing");
  const [source, setSource] = useState("");
  const [campaign, setCampaign] = useState("");
  const [promo, setPromo] = useState("");
  const [tier, setTier] = useState<"accelerator" | "vip" | "basic">("accelerator");
  const [copied, setCopied] = useState<string | null>(null);

  // Derive production base URL
  const baseUrl = typeof window !== "undefined" ? window.location.origin : "https://spark-school.ru";

  // Build Web Landing URL
  const buildWebUrl = () => {
    let path = "/";
    if (targetType === "pricing") path = "/#pricing";
    if (targetType === "calc") path = "/#calculator";

    const params = new URLSearchParams();
    const cleanSource = source.trim().toLowerCase().replace(/[^a-z0-9_-]/g, "");
    const cleanCampaign = campaign.trim().toLowerCase().replace(/[^a-z0-9_-]/g, "");
    const cleanPromo = promo.trim().toUpperCase().replace(/[^A-Z0-9_-]/g, "");

    if (cleanSource) params.set("utm_source", cleanSource);
    params.set("utm_medium", targetType === "bot" ? "bot" : "cpa");
    if (cleanCampaign) params.set("utm_campaign", cleanCampaign);
    if (cleanPromo) params.set("promo", cleanPromo);

    const qs = params.toString();
    return `${baseUrl}${path}${qs ? `?${qs}` : ""}`;
  };

  // Build Telegram Bot Deep Link
  const buildBotUrl = () => {
    const cleanSource = source.trim().toLowerCase().replace(/[^a-z0-9_-]/g, "");
    const cleanPromo = promo.trim().toUpperCase().replace(/[^A-Z0-9_-]/g, "");

    let startPayload = "";

    if (targetType === "bot") {
      startPayload = cleanSource ? `src_${cleanSource}` : "";
    } else {
      // pay deep-link
      startPayload = `pay_${tier}`;
      if (cleanPromo) {
        startPayload += `_${cleanPromo}`;
      }
      if (cleanSource) {
        startPayload += `_src_${cleanSource}`;
      }
    }

    // Telegram allows max 64 chars
    const param = startPayload.slice(0, 64);
    return param ? `https://t.me/spark_prep_bot?start=${param}` : "https://t.me/spark_prep_bot";
  };

  const webUrl = buildWebUrl();
  const botUrl = buildBotUrl();

  const handleCopy = (text: string, type: string) => {
    if (navigator?.clipboard) {
      navigator.clipboard.writeText(text);
      setCopied(type);
      setTimeout(() => setCopied(null), 2000);
    }
  };

  return (
    <div className="rounded-3xl border border-slate-200 dark:border-white/10 bg-white dark:bg-white/5 p-6 mb-6">
      <div className="flex flex-wrap items-center justify-between gap-3 mb-5">
        <div>
          <div className="flex items-center gap-2">
            <span className="p-2 rounded-xl bg-sky-500/10 text-sky-600 dark:text-sky-400">
              <Link2 className="w-5 h-5" />
            </span>
            <div>
              <h3 className="font-semibold text-lg text-slate-900 dark:text-white">
                Генератор рекламных ссылок и меток
              </h3>
              <p className="text-xs text-slate-500 dark:text-slate-400">
                Создание ссылок для рекламы у блогеров, посевов в Telegram и таргета с автоматической сквозной атрибуцией до покупки.
              </p>
            </div>
          </div>
        </div>
      </div>

      {/* Input Parameters Grid */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-6">
        <div>
          <label className="block text-xs font-mono text-slate-600 dark:text-slate-400 mb-1.5 font-medium">
            Точка входа (куда ведём)
          </label>
          <select
            value={targetType}
            onChange={(e) => setTargetType(e.target.value as any)}
            className="w-full text-xs font-mono rounded-xl border border-slate-200 dark:border-white/10 bg-slate-50 dark:bg-white/5 px-3 py-2 text-slate-900 dark:text-white focus:outline-none focus:border-sky-500"
          >
            <option value="landing">Главная страница (Лендинг)</option>
            <option value="pricing">Секция тарифов (#pricing)</option>
            <option value="calc">Калькулятор шансов (#calculator)</option>
            <option value="bot">Сразу в Telegram-бота (Диплинк)</option>
          </select>
        </div>

        <div>
          <label className="block text-xs font-mono text-slate-600 dark:text-slate-400 mb-1.5 font-medium">
            Канал / Блогер (utm_source) *
          </label>
          <input
            type="text"
            placeholder="например: nemtsov, tg_study_usa"
            value={source}
            onChange={(e) => setSource(e.target.value)}
            className="w-full text-xs font-mono rounded-xl border border-slate-200 dark:border-white/10 bg-slate-50 dark:bg-white/5 px-3 py-2 text-slate-900 dark:text-white focus:outline-none focus:border-sky-500"
          />
        </div>

        <div>
          <label className="block text-xs font-mono text-slate-600 dark:text-slate-400 mb-1.5 font-medium">
            Кампания (utm_campaign)
          </label>
          <input
            type="text"
            placeholder="например: post_october, stories_1"
            value={campaign}
            onChange={(e) => setCampaign(e.target.value)}
            className="w-full text-xs font-mono rounded-xl border border-slate-200 dark:border-white/10 bg-slate-50 dark:bg-white/5 px-3 py-2 text-slate-900 dark:text-white focus:outline-none focus:border-sky-500"
          />
        </div>

        <div>
          <label className="block text-xs font-mono text-slate-600 dark:text-slate-400 mb-1.5 font-medium">
            Промокод (если есть)
          </label>
          <input
            type="text"
            placeholder="например: SPARK5"
            value={promo}
            onChange={(e) => setPromo(e.target.value)}
            className="w-full text-xs font-mono rounded-xl border border-slate-200 dark:border-white/10 bg-slate-50 dark:bg-white/5 px-3 py-2 text-slate-900 dark:text-white focus:outline-none focus:border-sky-500 uppercase"
          />
        </div>
      </div>

      {targetType !== "bot" && (
        <div className="mb-6 flex items-center gap-2">
          <span className="text-xs font-mono text-slate-500">Выбранный тариф для быстрой ссылки:</span>
          <div className="inline-flex rounded-lg border border-slate-200 dark:border-white/10 p-0.5 text-xs font-mono">
            <button
              type="button"
              onClick={() => setTier("basic")}
              className={`px-2.5 py-1 rounded-md transition-colors ${tier === "basic" ? "bg-slate-900 text-white dark:bg-white dark:text-slate-900 font-bold" : "text-slate-600 dark:text-slate-400"}`}
            >
              Базовый
            </button>
            <button
              type="button"
              onClick={() => setTier("accelerator")}
              className={`px-2.5 py-1 rounded-md transition-colors ${tier === "accelerator" ? "bg-emerald-600 text-white font-bold" : "text-slate-600 dark:text-slate-400"}`}
            >
              Акселератор
            </button>
            <button
              type="button"
              onClick={() => setTier("vip")}
              className={`px-2.5 py-1 rounded-md transition-colors ${tier === "vip" ? "bg-amber-600 text-white font-bold" : "text-slate-600 dark:text-slate-400"}`}
            >
              VIP
            </button>
          </div>
        </div>
      )}

      {/* Generated Results */}
      <div className="space-y-3 pt-4 border-t border-slate-100 dark:border-white/5 font-mono">
        {/* Link for Web / Landing */}
        <div className="p-3.5 rounded-2xl bg-slate-50 dark:bg-white/[0.03] border border-slate-200/80 dark:border-white/10 flex flex-col sm:flex-row items-start sm:items-center justify-between gap-3">
          <div className="flex-1 min-w-0">
            <div className="flex items-center gap-2 mb-1">
              <span className="text-[10px] font-bold uppercase tracking-wider px-2 py-0.5 rounded bg-sky-500/15 text-sky-700 dark:text-sky-300">
                Ссылка на сайт (Лендинг)
              </span>
              <span className="text-[11px] text-slate-500">Для постов, баннеров, сторис и рекламы</span>
            </div>
            <p className="text-xs text-slate-800 dark:text-slate-200 truncate select-all">
              {webUrl}
            </p>
          </div>
          <div className="flex items-center gap-2 shrink-0">
            <button
              type="button"
              onClick={() => handleCopy(webUrl, "web")}
              className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-xl bg-white dark:bg-white/10 hover:bg-slate-100 dark:hover:bg-white/15 text-xs font-medium text-slate-800 dark:text-slate-200 border border-slate-200 dark:border-white/10 shadow-sm transition-all active:scale-95"
            >
              {copied === "web" ? <Check className="w-3.5 h-3.5 text-emerald-500" /> : <Copy className="w-3.5 h-3.5" />}
              <span>{copied === "web" ? "Скопировано!" : "Копировать"}</span>
            </button>
            <a
              href={webUrl}
              target="_blank"
              rel="noopener noreferrer"
              className="p-1.5 rounded-xl text-slate-400 hover:text-slate-600 dark:hover:text-slate-200 transition-colors"
              title="Открыть ссылку в новой вкладке"
            >
              <ExternalLink className="w-4 h-4" />
            </a>
          </div>
        </div>

        {/* Link for Direct Bot */}
        <div className="p-3.5 rounded-2xl bg-slate-50 dark:bg-white/[0.03] border border-slate-200/80 dark:border-white/10 flex flex-col sm:flex-row items-start sm:items-center justify-between gap-3">
          <div className="flex-1 min-w-0">
            <div className="flex items-center gap-2 mb-1">
              <span className="text-[10px] font-bold uppercase tracking-wider px-2 py-0.5 rounded bg-emerald-500/15 text-emerald-700 dark:text-emerald-300">
                Прямая ссылка в Telegram-бота
              </span>
              <span className="text-[11px] text-slate-500">Фиксирует источник прямо при старте бота</span>
            </div>
            <p className="text-xs text-slate-800 dark:text-slate-200 truncate select-all">
              {botUrl}
            </p>
          </div>
          <div className="flex items-center gap-2 shrink-0">
            <button
              type="button"
              onClick={() => handleCopy(botUrl, "bot")}
              className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-xl bg-white dark:bg-white/10 hover:bg-slate-100 dark:hover:bg-white/15 text-xs font-medium text-slate-800 dark:text-slate-200 border border-slate-200 dark:border-white/10 shadow-sm transition-all active:scale-95"
            >
              {copied === "bot" ? <Check className="w-3.5 h-3.5 text-emerald-500" /> : <Copy className="w-3.5 h-3.5" />}
              <span>{copied === "bot" ? "Скопировано!" : "Копировать"}</span>
            </button>
            <a
              href={botUrl}
              target="_blank"
              rel="noopener noreferrer"
              className="p-1.5 rounded-xl text-slate-400 hover:text-slate-600 dark:hover:text-slate-200 transition-colors"
              title="Открыть в Telegram"
            >
              <Send className="w-4 h-4" />
            </a>
          </div>
        </div>
      </div>
    </div>
  );
}
