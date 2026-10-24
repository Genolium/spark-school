"use client";

import React, { useState, useEffect } from "react";
import { api, CoursePricing } from "@/lib/api";
import { DEFAULT_PRICING } from "@/lib/pricing-defaults";
import {
  DollarSign,
  Save,
  RotateCcw,
  CheckCircle2,
  AlertCircle,
  Sparkles,
  Crown,
  ArrowRight,
  TrendingUp,
  Bot,
  Globe,
  Building2,
} from "lucide-react";

export function PricingSettingsTable() {
  const [pricing, setPricing] = useState<CoursePricing>({ ...DEFAULT_PRICING });

  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [msg, setMsg] = useState<{ text: string; type: "success" | "error" } | null>(null);

  useEffect(() => {
    loadPricing();
  }, []);

  const loadPricing = async () => {
    setLoading(true);
    const data = await api.getCoursePricing();
    setPricing({
      ...data,
      basic_price: data.basic_price || 3990,
      basic_old_price: data.basic_old_price || 5900,
    });
    setLoading(false);
  };

  const handleSave = async (e: React.FormEvent) => {
    e.preventDefault();
    setSaving(true);
    setMsg(null);

    const bPrice = pricing.basic_price || 3990;

    // Frontend validation
    if (bPrice <= 0) {
      setMsg({ text: "Цена тарифа «Базовый» должна быть положительным числом", type: "error" });
      setSaving(false);
      return;
    }

    if (pricing.accelerator_price <= 0) {
      setMsg({ text: "Цена тарифа «Акселератор» должна быть положительным числом", type: "error" });
      setSaving(false);
      return;
    }

    if (pricing.vip_price <= 0) {
      setMsg({ text: "Цена тарифа «VIP» должна быть положительным числом", type: "error" });
      setSaving(false);
      return;
    }

    if (pricing.accelerator_price <= bPrice) {
      setMsg({ text: "Цена тарифа «Акселератор» должна быть выше тарифа «Базовый»", type: "error" });
      setSaving(false);
      return;
    }

    if (pricing.vip_price <= pricing.accelerator_price) {
      setMsg({ text: "Цена тарифа «VIP» должна быть выше тарифа «Акселератор»", type: "error" });
      setSaving(false);
      return;
    }

    const bOld = pricing.basic_old_price || 0;
    const accOld = pricing.accelerator_old_price || 0;
    const vipOld = pricing.vip_old_price || 0;

    const payload = {
      ...pricing,
      basic_price: bPrice,
      basic_old_price: bOld,
      accelerator_old_price: accOld,
      vip_old_price: vipOld,
      bank_name: (pricing.bank_name || "").trim() || "БСПБ (Банк «Санкт-Петербург»)",
    };

    const res = await api.updateCoursePricing(payload);
    setSaving(false);

    if (res.success) {
      setMsg({ text: "Цены и реквизиты успешно обновлены и синхронизированы на сайте и в боте!", type: "success" });
      if (res.pricing) {
        setPricing({
          ...res.pricing,
          basic_price: res.pricing.basic_price || 3990,
          basic_old_price: res.pricing.basic_old_price || 5900,
        });
      }
      setTimeout(() => setMsg(null), 5000);
    } else {
      setMsg({ text: res.error || "Ошибка сохранения настроек", type: "error" });
    }
  };

  const handleResetDefaults = () => {
    setPricing({ ...DEFAULT_PRICING });
    setMsg({ text: "Установлены стандартные значения (3 990 / 10 900 / 14 900 ₽). Нажмите «Сохранить», чтобы применить.", type: "success" });
  };

  const formatRub = (num: number) => {
    return (num || 0).toString().replace(/\B(?=(\d{3})+(?!\d))/g, " ");
  };

  const handlePriceChange = (field: keyof CoursePricing, rawValue: string) => {
    // Strip non-digits and leading zeros (e.g. "07800" -> "7800")
    const clean = rawValue.replace(/\D/g, "").replace(/^0+(?=\d)/, "");
    const num = clean === "" ? 0 : parseInt(clean, 10);
    setPricing((prev) => ({
      ...prev,
      [field]: isNaN(num) ? 0 : num,
    }));
  };

  // Calculations for promo discount preview
  const basicPromo = Math.round((pricing.basic_price || 3990) * 0.95);
  const basicPromoDiff = (pricing.basic_price || 3990) - basicPromo;

  const accPromo = Math.round(pricing.accelerator_price * 0.95);
  const accPromoDiff = pricing.accelerator_price - accPromo;

  const vipPromo = Math.round(pricing.vip_price * 0.95);
  const vipPromoDiff = pricing.vip_price - vipPromo;

  const basicToAccDiff = Math.max(0, pricing.accelerator_price - (pricing.basic_price || 3990));
  const accToVipDiff = Math.max(0, pricing.vip_price - pricing.accelerator_price);
  const basicToVipDiff = Math.max(0, pricing.vip_price - (pricing.basic_price || 3990));

  return (
    <div className="space-y-8">
      {/* Top Banner / Sync Info */}
      <div className="p-6 rounded-[28px] bg-white dark:bg-[#12161f] border border-slate-200 dark:border-white/10 shadow-sm flex flex-col md:flex-row md:items-center justify-between gap-4">
        <div>
          <div className="flex items-center gap-2 mb-1">
            <span className="w-2.5 h-2.5 rounded-full bg-emerald-500 animate-pulse" />
            <h2 className="text-lg sm:text-xl font-bold font-editorial text-slate-900 dark:text-white">
              Управление стоимостью курса
            </h2>
          </div>
          <p className="text-xs font-mono text-slate-600 dark:text-slate-400">
            Изменения вступают в силу немедленно на лендинге и во всех платежных сценариях Telegram-бота.
          </p>
        </div>

        <div className="flex items-center gap-3 text-xs font-mono">
          <div className="flex items-center gap-1.5 px-3 py-1.5 rounded-full bg-emerald-500/10 text-emerald-700 dark:text-emerald-300 border border-emerald-500/20">
            <Globe className="w-3.5 h-3.5" />
            <span>Сайт</span>
          </div>
          <div className="flex items-center gap-1.5 px-3 py-1.5 rounded-full bg-sky-500/10 text-sky-700 dark:text-sky-300 border border-sky-500/20">
            <Bot className="w-3.5 h-3.5" />
            <span>Telegram Bot</span>
          </div>
        </div>
      </div>

      {msg && (
        <div
          className={`p-4 rounded-2xl text-xs font-mono flex items-center justify-between transition-all ${
            msg.type === "success"
              ? "bg-emerald-500/10 border border-emerald-500/30 text-emerald-800 dark:text-emerald-300"
              : "bg-rose-500/10 border border-rose-500/30 text-rose-800 dark:text-rose-300"
          }`}
        >
          <div className="flex items-center gap-2">
            {msg.type === "success" ? (
              <CheckCircle2 className="w-4 h-4 shrink-0" />
            ) : (
              <AlertCircle className="w-4 h-4 shrink-0" />
            )}
            <span>{msg.text}</span>
          </div>
          <button
            onClick={() => setMsg(null)}
            className="text-slate-400 hover:text-slate-600 dark:hover:text-white font-bold ml-4"
          >
            ✕
          </button>
        </div>
      )}

      {/* Main Pricing Cards Grid */}
      <form onSubmit={handleSave} noValidate className="space-y-8">
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6 lg:gap-8">
          
          {/* ========================================================================= */}
          {/* CARD 1: ТАРИФ БАЗОВЫЙ                                                     */}
          {/* ========================================================================= */}
          <div className="p-6 sm:p-8 rounded-[32px] bg-white dark:bg-[#12161f] border border-slate-200 dark:border-white/10 shadow-sm relative overflow-hidden flex flex-col justify-between">
            <div className="absolute top-0 right-0 left-0 h-1.5 bg-gradient-to-r from-slate-400 to-slate-500" />
            
            <div>
              <div className="flex items-center justify-between mb-4">
                <span className="px-3 py-1 rounded-full text-xs font-mono font-medium uppercase bg-slate-100 dark:bg-white/10 text-slate-700 dark:text-slate-300 border border-slate-200 dark:border-white/10">
                  Self-Paced
                </span>
                <span className="text-[11px] font-mono text-rose-600 dark:text-rose-400 font-semibold">
                  Без обратной связи
                </span>
              </div>

              <h3 className="font-editorial text-2xl font-bold text-slate-900 dark:text-white mb-1">
                Тариф «Базовый»
              </h3>
              <p className="text-xs text-slate-500 dark:text-slate-400 mb-6">
                Видеоуроки, уроки текстом и их аудиоверсия без проверки заданий и мок-интервью.
              </p>

              {/* Input Fields */}
              <div className="space-y-4 mb-6">
                <div>
                  <label className="block text-xs font-mono font-bold text-slate-700 dark:text-slate-300 mb-1.5">
                    Текущая цена (₽) *
                  </label>
                  <div className="relative">
                    <input
                      type="text"
                      inputMode="numeric"
                      value={pricing.basic_price === 0 ? "" : (pricing.basic_price || 3990)}
                      placeholder="3990"
                      onChange={(e) => handlePriceChange("basic_price", e.target.value)}
                      className="w-full px-4 py-3 rounded-2xl bg-slate-50 dark:bg-black/30 border border-slate-300 dark:border-white/10 text-slate-900 dark:text-white font-mono text-base font-bold focus:outline-none focus:border-slate-500 transition-colors"
                    />
                    <span className="absolute right-4 top-1/2 -translate-y-1/2 text-xs font-mono text-slate-400">
                      RUB
                    </span>
                  </div>
                </div>

                <div>
                  <label className="block text-xs font-mono text-slate-600 dark:text-slate-400 mb-1.5">
                    Старая зачёркнутая цена (₽)
                  </label>
                  <div className="relative">
                    <input
                      type="text"
                      inputMode="numeric"
                      value={pricing.basic_old_price === 0 ? "" : (pricing.basic_old_price || 5900)}
                      placeholder="5900"
                      onChange={(e) => handlePriceChange("basic_old_price", e.target.value)}
                      className="w-full px-4 py-2.5 rounded-2xl bg-slate-50 dark:bg-black/30 border border-slate-300 dark:border-white/10 text-slate-700 dark:text-slate-300 font-mono text-sm focus:outline-none focus:border-slate-500 transition-colors"
                    />
                    <span className="absolute right-4 top-1/2 -translate-y-1/2 text-xs font-mono text-slate-400">
                      RUB
                    </span>
                  </div>
                </div>
              </div>

              {/* Live Preview Box */}
              <div className="p-4 rounded-2xl bg-slate-500/5 dark:bg-slate-500/10 border border-slate-500/20 text-xs font-mono space-y-2">
                <div className="flex items-center justify-between text-slate-500 dark:text-slate-400">
                  <span>Отображение на сайте:</span>
                  <div className="flex items-center gap-2">
                    <span className="line-through">{formatRub(pricing.basic_old_price || 5900)} ₽</span>
                    <span className="font-bold text-slate-900 dark:text-white text-sm">
                      {formatRub(pricing.basic_price || 3990)} ₽
                    </span>
                  </div>
                </div>

                <div className="flex items-center justify-between text-slate-700 dark:text-slate-300 pt-1 border-t border-slate-500/15">
                  <span>Со скидкой -5% (промокод):</span>
                  <span className="font-bold text-slate-900 dark:text-white">
                    {formatRub(basicPromo)} ₽ <span className="text-emerald-600 dark:text-emerald-400">(-{formatRub(basicPromoDiff)} ₽)</span>
                  </span>
                </div>
              </div>
            </div>
          </div>

          {/* ========================================================================= */}
          {/* CARD 2: ТАРИФ АКСЕЛЕРАТОР                                                 */}
          {/* ========================================================================= */}
          <div className="p-6 sm:p-8 rounded-[32px] bg-white dark:bg-[#12161f] border-2 border-emerald-500/40 shadow-lg relative overflow-hidden flex flex-col justify-between">
            <div className="absolute top-0 right-0 left-0 h-1.5 bg-gradient-to-r from-emerald-500 to-teal-400" />
            
            <div>
              <div className="flex items-center justify-between mb-4">
                <span className="px-3 py-1 rounded-full text-xs font-mono font-bold uppercase bg-emerald-500/15 text-emerald-700 dark:text-emerald-300 border border-emerald-500/30 flex items-center gap-1.5">
                  <Sparkles className="w-3.5 h-3.5 fill-current" />
                  Хит продаж
                </span>
                <span className="text-[11px] font-mono text-emerald-600 dark:text-emerald-400 font-semibold">
                  Основной тариф
                </span>
              </div>

              <h3 className="font-editorial text-2xl font-bold text-slate-900 dark:text-white mb-1">
                Тариф «Акселератор»
              </h3>
              <p className="text-xs text-slate-500 dark:text-slate-400 mb-6">
                Полный цикл подготовки: закрытый канал, аудит документов, мок-интервью 1-на-1 и визовый гайд.
              </p>

              {/* Input Fields */}
              <div className="space-y-4 mb-6">
                <div>
                  <label className="block text-xs font-mono font-bold text-slate-700 dark:text-slate-300 mb-1.5">
                    Текущая цена (₽) *
                  </label>
                  <div className="relative">
                    <input
                      type="text"
                      inputMode="numeric"
                      value={pricing.accelerator_price === 0 ? "" : pricing.accelerator_price}
                      placeholder="10900"
                      onChange={(e) => handlePriceChange("accelerator_price", e.target.value)}
                      className="w-full px-4 py-3 rounded-2xl bg-slate-50 dark:bg-black/30 border border-slate-300 dark:border-white/10 text-slate-900 dark:text-white font-mono text-base font-bold focus:outline-none focus:border-emerald-500 transition-colors"
                    />
                    <span className="absolute right-4 top-1/2 -translate-y-1/2 text-xs font-mono text-slate-400">
                      RUB
                    </span>
                  </div>
                </div>

                <div>
                  <label className="block text-xs font-mono text-slate-600 dark:text-slate-400 mb-1.5">
                    Старая зачёркнутая цена (₽)
                  </label>
                  <div className="relative">
                    <input
                      type="text"
                      inputMode="numeric"
                      value={pricing.accelerator_old_price === 0 ? "" : pricing.accelerator_old_price}
                      placeholder="9900"
                      onChange={(e) => handlePriceChange("accelerator_old_price", e.target.value)}
                      className="w-full px-4 py-2.5 rounded-2xl bg-slate-50 dark:bg-black/30 border border-slate-300 dark:border-white/10 text-slate-700 dark:text-slate-300 font-mono text-sm focus:outline-none focus:border-emerald-500 transition-colors"
                    />
                    <span className="absolute right-4 top-1/2 -translate-y-1/2 text-xs font-mono text-slate-400">
                      RUB
                    </span>
                  </div>
                </div>
              </div>

              {/* Live Preview Box */}
              <div className="p-4 rounded-2xl bg-emerald-500/5 dark:bg-emerald-500/10 border border-emerald-500/20 text-xs font-mono space-y-2">
                <div className="flex items-center justify-between text-slate-500 dark:text-slate-400">
                  <span>Отображение на сайте:</span>
                  <div className="flex items-center gap-2">
                    <span className="line-through">{formatRub(pricing.accelerator_old_price)} ₽</span>
                    <span className="font-bold text-emerald-600 dark:text-emerald-400 text-sm">
                      {formatRub(pricing.accelerator_price)} ₽
                    </span>
                  </div>
                </div>

                <div className="flex items-center justify-between text-slate-700 dark:text-slate-300 pt-1 border-t border-emerald-500/15">
                  <span>Со скидкой -5% (промокод):</span>
                  <span className="font-bold text-slate-900 dark:text-white">
                    {formatRub(accPromo)} ₽ <span className="text-emerald-600 dark:text-emerald-400">(-{formatRub(accPromoDiff)} ₽)</span>
                  </span>
                </div>
              </div>
            </div>
          </div>

          {/* ========================================================================= */}
          {/* CARD 2: ТАРИФ VIP                                                         */}
          {/* ========================================================================= */}
          <div className="p-6 sm:p-8 rounded-[32px] bg-white dark:bg-[#12161f] border-2 border-amber-500/40 shadow-lg relative overflow-hidden flex flex-col justify-between">
            <div className="absolute top-0 right-0 left-0 h-1.5 bg-gradient-to-r from-amber-500 to-orange-400" />
            
            <div>
              <div className="flex items-center justify-between mb-4">
                <span className="px-3 py-1 rounded-full text-xs font-mono font-bold uppercase bg-amber-500/15 text-amber-700 dark:text-amber-300 border border-amber-500/30 flex items-center gap-1.5">
                  <Crown className="w-3.5 h-3.5" />
                  Строго 3 места
                </span>
                <span className="text-[11px] font-mono text-amber-600 dark:text-amber-400 font-semibold">
                  Премиум
                </span>
              </div>

              <h3 className="font-editorial text-2xl font-bold text-slate-900 dark:text-white mb-1">
                Тариф «VIP»
              </h3>
              <p className="text-xs text-slate-500 dark:text-slate-400 mb-6">
                Максимальный персональный фокус: 3 мок-интервью и приоритетный личный чат до самого вылета в США.
              </p>

              {/* Input Fields */}
              <div className="space-y-4 mb-6">
                <div>
                  <label className="block text-xs font-mono font-bold text-slate-700 dark:text-slate-300 mb-1.5">
                    Текущая цена (₽) *
                  </label>
                  <div className="relative">
                    <input
                      type="text"
                      inputMode="numeric"
                      value={pricing.vip_price === 0 ? "" : pricing.vip_price}
                      placeholder="14900"
                      onChange={(e) => handlePriceChange("vip_price", e.target.value)}
                      className="w-full px-4 py-3 rounded-2xl bg-slate-50 dark:bg-black/30 border border-slate-300 dark:border-white/10 text-slate-900 dark:text-white font-mono text-base font-bold focus:outline-none focus:border-amber-500 transition-colors"
                    />
                    <span className="absolute right-4 top-1/2 -translate-y-1/2 text-xs font-mono text-slate-400">
                      RUB
                    </span>
                  </div>
                </div>

                <div>
                  <label className="block text-xs font-mono text-slate-600 dark:text-slate-400 mb-1.5">
                    Старая зачёркнутая цена (₽)
                  </label>
                  <div className="relative">
                    <input
                      type="text"
                      inputMode="numeric"
                      value={pricing.vip_old_price === 0 ? "" : pricing.vip_old_price}
                      placeholder="19900"
                      onChange={(e) => handlePriceChange("vip_old_price", e.target.value)}
                      className="w-full px-4 py-2.5 rounded-2xl bg-slate-50 dark:bg-black/30 border border-slate-300 dark:border-white/10 text-slate-700 dark:text-slate-300 font-mono text-sm focus:outline-none focus:border-amber-500 transition-colors"
                    />
                    <span className="absolute right-4 top-1/2 -translate-y-1/2 text-xs font-mono text-slate-400">
                      RUB
                    </span>
                  </div>
                </div>
              </div>

              {/* Live Preview Box */}
              <div className="p-4 rounded-2xl bg-amber-500/5 dark:bg-amber-500/10 border border-amber-500/20 text-xs font-mono space-y-2">
                <div className="flex items-center justify-between text-slate-500 dark:text-slate-400">
                  <span>Отображение на сайте:</span>
                  <div className="flex items-center gap-2">
                    <span className="line-through">{formatRub(pricing.vip_old_price)} ₽</span>
                    <span className="font-bold text-amber-600 dark:text-amber-400 text-sm">
                      {formatRub(pricing.vip_price)} ₽
                    </span>
                  </div>
                </div>

                <div className="flex items-center justify-between text-slate-700 dark:text-slate-300 pt-1 border-t border-amber-500/15">
                  <span>Со скидкой -5% (промокод):</span>
                  <span className="font-bold text-slate-900 dark:text-white">
                    {formatRub(vipPromo)} ₽ <span className="text-amber-600 dark:text-amber-400">(-{formatRub(vipPromoDiff)} ₽)</span>
                  </span>
                </div>
              </div>
            </div>
          </div>

        </div>

        {/* ========================================================================= */}
        {/* BANK SETTINGS CARD (Реквизиты банка для СБП и переводов)                  */}
        {/* ========================================================================= */}
        <div className="p-6 sm:p-8 rounded-[32px] bg-white dark:bg-[#12161f] border border-slate-200 dark:border-white/10 shadow-sm">
          <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 mb-6 pb-6 border-b border-slate-100 dark:border-white/5">
            <div className="flex items-center gap-3">
              <div className="w-10 h-10 rounded-2xl bg-indigo-500/10 text-indigo-600 dark:text-indigo-400 flex items-center justify-center shrink-0">
                <Building2 className="w-5 h-5" />
              </div>
              <div>
                <h3 className="font-editorial text-xl font-bold text-slate-900 dark:text-white">
                  Банк получателя для СБП / перевода
                </h3>
                <p className="text-xs text-slate-500 dark:text-slate-400 font-mono mt-0.5">
                  Отображается студенту в окне оплаты на сайте и во всех сообщениях Telegram-бота с инструкцией по переводу.
                </p>
              </div>
            </div>
            <div className="px-3 py-1 rounded-full text-xs font-mono bg-indigo-500/10 text-indigo-700 dark:text-indigo-300 border border-indigo-500/20 w-fit">
              СБП &amp; Прямой перевод
            </div>
          </div>

          <div className="space-y-4">
            <div>
              <label className="block text-xs font-mono font-bold text-slate-700 dark:text-slate-300 mb-2">
                Название банка получателя *
              </label>
              <input
                type="text"
                value={pricing.bank_name || ""}
                placeholder="БСПБ (Банк «Санкт-Петербург»)"
                onChange={(e) => setPricing((prev) => ({ ...prev, bank_name: e.target.value }))}
                className="w-full max-w-xl px-4 py-3 rounded-2xl bg-slate-50 dark:bg-black/30 border border-slate-300 dark:border-white/10 text-slate-900 dark:text-white font-mono text-sm font-semibold focus:outline-none focus:border-indigo-500 transition-colors"
              />
            </div>

            <div>
              <div className="text-[11px] font-mono text-slate-500 dark:text-slate-400 mb-2">
                Быстрый выбор популярного банка:
              </div>
              <div className="flex flex-wrap gap-2">
                {[
                  "БСПБ (Банк «Санкт-Петербург»)",
                  "Т-Банк (Тинькофф)",
                  "СберБанк",
                  "Альфа-Банк",
                  "ВТБ",
                  "Райффайзенбанк",
                ].map((b) => (
                  <button
                    key={b}
                    type="button"
                    onClick={() => setPricing((prev) => ({ ...prev, bank_name: b }))}
                    className={`px-3 py-1.5 rounded-xl text-xs font-mono transition-all ${
                      pricing.bank_name === b
                        ? "bg-indigo-600 text-white shadow-sm font-bold"
                        : "bg-slate-100 hover:bg-slate-200 dark:bg-white/5 dark:hover:bg-white/10 text-slate-700 dark:text-slate-300"
                    }`}
                  >
                    {b}
                  </button>
                ))}
              </div>
            </div>
          </div>
        </div>

        {/* Upgrade Math Auto-Preview */}
        <div className="p-6 rounded-[28px] bg-slate-900 text-white dark:bg-black/40 border border-slate-800 dark:border-white/10 shadow-md">
          <div className="flex flex-col lg:flex-row lg:items-center justify-between gap-6">
            <div className="flex items-start gap-3">
              <div className="w-10 h-10 rounded-xl bg-amber-500/20 text-amber-400 flex items-center justify-center font-bold shrink-0">
                <TrendingUp className="w-5 h-5" />
              </div>
              <div>
                <h4 className="font-bold font-editorial text-base sm:text-lg">
                  Автоматический расчет апгрейда в боте
                </h4>
                <p className="text-xs text-slate-400 font-mono mt-0.5">
                  Студенты могут повысить тариф в любой момент, оплатив только математическую разницу стоимости.
                </p>
              </div>
            </div>

            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 font-mono">
              <div className="p-3 rounded-2xl bg-white/5 border border-white/5">
                <div className="text-[11px] text-slate-400">Базовый ➔ Акселератор:</div>
                <div className="text-lg font-bold text-amber-400 mt-0.5">
                  +{formatRub(basicToAccDiff)} ₽
                </div>
                <div className="text-[10px] text-slate-500">
                  ({formatRub(pricing.accelerator_price)} − {formatRub(pricing.basic_price || 3990)})
                </div>
              </div>

              <div className="p-3 rounded-2xl bg-white/5 border border-white/5">
                <div className="text-[11px] text-slate-400">Акселератор ➔ VIP:</div>
                <div className="text-lg font-bold text-amber-400 mt-0.5">
                  +{formatRub(accToVipDiff)} ₽
                </div>
                <div className="text-[10px] text-slate-500">
                  ({formatRub(pricing.vip_price)} − {formatRub(pricing.accelerator_price)})
                </div>
              </div>
            </div>
          </div>
        </div>

        {/* Action Buttons */}
        <div className="flex flex-wrap items-center justify-between gap-4 pt-4">
          <button
            type="button"
            onClick={handleResetDefaults}
            className="px-5 py-3 rounded-2xl bg-slate-200/80 dark:bg-white/10 hover:bg-slate-300 dark:hover:bg-white/20 text-xs font-mono font-semibold text-slate-700 dark:text-slate-300 flex items-center gap-2 transition-all active:scale-95"
          >
            <RotateCcw className="w-4 h-4" />
            <span>Сбросить к рекомендованным (3 990 / 10 900 / 14 900 ₽)</span>
          </button>

          <button
            type="submit"
            disabled={saving || loading}
            className="px-8 py-4 rounded-2xl bg-amber-500 hover:bg-amber-400 text-black text-sm font-mono font-bold flex items-center gap-2.5 shadow-xl hover:shadow-amber-500/25 transition-all active:scale-95 disabled:opacity-50"
          >
            <Save className={`w-4 h-4 ${saving ? "animate-spin" : ""}`} />
            <span>{saving ? "Сохранение..." : "Сохранить изменения"}</span>
          </button>
        </div>
      </form>
    </div>
  );
}
