"use client";

import React, { useState, useEffect } from "react";
import { motion } from "framer-motion";
import { Check, ArrowUpRight, ShieldCheck, CreditCard, Sparkles, Crown, TrendingUp } from "lucide-react";
import { api, PlacesStats, CoursePricing } from "@/lib/api";
import { DeadlineCountdown } from "./DeadlineCountdown";

interface PricingSectionProps {
  onOpenPayment?: (tier?: "basic" | "accelerator" | "vip") => void;
}

export function PricingSection({ onOpenPayment }: PricingSectionProps) {
  const [places, setPlaces] = useState<PlacesStats>({
    total_capacity: 25,
    active_students: 16,
    spots_left: 9,
  });

  // Default pricing fallbacks: 3 990 ₽ (Basic), 6 900 ₽ (Accelerator) and 14 900 ₽ (VIP)
  const [pricing, setPricing] = useState<CoursePricing>({
    basic_price: 3990,
    basic_old_price: 5900,
    accelerator_price: 6900,
    accelerator_old_price: 9900,
    vip_price: 14900,
    vip_old_price: 19900,
  });

  useEffect(() => {
    api.getPlacesStats().then(setPlaces).catch(() => {});
    api.getCoursePricing().then(setPricing).catch(() => {});
  }, []);

  const formatRub = (num: number) => {
    return (num || 0).toString().replace(/\B(?=(\d{3})+(?!\d))/g, " ");
  };

  const basicPrice = pricing.basic_price || 3990;
  const basicOldPrice = (pricing.basic_old_price && pricing.basic_old_price > basicPrice)
    ? pricing.basic_old_price
    : Math.round((basicPrice * 1.4) / 100) * 100;

  // Dynamic strikethrough prices: always guaranteed to be strictly higher than current price
  const accOldPrice = pricing.accelerator_old_price > pricing.accelerator_price
    ? pricing.accelerator_old_price
    : Math.round((pricing.accelerator_price * 1.4) / 100) * 100;

  const vipOldPrice = pricing.vip_old_price > pricing.vip_price
    ? pricing.vip_old_price
    : Math.round((pricing.vip_price * 1.4) / 100) * 100;

  return (
    <section id="pricing" className="relative w-full max-w-[1520px] mx-auto px-3 sm:px-6 py-16 sm:py-24 scroll-mt-20 sm:scroll-mt-24">
      {/* Section Header */}
      <div className="text-center max-w-3xl mx-auto mb-10 sm:mb-12">
        <div className="inline-flex items-center gap-2 px-3.5 py-1.5 rounded-full bento-pill text-xs font-mono text-emerald-800 dark:text-emerald-300 mb-4">
          <span className="w-2 h-2 rounded-full bg-emerald-500 animate-pulse" />
          <span>Тарифная сетка</span>
        </div>
        <h2 className="font-editorial text-3xl sm:text-5xl lg:text-6xl font-bold tracking-tight text-slate-900 dark:text-white leading-[1.1]">
          Инвестируй в шанс <br />
          <span className="text-emerald-600 dark:text-emerald-400 font-serif italic">выиграть грант $20,000</span>
        </h2>
        <p className="mt-4 text-base text-slate-600 dark:text-slate-300 leading-relaxed font-normal">
          Выбери комфортный формат подготовки: самостоятельный по видео- и аудиоурокам, полное персональное менторство от финалиста SPARK 2026 или VIP-сопровождение под ключ.
        </p>

        {/* Live Scarcity Counter Bar */}
        <div className="mt-6 inline-flex flex-wrap items-center justify-center gap-3 px-4 py-2 rounded-full bg-amber-500/10 border border-amber-500/25 text-xs font-mono">
          <span className="w-2 h-2 rounded-full bg-amber-500 animate-ping" />
          <span className="font-semibold text-slate-900 dark:text-white">
            Лимит потока: {places.total_capacity} мест
          </span>
          <span className="text-slate-400 dark:text-white/30">•</span>
          <span className="text-amber-900 dark:text-amber-300 font-bold">
            Осталось {places.spots_left} мест
          </span>
        </div>
      </div>

      {/* Deadline Countdown Widget (20 ноября 23:59) */}
      <div className="max-w-5xl mx-auto mb-10 sm:mb-12">
        <DeadlineCountdown onOpenPayment={() => onOpenPayment?.("accelerator")} />
      </div>

      {/* Value Contrast & ROI Breakdown Card (6 900 ₽ vs Цена ошибки) */}
      <motion.div
        initial={{ opacity: 0, y: 16 }}
        whileInView={{ opacity: 1, y: 0 }}
        viewport={{ once: true }}
        transition={{ duration: 0.45 }}
        className="max-w-5xl mx-auto mb-12 p-6 sm:p-8 rounded-3xl bento-card-dark border border-emerald-500/25 dark:border-white/10 shadow-[0_15px_45px_rgba(16,185,129,0.06)] relative overflow-hidden bg-white/80 dark:bg-[#12161f]/80"
      >
        <div className="flex items-center justify-between mb-6 pb-4 border-b border-emerald-500/15 dark:border-white/10">
          <div className="flex items-center gap-2.5">
            <div className="w-8 h-8 rounded-xl bg-emerald-500/15 text-emerald-600 dark:text-emerald-400 flex items-center justify-center font-bold">
              <TrendingUp className="w-4 h-4" />
            </div>
            <div>
              <h3 className="font-editorial text-lg sm:text-xl font-bold text-slate-900 dark:text-white">
                Математика инвестиции: почему курс окупается сразу
              </h3>
              <p className="text-xs text-slate-500 dark:text-slate-400 font-mono mt-0.5">
                Сравните разовую стоимость системной подготовки со стоимостью одной ошибки в заявке
              </p>
            </div>
          </div>
          <span className="hidden sm:inline-flex px-3 py-1 rounded-full text-xs font-mono font-bold uppercase bg-emerald-500/15 text-emerald-700 dark:text-emerald-300 border border-emerald-500/30">
            ROI 1 : 290
          </span>
        </div>

        <div className="grid grid-cols-1 lg:grid-cols-11 gap-6 items-center">
          {/* Left Side: Course Investment (5 cols) */}
          <div className="lg:col-span-5 p-5 sm:p-6 rounded-2xl bg-emerald-500/5 dark:bg-emerald-500/10 border border-emerald-500/20 flex flex-col justify-between h-full">
            <div>
              <div className="flex items-center gap-2 mb-2">
                <span className="w-2 h-2 rounded-full bg-emerald-500" />
                <span className="text-[11px] font-mono font-bold uppercase tracking-wider text-emerald-700 dark:text-emerald-300">
                  Стоимость подготовки
                </span>
              </div>
              <div className="text-3xl sm:text-4xl font-black font-mono text-emerald-600 dark:text-emerald-400 my-1">
                {formatRub(pricing.accelerator_price)} ₽
              </div>
              <p className="text-xs sm:text-sm text-slate-600 dark:text-slate-300 font-doc leading-relaxed mt-2 mb-4">
                Меньше одного ужина в кафе или 2 часов с репетитором по английскому. Разовая инвестиция за полный цикл подготовки под ключ.
              </p>
            </div>

            <div className="space-y-2 pt-3 border-t border-emerald-500/15 text-xs font-mono text-slate-700 dark:text-slate-300">
              <div className="flex items-center gap-2">
                <Check className="w-3.5 h-3.5 text-emerald-500 shrink-0" />
                <span>Личный аудит эссе, резюме и видеовизитки</span>
              </div>
              <div className="flex items-center gap-2">
                <Check className="w-3.5 h-3.5 text-emerald-500 shrink-0" />
                <span>45-мин персональный Zoom-интервью 1-на-1</span>
              </div>
              <div className="flex items-center gap-2">
                <Check className="w-3.5 h-3.5 text-emerald-500 shrink-0" />
                <span>Пошаговый визовый гайд J-1 и закрытый чат</span>
              </div>
            </div>
          </div>

          {/* VS Divider (1 col) */}
          <div className="lg:col-span-1 flex items-center justify-center">
            <span className="w-10 h-10 rounded-full bg-slate-900 text-white dark:bg-white dark:text-black flex items-center justify-center font-mono text-xs font-black uppercase shadow-lg">
              VS
            </span>
          </div>

          {/* Right Side: The Cost of Mistake / What is at stake (5 cols) */}
          <div className="lg:col-span-5 p-5 sm:p-6 rounded-2xl bg-rose-500/5 dark:bg-rose-500/10 border border-rose-500/20 flex flex-col justify-between h-full">
            <div>
              <div className="flex items-center gap-2 mb-2">
                <span className="w-2 h-2 rounded-full bg-rose-500" />
                <span className="text-[11px] font-mono font-bold uppercase tracking-wider text-rose-700 dark:text-rose-300">
                  Что стоит на кону при отказе
                </span>
              </div>
              <div className="text-3xl sm:text-4xl font-black font-mono text-rose-600 dark:text-rose-400 my-1">
                $20,000 + 1 год
              </div>
              <p className="text-xs sm:text-sm text-slate-600 dark:text-slate-300 font-doc leading-relaxed mt-2 mb-4">
                Отбор в SPARK проходит строго 1 раз в год. Ошибка в первом туре лишает вас гранта и откладывает поездку в США на год (когда вы можете уже не пройти возрастной ценз 18–21).
              </p>
            </div>

            <div className="space-y-2 pt-3 border-t border-rose-500/15 text-xs font-mono text-slate-700 dark:text-slate-300">
              <div className="flex items-start gap-2">
                <span className="text-rose-500 font-bold shrink-0">✕</span>
                <span>$20,000 расходов (перелёт, проживание, кампус в США)</span>
              </div>
              <div className="flex items-start gap-2">
                <span className="text-rose-500 font-bold shrink-0">✕</span>
                <span>Нетворкинг с Николаем Давыдовым (Davidovs vc) и топ-фаундерами</span>
              </div>
              <div className="flex items-start gap-2">
                <span className="text-rose-500 font-bold shrink-0">✕</span>
                <span>Соотношение ценности курса к гранту: <b>1 к 290</b></span>
              </div>
            </div>
          </div>
        </div>
      </motion.div>

      {/* 3-Tier Pricing Grid: Базовый, Акселератор & VIP */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6 lg:gap-8 items-stretch max-w-7xl mx-auto">
        
        {/* ========================================================================= */}
        {/* TIER 1: БАЗОВЫЙ (Self-Paced) - БЕЗ ОБРАТНОЙ СВЯЗИ                          */}
        {/* ========================================================================= */}
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          whileInView={{ opacity: 1, y: 0 }}
          viewport={{ once: true }}
          transition={{ duration: 0.5 }}
          className="bento-card-dark p-6 sm:p-8 rounded-3xl border border-slate-200 dark:border-white/10 flex flex-col justify-between relative overflow-hidden bg-white/70 dark:bg-[#12161f]/70"
        >
          <div>
            <div className="flex items-center justify-between mb-4">
              <span className="px-3 py-1 rounded-full text-xs font-mono font-medium uppercase bg-slate-100 dark:bg-white/10 text-slate-700 dark:text-slate-300 border border-slate-200 dark:border-white/10">
                Self-Paced
              </span>
              <span className="text-[11px] font-mono text-rose-600 dark:text-rose-400 font-semibold">
                Без обратной связи
              </span>
            </div>

            <h3 className="font-editorial text-2xl sm:text-3xl font-bold text-slate-900 dark:text-white mb-2">
              Базовый
            </h3>
            <p className="text-xs sm:text-sm text-slate-600 dark:text-slate-300 font-doc mb-6 leading-relaxed">
              Самостоятельное прохождение курса по материалам и видеоразборам заявок в удобном темпе.
            </p>

            <div className="mb-6 pb-6 border-b border-emerald-500/15 dark:border-white/10">
              <div className="text-slate-400 line-through text-sm font-mono">
                {formatRub(basicOldPrice)} ₽
              </div>
              <div className="text-3xl sm:text-4xl font-black font-mono text-slate-900 dark:text-white tracking-tight">
                {formatRub(basicPrice)} ₽
              </div>
            </div>

            {/* Features */}
            <div className="space-y-3 font-doc text-xs sm:text-sm text-slate-700 dark:text-slate-200 mb-8">
              <div className="flex items-start gap-2.5">
                <Check className="w-4 h-4 text-emerald-500 shrink-0 mt-0.5" />
                <span><b>Видеоуроки</b> по всем этапам и критериям отбора</span>
              </div>
              <div className="flex items-start gap-2.5">
                <Check className="w-4 h-4 text-emerald-500 shrink-0 mt-0.5" />
                <span><b>Уроки текстом</b> с подробными конспектами и примерами</span>
              </div>
              <div className="flex items-start gap-2.5">
                <Check className="w-4 h-4 text-emerald-500 shrink-0 mt-0.5" />
                <span><b>Аудиоверсия всех уроков</b> — удобно слушать в дороге</span>
              </div>
              <div className="flex items-start gap-2.5">
                <Check className="w-4 h-4 text-emerald-500 shrink-0 mt-0.5" />
                <span>Доступ в закрытый Telegram-канал со всеми материалами</span>
              </div>
              <div className="flex items-start gap-2.5 text-slate-400 dark:text-slate-500">
                <span className="w-4 h-4 text-center shrink-0 font-bold">—</span>
                <span className="line-through"><b>БЕЗ обратной связи</b> и проверки заданий</span>
              </div>
              <div className="flex items-start gap-2.5 text-slate-400 dark:text-slate-500">
                <span className="w-4 h-4 text-center shrink-0 font-bold">—</span>
                <span className="line-through">Без личного Zoom мок-интервью</span>
              </div>
            </div>
          </div>

          <button
            type="button"
            onClick={() => onOpenPayment?.("basic")}
            className="w-full py-3.5 px-4 rounded-2xl bento-pill hover:bg-emerald-500/15 border border-emerald-500/25 text-xs sm:text-sm font-mono font-bold text-slate-900 dark:text-white flex items-center justify-center gap-2 transition-all active:scale-95"
          >
            <span>Выбрать Базовый ({formatRub(basicPrice)} ₽)</span>
            <ArrowUpRight className="w-4 h-4" />
          </button>
        </motion.div>

        {/* ========================================================================= */}
        {/* TIER 2: АКСЕЛЕРАТОР (Full Mentorship) - ХИТ ПРОДАЖ                         */}
        {/* ========================================================================= */}
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          whileInView={{ opacity: 1, y: 0 }}
          viewport={{ once: true }}
          transition={{ duration: 0.5, delay: 0.1 }}
          className="bento-card-dark p-6 sm:p-8 rounded-3xl border-2 border-emerald-500 shadow-[0_20px_60px_rgba(16,185,129,0.18)] dark:shadow-[0_25px_80px_rgba(16,185,129,0.25)] flex flex-col justify-between relative overflow-hidden bg-white dark:bg-[#12161f]"
        >
          {/* Accent Ribbon */}
          <div className="absolute top-0 right-0 left-0 h-1.5 bg-gradient-to-r from-emerald-500 via-teal-400 to-emerald-600" />
          
          <div>
            <div className="flex items-center justify-between mb-4">
              <span className="px-3 py-1 rounded-full text-xs font-mono font-bold uppercase bg-emerald-500/15 text-emerald-700 dark:text-emerald-300 border border-emerald-500/30 flex items-center gap-1.5">
                <Sparkles className="w-3.5 h-3.5 fill-current" />
                Хит продаж
              </span>
              <span className="text-[11px] font-mono text-emerald-600 dark:text-emerald-400 font-semibold">
                Выбор 85% студентов
              </span>
            </div>

            <h3 className="font-editorial text-2xl sm:text-3xl font-bold text-slate-900 dark:text-white mb-2">
              Акселератор
            </h3>
            <p className="text-xs sm:text-sm text-slate-600 dark:text-slate-300 font-doc mb-6 leading-relaxed">
              Полный цикл подготовки под ключ: аудит эссе и резюме, личная репетиция интервью 1-на-1 и визовый гайд.
            </p>

            <div className="mb-6 pb-6 border-b border-emerald-500/20 dark:border-white/10">
              <div className="text-slate-500 dark:text-slate-400 line-through text-sm font-mono">
                {formatRub(accOldPrice)} ₽
              </div>
              <div className="text-3xl sm:text-4xl font-black font-mono text-emerald-600 dark:text-emerald-400 tracking-tight">
                {formatRub(pricing.accelerator_price)} ₽
              </div>
            </div>

            {/* Features */}
            <div className="space-y-3 font-doc text-xs sm:text-sm text-slate-700 dark:text-slate-200 mb-8">
              <div className="flex items-start gap-2.5 font-semibold text-slate-900 dark:text-white">
                <Check className="w-4 h-4 text-emerald-500 shrink-0 mt-0.5" />
                <span>Бессрочный доступ в закрытый канал потока со всеми материалами</span>
              </div>
              <div className="flex items-start gap-2.5 font-semibold text-slate-900 dark:text-white">
                <Check className="w-4 h-4 text-emerald-500 shrink-0 mt-0.5" />
                <span>Подробный курс по каждому элементу заявки</span>
              </div>
              <div className="flex items-start gap-2.5">
                <Check className="w-4 h-4 text-emerald-500 shrink-0 mt-0.5" />
                <span>Личный аудит материалов: разбор эссе, резюме и видеовизитки</span>
              </div>
              <div className="flex items-start gap-2.5 font-medium text-emerald-800 dark:text-emerald-300">
                <Check className="w-4 h-4 text-emerald-500 shrink-0 mt-0.5" />
                <span>45-мин персональное Zoom-интервью 1-на-1 с финалистом SPARK 2026</span>
              </div>
              <div className="flex items-start gap-2.5">
                <Check className="w-4 h-4 text-emerald-500 shrink-0 mt-0.5" />
                <span>Пошаговый визовый гайд J-1 (DS-160, привязка к родине, собеседование)</span>
              </div>
              <div className="flex items-start gap-2.5">
                <Check className="w-4 h-4 text-emerald-500 shrink-0 mt-0.5" />
                <span>Закрытый Telegram-чат участников потока и ответы на вопросы</span>
              </div>
            </div>
          </div>

          <button
            type="button"
            onClick={() => onOpenPayment?.("accelerator")}
            className="btn-primary-mono w-full py-4 text-sm font-bold flex items-center justify-center gap-2 group shadow-xl active:scale-95"
          >
            <span>Занять место за {formatRub(pricing.accelerator_price)} ₽</span>
            <ArrowUpRight className="w-4 h-4 stroke-[2.5]" />
          </button>
        </motion.div>

        {/* ========================================================================= */}
        {/* TIER 3: VIP                                                               */}
        {/* ========================================================================= */}
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          whileInView={{ opacity: 1, y: 0 }}
          viewport={{ once: true }}
          transition={{ duration: 0.5, delay: 0.2 }}
          className="bento-card-dark p-6 sm:p-8 rounded-3xl border border-amber-500/30 dark:border-amber-400/20 flex flex-col justify-between relative overflow-hidden bg-white/70 dark:bg-[#12161f]/70"
        >
          <div>
            <div className="flex items-center justify-between mb-4">
              <span className="px-3 py-1 rounded-full text-xs font-mono font-bold uppercase bg-amber-500/15 text-amber-700 dark:text-amber-300 border border-amber-500/30 flex items-center gap-1.5">
                <Crown className="w-3.5 h-3.5" />
                Строго 3 места
              </span>
            </div>

            <h3 className="font-editorial text-2xl sm:text-3xl font-bold text-slate-900 dark:text-white mb-2">
              VIP
            </h3>
            <p className="text-xs sm:text-sm text-slate-600 dark:text-slate-300 font-doc mb-6 leading-relaxed">
              Максимальный персональный фокус: 3 мок-интервью и приоритетный личный чат до самого вылета в США.
            </p>

            <div className="mb-6 pb-6 border-b border-emerald-500/15 dark:border-white/10">
              <div className="text-slate-500 dark:text-slate-400 line-through text-sm font-mono">
                {formatRub(vipOldPrice)} ₽
              </div>
              <div className="text-3xl sm:text-4xl font-black font-mono text-amber-600 dark:text-amber-400 tracking-tight">
                {formatRub(pricing.vip_price)} ₽
              </div>
            </div>

            {/* Features */}
            <div className="space-y-3 font-doc text-xs sm:text-sm text-slate-700 dark:text-slate-200 mb-8">
              <div className="flex items-start gap-2.5 font-semibold text-slate-900 dark:text-white">
                <Check className="w-4 h-4 text-amber-500 shrink-0 mt-0.5" />
                <span>Всё, что входит в тариф Акселератор</span>
              </div>
              <div className="flex items-start gap-2.5 font-medium text-amber-800 dark:text-amber-300">
                <Check className="w-4 h-4 text-amber-500 shrink-0 mt-0.5" />
                <span>3 индивидуальных Zoom-симуляции интервью с детальным разбором</span>
              </div>
              <div className="flex items-start gap-2.5 font-medium text-amber-800 dark:text-amber-300">
                <Check className="w-4 h-4 text-amber-500 shrink-0 mt-0.5" />
                <span>Приоритетный личный чат с финалистом SPARK 2026 до самого вылета в США</span>
              </div>
              <div className="flex items-start gap-2.5">
                <Check className="w-4 h-4 text-amber-500 shrink-0 mt-0.5" />
                <span>Персональный контроль всех дедлайнов и вычитка всех полей заявки</span>
              </div>
              <div className="flex items-start gap-2.5">
                <Check className="w-4 h-4 text-amber-500 shrink-0 mt-0.5" />
                <span>Помощь с подбором американских курсов и адаптацией на кампусе</span>
              </div>
            </div>
          </div>

          <button
            type="button"
            onClick={() => onOpenPayment?.("vip")}
            className="w-full py-3.5 px-4 rounded-2xl bento-pill hover:bg-amber-500/15 border border-amber-500/30 text-xs sm:text-sm font-mono font-bold text-amber-700 dark:text-amber-300 flex items-center justify-center gap-2 transition-all active:scale-95"
          >
            <span>Выбрать VIP ({formatRub(pricing.vip_price)} ₽)</span>
            <ArrowUpRight className="w-4 h-4" />
          </button>
        </motion.div>

      </div>

      {/* Security & Guarantees Footer Bar */}
      <div className="mt-12 max-w-4xl mx-auto flex flex-col sm:flex-row items-center justify-between gap-4 p-5 rounded-2xl bento-card-dark border border-emerald-500/15 dark:border-white/10 text-xs font-mono text-slate-600 dark:text-slate-400">
        <div className="flex items-center gap-2">
          <ShieldCheck className="w-4 h-4 text-emerald-500 shrink-0" />
          <span>Официальный фискальный чек ФНС РФ (Самозанятый Васюнин Илья Олегович)</span>
        </div>
        <div className="flex items-center gap-2">
          <CreditCard className="w-4 h-4 text-sky-500 shrink-0" />
          <span>СБП 0% • Любые карты РФ • Безопасный платёж</span>
        </div>
      </div>
    </section>
  );
}
