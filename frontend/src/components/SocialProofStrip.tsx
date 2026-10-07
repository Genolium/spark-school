"use client";

import React from "react";
import { motion } from "framer-motion";
import { GraduationCap, Home, Plane, Wallet, CheckCircle2 } from "lucide-react";

interface SocialProofStripProps {
  onOpenChances?: () => void;
}

export function SocialProofStrip({ onOpenChances }: SocialProofStripProps) {
  const coverageItems = [
    {
      icon: GraduationCap,
      title: "Обучение в США",
      desc: "100% покрытие стоимости учебного семестра",
    },
    {
      icon: Home,
      title: "Жильё и питание",
      desc: "Комната в кампусе и полноценный Meal Plan",
    },
    {
      icon: Plane,
      title: "Авиаперелёты",
      desc: "Билеты в США и обратно за счёт программы",
    },
    {
      icon: Wallet,
      title: "Ежемесячная стипендия",
      desc: "Выплаты на личные расходы и страховка",
    },
  ];

  return (
    <section
      id="grant"
      className="relative w-full max-w-[1520px] mx-auto px-3 sm:px-6 py-4 sm:py-6 scroll-mt-24 sm:scroll-mt-28"
    >
      <motion.div
        initial={{ opacity: 0, y: 15 }}
        whileInView={{ opacity: 1, y: 0 }}
        viewport={{ once: true }}
        transition={{ duration: 0.45 }}
        className="bento-card-dark p-6 sm:p-8 lg:p-10 rounded-2xl sm:rounded-3xl border border-emerald-500/20 dark:border-white/20 relative overflow-hidden group shadow-[0_16px_40px_rgba(16,185,129,0.08)] dark:shadow-[0_20px_50px_rgba(0,0,0,0.4)]"
      >
        {/* Subtle Ambient Emerald Lighting */}
        <div className="absolute -right-24 -top-24 w-80 h-80 bg-emerald-500/10 dark:bg-emerald-500/5 rounded-full blur-3xl pointer-events-none" />

        <div className="grid grid-cols-1 lg:grid-cols-12 gap-8 items-center relative z-10">
          {/* Left: Giant $20,000 Metric & Title */}
          <div className="lg:col-span-5 flex flex-col justify-center">
            <div className="flex items-center gap-2.5 mb-3">
              <span className="bento-pill px-3 py-1 rounded-full text-xs font-mono font-semibold flex items-center gap-1.5 text-emerald-800 dark:text-emerald-300">
                <CheckCircle2 className="w-3.5 h-3.5 text-emerald-500 dark:text-emerald-400" />
                100% покрытие расходов
              </span>
              <span className="text-xs font-mono text-slate-600 dark:text-slate-300">#Full_Grant</span>
            </div>

            <div className="text-6xl sm:text-7xl lg:text-[88px] font-black font-mono tracking-tighter leading-[0.95] text-emerald-600 dark:text-emerald-400 drop-shadow-[0_8px_30px_rgba(16,185,129,0.25)] dark:drop-shadow-[0_12px_45px_rgba(52,211,153,0.4)] flex items-baseline gap-2 my-1">
              <span>$20,000</span>
            </div>

            <h2 className="text-xl sm:text-2xl font-bold text-slate-900 dark:text-white mt-2 tracking-tight">
              Грант Госдепа США
            </h2>

            <p className="text-sm text-slate-600 dark:text-slate-300 mt-2.5 leading-relaxed font-normal">
              Обучение, жильё, перелёт и стипендия. Финалист программы практически ничего не платит — основные расходы полностью покрываются грантом (личные путешествия — по большей части за свои деньги).
            </p>
          </div>

          {/* Right: 4 Clean Coverage Bento Pills + Evaluation Button */}
          <div className="lg:col-span-7 flex flex-col gap-4">
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-3.5">
              {coverageItems.map((item, idx) => {
                const Icon = item.icon;
                return (
                  <div
                    key={idx}
                    className="p-4 rounded-2xl bg-emerald-500/[0.04] border border-emerald-500/15 hover:border-emerald-500/35 dark:bg-white/[0.04] dark:border-white/10 dark:hover:border-white/25 transition-all flex items-start gap-3.5 group/item"
                  >
                    <div className="w-9 h-9 rounded-xl bg-emerald-500/10 dark:bg-white/10 flex items-center justify-center text-emerald-700 dark:text-white shrink-0 mt-0.5 group-hover/item:scale-105 transition-transform">
                      <Icon className="w-4 h-4 text-emerald-600 dark:text-emerald-400" />
                    </div>
                    <div>
                      <div className="text-sm font-bold text-slate-900 dark:text-white font-mono">
                        {item.title}
                      </div>
                      <div className="text-xs text-slate-600 dark:text-slate-400 mt-0.5 leading-snug">
                        {item.desc}
                      </div>
                    </div>
                  </div>
                );
              })}
            </div>

            {/* CTA Button to open chances estimation modal */}
            {onOpenChances && (
              <div className="pt-2">
                <button
                  type="button"
                  onClick={onOpenChances}
                  className="w-full sm:w-auto px-6 py-3.5 rounded-2xl bento-pill hover:bg-emerald-500/15 border border-emerald-500/30 text-xs sm:text-sm font-mono font-bold text-slate-900 dark:text-white flex items-center justify-center gap-2.5 transition-all shadow-sm hover:shadow active:scale-95 group"
                >
                  <span className="w-2 h-2 rounded-full bg-emerald-500 animate-pulse" />
                  <span>Оценить свои шансы на грант</span>
                  <span className="text-emerald-600 dark:text-emerald-400 group-hover:translate-x-0.5 transition-transform">➔</span>
                </button>
              </div>
            )}
          </div>
        </div>
      </motion.div>
    </section>
  );
}
