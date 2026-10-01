"use client";

import React, { useState } from "react";
import Link from "next/link";
import { motion } from "framer-motion";
import { Coins, ArrowUpRight, TrendingUp } from "lucide-react";

export function AffiliateHook() {
  const [friendsCount, setFriendsCount] = useState(3);
  const payoutPerPerson = 1035; // 15% from 6900
  const tier2Payout = 345; // 5% from 6900
  const totalDirectEarnings = friendsCount * payoutPerPerson;
  const estimatedTier2 = Math.floor(friendsCount * 1.5) * tier2Payout;

  return (
    <section id="affiliate" className="relative w-full max-w-[1520px] mx-auto px-3 sm:px-6 py-16 sm:py-24">
      <div className="bento-card-dark p-8 sm:p-12 lg:p-14 rounded-2xl sm:rounded-3xl border border-emerald-500/20 dark:border-white/20 relative overflow-hidden shadow-[0_12px_35px_rgba(16,185,129,0.06)] dark:shadow-none">
        
        <div className="grid grid-cols-1 lg:grid-cols-12 gap-10 items-center">
          
          {/* Left Column: Offer & Rates */}
          <div className="lg:col-span-6">
            <div className="inline-flex items-center gap-2 px-3.5 py-1.5 rounded-full bento-pill text-xs font-mono text-emerald-800 dark:text-emerald-300 mb-4">
              <Coins className="w-3.5 h-3.5 text-amber-500 dark:text-amber-400" />
              <span>Двухуровневая партнёрская программа</span>
            </div>

            <h2 className="font-editorial text-3xl sm:text-4xl lg:text-5xl font-bold text-slate-900 dark:text-white leading-tight mb-4">
              Оплати своё участие <br />
              <span className="text-emerald-600 dark:text-emerald-400 font-serif italic">или заработай на поездку в США</span>
            </h2>

            <p className="text-sm sm:text-base text-slate-600 dark:text-slate-300 leading-relaxed mb-6 font-normal">
              Рекомендуй курс одногруппникам и друзьям. Мы выплачиваем щедрые комиссионные за каждую успешную оплату по твоей персональной реферальной ссылке.
            </p>

            {/* Commission Rate Capsules */}
            <div className="grid grid-cols-2 gap-3 mb-8">
              <div className="p-4 rounded-xl bg-emerald-500/[0.04] border border-emerald-500/15 dark:bg-white/5 dark:border-white/10 flex flex-col justify-between">
                <span className="text-xs font-mono text-slate-500 dark:text-slate-400">1-й уровень (Прямой)</span>
                <div className="text-2xl font-bold font-mono text-slate-900 dark:text-white mt-1">
                  15% <span className="text-sm text-emerald-600 dark:text-emerald-400 font-normal">(1 035 ₽)</span>
                </div>
              </div>

              <div className="p-4 rounded-xl bg-emerald-500/[0.04] border border-emerald-500/15 dark:bg-white/5 dark:border-white/10 flex flex-col justify-between">
                <span className="text-xs font-mono text-slate-500 dark:text-slate-400">2-й уровень (Друзья друзей)</span>
                <div className="text-2xl font-bold font-mono text-slate-900 dark:text-white mt-1">
                  5% <span className="text-sm text-sky-600 dark:text-sky-400 font-normal">(345 ₽)</span>
                </div>
              </div>
            </div>

            <Link
              href="/partner"
              className="btn-primary-mono inline-flex items-center gap-2 px-7 py-3.5 text-sm font-bold shadow-xl"
            >
              <span>Стать партнёром и получить ссылку</span>
              <ArrowUpRight className="w-4 h-4 stroke-[2.5]" />
            </Link>
          </div>

          {/* Right Column: Interactive Earnings Slider Calculator */}
          <div className="lg:col-span-6">
            <div className="bento-card-glass p-6 sm:p-8 rounded-2xl border border-emerald-500/20 dark:border-white/20">
              <div className="flex items-center justify-between mb-6">
                <span className="text-xs font-mono text-emerald-800 dark:text-slate-200 uppercase tracking-wider font-bold flex items-center gap-2">
                  <TrendingUp className="w-4 h-4 text-emerald-600 dark:text-emerald-400" />
                  Калькулятор заработка
                </span>
                <span className="bento-pill px-2.5 py-0.5 rounded-full text-xs font-mono text-emerald-800 dark:text-slate-200">
                  Live Calculator
                </span>
              </div>

              {/* Slider Controller */}
              <div className="mb-6">
                <div className="flex items-center justify-between text-xs font-mono text-slate-600 dark:text-slate-300 mb-2">
                  <span>Сколько друзей придёт по твоей ссылке:</span>
                  <span className="text-lg font-bold font-mono text-slate-900 dark:text-white">
                    {friendsCount} {friendsCount === 1 ? "друг" : friendsCount < 5 ? "друга" : "друзей"}
                  </span>
                </div>

                <input
                  type="range"
                  min="1"
                  max="15"
                  value={friendsCount}
                  onChange={(e) => setFriendsCount(parseInt(e.target.value))}
                  className="w-full h-2 bg-emerald-500/20 dark:bg-white/20 rounded-lg appearance-none cursor-pointer accent-emerald-600 dark:accent-white"
                />

                <div className="flex justify-between text-[10px] font-mono text-slate-500 dark:text-slate-400 mt-1">
                  <span>1</span>
                  <span>5</span>
                  <span>10</span>
                  <span>15</span>
                </div>
              </div>

              {/* Live Calculation Output Card */}
              <div className="p-5 rounded-2xl bg-white/90 dark:bg-black/40 border border-emerald-500/20 dark:border-white/10 space-y-3 shadow-sm">
                <div className="flex items-center justify-between text-xs font-mono">
                  <span className="text-slate-600 dark:text-slate-300">Прямой доход (15%):</span>
                  <span className="text-slate-900 dark:text-white font-bold">{totalDirectEarnings.toLocaleString("ru-RU")} ₽</span>
                </div>
                <div className="flex items-center justify-between text-xs font-mono">
                  <span className="text-slate-600 dark:text-slate-300">Ожидаемый 2-й уровень (5%):</span>
                  <span className="text-slate-500 dark:text-slate-400">+{estimatedTier2.toLocaleString("ru-RU")} ₽</span>
                </div>
                <div className="pt-3 border-t border-emerald-500/15 dark:border-white/10 flex items-center justify-between">
                  <span className="text-xs font-mono uppercase tracking-wider text-slate-700 dark:text-slate-200 font-bold">
                    Итоговый потенциал:
                  </span>
                  <span className="text-2xl sm:text-3xl font-black font-mono text-emerald-700 dark:text-emerald-400">
                    {(totalDirectEarnings + estimatedTier2).toLocaleString("ru-RU")} ₽
                  </span>
                </div>
              </div>

              <div className="mt-4 text-[11px] font-mono text-slate-600 dark:text-slate-300 text-center">
                Выплаты осуществляются на любую банковскую карту РФ или СБП по запросу в личном кабинете.
              </div>
            </div>
          </div>

        </div>

      </div>
    </section>
  );
}
