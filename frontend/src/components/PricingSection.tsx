"use client";

import React, { useState, useEffect } from "react";
import { motion } from "framer-motion";
import { Check, ArrowUpRight, ShieldCheck, CreditCard, Sparkles, Crown } from "lucide-react";
import { api, PlacesStats } from "@/lib/api";

interface PricingSectionProps {
  onOpenPayment?: (tier?: "basic" | "accelerator" | "vip") => void;
}

export function PricingSection({ onOpenPayment }: PricingSectionProps) {
  const [places, setPlaces] = useState<PlacesStats>({
    total_capacity: 25,
    active_students: 16,
    spots_left: 9,
  });

  useEffect(() => {
    api.getPlacesStats().then(setPlaces).catch(() => {});
  }, []);

  return (
    <section id="pricing" className="relative w-full max-w-[1520px] mx-auto px-3 sm:px-6 py-16 sm:py-24 scroll-mt-20 sm:scroll-mt-24">
      {/* Section Header */}
      <div className="text-center max-w-3xl mx-auto mb-12 sm:mb-16">
        <div className="inline-flex items-center gap-2 px-3.5 py-1.5 rounded-full bento-pill text-xs font-mono text-emerald-800 dark:text-emerald-300 mb-4">
          <span className="w-2 h-2 rounded-full bg-emerald-500 animate-pulse" />
          <span>Тарифная сетка</span>
        </div>
        <h2 className="font-editorial text-3xl sm:text-5xl lg:text-6xl font-bold tracking-tight text-slate-900 dark:text-white leading-[1.1]">
          Инвестируй в шанс <br />
          <span className="text-emerald-600 dark:text-emerald-400 font-serif italic">выиграть грант $20,000</span>
        </h2>
        <p className="mt-4 text-base text-slate-600 dark:text-slate-300 leading-relaxed font-normal">
          Выбери комфортный формат подготовки: самостоятельный по проверенным шаблонам, полное персональное менторство от так называемого Иля или VIP-сопровождение под ключ.
        </p>

        {/* Live Scarcity Counter Bar */}
        <div className="mt-6 inline-flex flex-wrap items-center justify-center gap-3 px-4 py-2 rounded-full bg-amber-500/10 border border-amber-500/25 text-xs font-mono">
          <span className="w-2 h-2 rounded-full bg-amber-500 animate-ping" />
          <span className="font-semibold text-slate-900 dark:text-white">
            Лимит потока: {places.total_capacity} мест
          </span>
          <span className="text-slate-400 dark:text-white/30">•</span>
          <span className="text-amber-700 dark:text-amber-400 font-bold">
            Осталось {places.spots_left} мест
          </span>
        </div>
      </div>

      {/* 3-Tier Pricing Grid */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6 lg:gap-8 items-stretch max-w-7xl mx-auto">
        
        {/* ========================================================================= */}
        {/* TIER 1: БАЗОВЫЙ (Self-Paced)                                              */}
        {/* ========================================================================= */}
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          whileInView={{ opacity: 1, y: 0 }}
          viewport={{ once: true }}
          transition={{ duration: 0.5 }}
          className="bento-card-dark p-6 sm:p-8 rounded-3xl border border-emerald-500/15 dark:border-white/10 flex flex-col justify-between relative overflow-hidden bg-white/70 dark:bg-[#12161f]/70"
        >
          <div>
            <div className="flex items-center justify-between mb-4">
              <span className="tag-neutral px-3 py-1 rounded-full text-xs font-mono font-medium uppercase">
                Self-Paced
              </span>
            </div>

            <h3 className="font-editorial text-2xl sm:text-3xl font-bold text-slate-900 dark:text-white mb-2">
              Базовый
            </h3>
            <p className="text-xs sm:text-sm text-slate-600 dark:text-slate-300 font-doc mb-6 leading-relaxed">
              Самостоятельная подготовка по проверенным шаблонам и видеоразборам реальных заявок.
            </p>

            <div className="mb-6 pb-6 border-b border-emerald-500/15 dark:border-white/10">
              <div className="text-slate-400 line-through text-sm font-mono">
                4 900 ₽
              </div>
              <div className="text-3xl sm:text-4xl font-black font-mono text-slate-900 dark:text-white tracking-tight">
                2 900 ₽
              </div>
              <div className="text-xs font-mono text-emerald-600 dark:text-emerald-400 mt-1 font-semibold">
                С промокодом -5%: 2 755 ₽
              </div>
            </div>

            {/* Features */}
            <div className="space-y-3 font-doc text-xs sm:text-sm text-slate-700 dark:text-slate-200 mb-8">
              <div className="flex items-start gap-2.5">
                <Check className="w-4 h-4 text-emerald-500 shrink-0 mt-0.5" />
                <span>Доступ в закрытый канал потока со всеми материалами и апдейтами отбора</span>
              </div>
              <div className="flex items-start gap-2.5">
                <Check className="w-4 h-4 text-emerald-500 shrink-0 mt-0.5" />
                <span>Шаблоны Google XYZ резюме и победные структуры эссе на 500 слов</span>
              </div>
              <div className="flex items-start gap-2.5">
                <Check className="w-4 h-4 text-emerald-500 shrink-0 mt-0.5" />
                <span>Видеоразборы удачных и провальных кейсов прошлых лет</span>
              </div>
              <div className="flex items-start gap-2.5">
                <Check className="w-4 h-4 text-emerald-500 shrink-0 mt-0.5" />
                <span>Закрытый Telegram-чат участников потока</span>
              </div>
              <div className="flex items-start gap-2.5 text-slate-400">
                <span className="w-4 h-4 text-center shrink-0">—</span>
                <span className="line-through">Без личного мок-интервью в Zoom</span>
              </div>
            </div>
          </div>

          <button
            type="button"
            onClick={() => onOpenPayment?.("basic")}
            className="w-full py-3.5 px-4 rounded-2xl bento-pill hover:bg-emerald-500/15 border border-emerald-500/25 text-xs sm:text-sm font-mono font-bold text-slate-900 dark:text-white flex items-center justify-center gap-2 transition-all active:scale-95"
          >
            <span>Выбрать Базовый (2 900 ₽)</span>
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
              <div className="text-slate-400 line-through text-sm font-mono">
                9 900 ₽
              </div>
              <div className="text-3xl sm:text-4xl font-black font-mono text-emerald-600 dark:text-emerald-400 tracking-tight">
                6 900 ₽
              </div>
              <div className="text-xs font-mono text-emerald-600 dark:text-emerald-400 mt-1 font-semibold">
                С промокодом -5%: 6 555 ₽
              </div>
            </div>

            {/* Features */}
            <div className="space-y-3 font-doc text-xs sm:text-sm text-slate-700 dark:text-slate-200 mb-8">
              <div className="flex items-start gap-2.5 font-semibold text-slate-900 dark:text-white">
                <Check className="w-4 h-4 text-emerald-500 shrink-0 mt-0.5" />
                <span>Всё, что входит в тариф Базовый</span>
              </div>
              <div className="flex items-start gap-2.5">
                <Check className="w-4 h-4 text-emerald-500 shrink-0 mt-0.5" />
                <span>Личный аудит материалов: разбор эссе, резюме и видеовизитки</span>
              </div>
              <div className="flex items-start gap-2.5 font-medium text-emerald-800 dark:text-emerald-300">
                <Check className="w-4 h-4 text-emerald-500 shrink-0 mt-0.5" />
                <span>45-мин персональное Zoom-интервью 1-на-1 с так называемым Илем</span>
              </div>
              <div className="flex items-start gap-2.5">
                <Check className="w-4 h-4 text-emerald-500 shrink-0 mt-0.5" />
                <span>Пошаговый визовый гайд J-1 (DS-160, привязка к родине, собеседование)</span>
              </div>
              <div className="flex items-start gap-2.5">
                <Check className="w-4 h-4 text-emerald-500 shrink-0 mt-0.5" />
                <span>Личные ответы на любые вопросы по ходу подачи заявки</span>
              </div>
            </div>
          </div>

          <button
            type="button"
            onClick={() => onOpenPayment?.("accelerator")}
            className="btn-primary-mono w-full py-4 text-sm font-bold flex items-center justify-center gap-2 group shadow-xl active:scale-95"
          >
            <span>Занять место за 6 900 ₽</span>
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
              <div className="text-slate-400 line-through text-sm font-mono">
                19 900 ₽
              </div>
              <div className="text-3xl sm:text-4xl font-black font-mono text-amber-600 dark:text-amber-400 tracking-tight">
                14 900 ₽
              </div>
              <div className="text-xs font-mono text-amber-600 dark:text-amber-400 mt-1 font-semibold">
                С промокодом -5%: 14 155 ₽
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
                <span>Приоритетный личный чат с так называемым Илем до самого вылета в США</span>
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
            <span>Выбрать VIP (14 900 ₽)</span>
            <ArrowUpRight className="w-4 h-4" />
          </button>
        </motion.div>

      </div>

      {/* Reverse Risk Guarantee Banner */}
      <div className="mt-10 max-w-4xl mx-auto p-6 sm:p-7 rounded-3xl bento-card-dark border border-amber-500/30 bg-gradient-to-r from-amber-500/[0.07] via-transparent to-emerald-500/[0.07] shadow-lg">
        <div className="flex flex-col sm:flex-row items-start sm:items-center gap-4">
          <div className="w-12 h-12 rounded-2xl bg-amber-500/15 border border-amber-500/30 flex items-center justify-center shrink-0 text-amber-500">
            <ShieldCheck className="w-6 h-6" />
          </div>
          <div className="flex-1">
            <div className="flex items-center gap-2 mb-1">
              <span className="font-editorial text-lg sm:text-xl font-bold text-slate-900 dark:text-white">
                Обратная гарантия риска: 100% возврат или перенос
              </span>
              <span className="text-[10px] font-mono uppercase px-2 py-0.5 rounded-full bg-amber-500/20 text-amber-700 dark:text-amber-300 font-bold border border-amber-500/30">
                Pied Piper Guarantee
              </span>
            </div>
            <p className="text-xs sm:text-sm text-slate-600 dark:text-slate-300 font-doc leading-relaxed">
              Если ты пройдешь все модули акселератора, выполнишь практические задания, и приёмная комиссия не примет твою заявку к рассмотрению — мы возвращаем 100% стоимости или бесплатно переносим участие на следующий сезон. Мы разделяем риск вместе с тобой.
            </p>
          </div>
        </div>
      </div>

      {/* Security & Guarantees Footer Bar */}
      <div className="mt-6 max-w-4xl mx-auto flex flex-col sm:flex-row items-center justify-between gap-4 p-5 rounded-2xl bento-card-dark border border-emerald-500/15 dark:border-white/10 text-xs font-mono text-slate-600 dark:text-slate-400">
        <div className="flex items-center gap-2">
          <ShieldCheck className="w-4 h-4 text-emerald-500 shrink-0" />
          <span>Официальный фискальный чек ФНС РФ (Самозанятый Иль О. В.)</span>
        </div>
        <div className="flex items-center gap-2">
          <CreditCard className="w-4 h-4 text-sky-500 shrink-0" />
          <span>СБП 0% • Любые карты РФ • Безопасный платёж</span>
        </div>
      </div>
    </section>
  );
}
