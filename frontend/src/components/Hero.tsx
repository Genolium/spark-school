"use client";

import React, { useState, useEffect } from "react";
import { BlurImage } from "./BlurImage";
import { motion } from "framer-motion";
import { ArrowUpRight, CheckCircle2, Video, FileText } from "lucide-react";
import { api, PlacesStats } from "@/lib/api";

interface HeroProps {
  onOpenPayment?: () => void;
  onOpenLeadMagnet?: () => void;
  onOpenChances?: () => void;
}

export function Hero({ onOpenPayment, onOpenLeadMagnet, onOpenChances }: HeroProps) {
  const [places, setPlaces] = useState<PlacesStats>({
    total_capacity: 25,
    active_students: 16,
    spots_left: 9,
  });

  useEffect(() => {
    api.getPlacesStats().then(setPlaces).catch(() => {});
  }, []);

  return (
    <section id="hero-canvas" className="relative w-full max-w-[1520px] mx-auto px-3 sm:px-6 pt-24 sm:pt-28 pb-8">
      {/* ========================================================================= */}
      {/* MASTER EXPANSIVE SPATIAL BENTO CANVAS                                      */}
      {/* ========================================================================= */}
      <div className="relative w-full min-h-[82vh] lg:min-h-[86vh] rounded-3xl sm:rounded-[32px] overflow-hidden border border-emerald-500/25 dark:border-white/20 shadow-[0_20px_50px_rgba(16,185,129,0.12)] dark:shadow-[0_30px_100px_rgba(0,0,0,0.5)] flex flex-col justify-between p-6 sm:p-12 lg:p-14 bg-[#121316]">
        
        {/* Responsive Background Photograph */}
        <div className="absolute inset-0 z-0">
          {/* Desktop & Tablet: Campus Architectural Photograph */}
          <div className="hidden sm:block absolute inset-0">
            <BlurImage
              src="/spatial-campus-hero.jpg"
              alt="Кампус University of Wyoming и Скалистые горы — место академической стажировки так называемого Иля"
              fill
              priority
              quality={100}
              unoptimized
              className="object-cover object-center"
              sizes="100vw"
            />
          </div>

          {/* Mobile: Golden Gate Bridge Vertical Photo */}
          <div className="block sm:hidden absolute inset-0">
            <BlurImage
              src="/spatial-campus-hero-mobile.jpg"
              alt="Golden Gate Bridge, California — академическая стажировка и путешествия в США"
              fill
              priority
              quality={100}
              unoptimized
              className="object-cover object-center"
              sizes="100vw"
            />
          </div>

          {/* Natural Vignette for Contrast (Reduced by 50% for brighter, vibrant background) */}
          <div className="absolute inset-0 bg-gradient-to-b from-black/25 via-transparent to-black/45" />
          <div className="absolute inset-0 bg-gradient-to-r from-black/35 via-transparent to-black/40" />
        </div>


        {/* ========================================================================= */}
        {/* TOP STATUS BAR: Micro-Tag Pills with Muted Accent Tints                   */}
        {/* ========================================================================= */}
        <div className="relative z-20 hidden sm:flex flex-wrap items-center justify-between gap-4">
          <motion.div
            initial={{ opacity: 0, y: -10 }}
            animate={{ opacity: 1, y: 0 }}
            className="inline-flex items-center gap-2.5 px-4 py-2 rounded-full bg-black/50 hover:bg-black/60 backdrop-blur-xl border border-white/25 text-xs font-mono text-white shadow-lg transition-all"
          >
            <span className="w-2 h-2 rounded-full bg-emerald-400 animate-pulse" />
            <span className="font-semibold text-white">Набор на обучение открыт</span>
            <span className="text-white/40">•</span>
            <span className="text-slate-200">Старт заявочной кампании: Октябрь 2026</span>
          </motion.div>

          {/* Micro Tags */}
          <div className="flex items-center gap-2.5">
            <span className="px-3.5 py-1.5 rounded-full text-xs font-mono font-medium bg-black/50 hover:bg-black/60 backdrop-blur-xl border border-white/25 text-white shadow-sm transition-all">
              #так_называемый_SPARK
            </span>
            <span className="px-3.5 py-1.5 rounded-full text-xs font-mono font-medium bg-black/50 hover:bg-black/60 backdrop-blur-xl border border-white/25 text-white shadow-sm transition-all">
              #Full_Grant_$20k
            </span>
          </div>
        </div>

        {/* ========================================================================= */}
        {/* MAIN EDITORIAL HERO CONTENT                                               */}
        {/* ========================================================================= */}
        <div className="relative z-20 grid grid-cols-1 lg:grid-cols-12 gap-10 items-end mt-4 sm:mt-12 mb-4">
          
          {/* LEFT: Massive Clean Headline & High-Contrast CTAs */}
          <div className="lg:col-span-8 flex flex-col justify-end">
            <motion.div
              initial={{ opacity: 0, y: 20 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ duration: 0.6 }}
            >
              <h1 className="font-heading text-4xl sm:text-6xl lg:text-[76px] font-extrabold tracking-tight leading-[1.04] text-white">
                ВЫИГРАЙ ГРАНТ $20,000
                <br />
                НА УЧЁБУ В США
                <br />
                ЭТИМ ЛЕТОМ
              </h1>

              {/* Subtitle */}
              <p className="mt-6 text-base sm:text-lg text-white max-w-2xl leading-relaxed font-medium drop-shadow-[0_2px_10px_rgba(0,0,0,0.7)]">
                Практический проект от финалиста программы SPARK 2026. Разборы победных заявок, шаблоны документов, персональная 45-минутная симуляция интервью в Zoom и закрытые инсайды отбора.
              </p>

              {/* High-Contrast Action Buttons */}
              <div className="mt-8 flex flex-col sm:flex-row items-stretch sm:items-center gap-4">
                <a
                  href="/#pricing"
                  className="btn-primary-mono px-8 py-4 text-sm sm:text-base flex items-center justify-center gap-2.5 active:scale-95 group shadow-2xl font-bold"
                >
                  <span>Выбрать тариф от 2 900 ₽</span>
                  <ArrowUpRight className="w-4 h-4 transition-transform group-hover:translate-x-0.5 group-hover:-translate-y-0.5 stroke-[2.5]" />
                </a>

                <button
                  type="button"
                  onClick={onOpenChances}
                  className="px-7 py-4 rounded-full text-xs sm:text-sm flex items-center justify-center gap-2.5 active:scale-95 transition-all font-semibold bg-white/20 hover:bg-white/30 text-white border border-white/35 backdrop-blur-xl shadow-xl group cursor-pointer"
                >
                  <span>Рассчитать шансы</span>
                </button>
              </div>

              {/* Trust Indicators with live database spots count */}
              <div className="mt-6 flex flex-wrap items-center gap-5 text-xs font-mono text-white/90 drop-shadow-[0_1px_4px_rgba(0,0,0,0.8)]">
                <span className="flex items-center gap-1.5 text-emerald-400 font-semibold">
                  <CheckCircle2 className="w-4 h-4 text-emerald-400" />
                  100% практический опыт
                </span>
                <span className="text-white/40">•</span>
                <span className="text-white/90">
                  Лимит: {places.total_capacity} мест • Осталось {places.spots_left} мест
                </span>
              </div>
            </motion.div>
          </div>

          {/* RIGHT: Floating Spatial Bento Micro-Widgets */}
          <div className="lg:col-span-4 flex flex-col gap-4 items-stretch lg:items-end">
            
            {/* Floating Widget 1: Mock-Interview Live Simulation */}
            <motion.div
              initial={{ opacity: 0, x: 20 }}
              animate={{ opacity: 1, x: 0 }}
              transition={{ duration: 0.6, delay: 0.2 }}
              className="w-full max-w-sm p-5 sm:p-6 rounded-2xl bg-white/95 backdrop-blur-3xl border border-emerald-500/25 shadow-[0_20px_50px_rgba(0,0,0,0.25)] transition-all group"
            >
              <div className="flex items-center gap-2.5 mb-3.5">
                <div className="w-8 h-8 rounded-full bg-emerald-500/15 flex items-center justify-center text-emerald-800">
                  <Video className="w-4 h-4" />
                </div>
                <div>
                  <span className="text-xs font-mono font-bold uppercase text-slate-900 tracking-wide block">
                    Live Mock-Interview
                  </span>
                  <span className="text-xs text-slate-600 font-semibold">
                    1-на-1 в Zoom • 45 минут
                  </span>
                </div>
              </div>

              <p className="text-xs sm:text-[13px] text-slate-700 leading-relaxed font-normal mb-4">
                Полная симуляция реального собеседования на английском с разбором стресс-вопросов американской комиссии.
              </p>

              <div className="pt-3.5 border-t border-emerald-500/15 flex items-center justify-end text-xs font-mono">
                <span className="text-slate-900 font-bold text-xs sm:text-[13px]">1-на-1 в Zoom • 45 мин</span>
              </div>
            </motion.div>

            {/* Floating Widget 2: Documents Checklist Card */}
            <motion.div
              initial={{ opacity: 0, x: 20 }}
              animate={{ opacity: 1, x: 0 }}
              transition={{ duration: 0.6, delay: 0.3 }}
              className="w-full max-w-sm p-5 sm:p-6 rounded-2xl bg-white/95 backdrop-blur-3xl border border-emerald-500/25 shadow-[0_20px_50px_rgba(0,0,0,0.25)] transition-all group"
            >
              <div className="flex items-center gap-2.5 mb-3.5">
                <div className="w-8 h-8 rounded-full bg-emerald-500/15 flex items-center justify-center text-emerald-800">
                  <FileText className="w-4 h-4" />
                </div>
                <div>
                  <span className="text-xs font-mono font-bold uppercase text-slate-900 tracking-wide block">
                    Пакет документов
                  </span>
                  <span className="text-xs text-slate-600 font-semibold">
                    1-й тур программы
                  </span>
                </div>
              </div>

              <div className="space-y-2 text-xs sm:text-[13px] mb-4">
                <div className="flex items-center gap-2.5">
                  <span className="text-emerald-500 font-bold text-sm">✓</span>
                  <span className="text-slate-900 font-semibold">Резюме</span>
                </div>
                <div className="flex items-center gap-2.5">
                  <span className="text-emerald-500 font-bold text-sm">✓</span>
                  <span className="text-slate-900 font-semibold">Эссе</span>
                </div>
                <div className="flex items-center gap-2.5">
                  <span className="text-emerald-500 font-bold text-sm">✓</span>
                  <span className="text-slate-900 font-semibold">Видеовизитка</span>
                </div>
              </div>

              <div className="pt-3.5 border-t border-emerald-500/15 hidden sm:flex items-center justify-between text-xs font-mono">
                <span className="text-emerald-700 font-semibold flex items-center gap-1.5 text-xs sm:text-[13px]">
                  <span className="w-2 h-2 rounded-full bg-emerald-500" />
                  Готовность к подаче
                </span>
                <span className="text-slate-900 font-bold text-xs sm:text-[13px]">100% соответствие</span>
              </div>
            </motion.div>

          </div>

        </div>

      </div>
    </section>
  );
}
