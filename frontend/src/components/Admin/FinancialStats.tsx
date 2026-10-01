"use client";

import React from "react";
import { TrendingUp, Users, Wallet, CreditCard, ArrowUpRight } from "lucide-react";
import { AdminStats } from "@/lib/api";

interface FinancialStatsProps {
  stats: AdminStats;
}

export function FinancialStats({ stats }: FinancialStatsProps) {
  return (
    <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4 mb-8">
      {/* Gross Revenue */}
      <div className="bento-card-dark p-6 rounded-3xl border border-white/15">
        <div className="flex items-center justify-between mb-4">
          <span className="text-xs font-mono uppercase tracking-wider text-slate-400">
            Общая выручка
          </span>
          <div className="w-8 h-8 rounded-xl bg-emerald-500/20 text-emerald-400 flex items-center justify-center">
            <TrendingUp className="w-4 h-4" />
          </div>
        </div>
        <div className="text-2xl sm:text-3xl font-black font-mono text-white">
          {stats.gross_volume.toLocaleString("ru-RU")} ₽
        </div>
        <div className="text-xs font-mono text-emerald-400 mt-1 flex items-center gap-1 font-semibold">
          <span>● Оплат тарифа: {stats.active_students}</span>
        </div>
      </div>

      {/* Active Students */}
      <div className="bento-card-dark p-6 rounded-3xl border border-white/15">
        <div className="flex items-center justify-between mb-4">
          <span className="text-xs font-mono uppercase tracking-wider text-slate-400">
            Студентов на потоке
          </span>
          <div className="w-8 h-8 rounded-xl bg-sky-500/20 text-sky-400 flex items-center justify-center">
            <Users className="w-4 h-4" />
          </div>
        </div>
        <div className="text-2xl sm:text-3xl font-black font-mono text-white">
          {stats.active_students} <span className="text-sm font-normal text-slate-400">/ 25 мест</span>
        </div>
        <div className="text-xs font-mono text-slate-400 mt-1">
          Всего зарегистрировано: {stats.total_registered}
        </div>
      </div>

      {/* Conversion Rate */}
      <div className="bento-card-dark p-6 rounded-3xl border border-white/15">
        <div className="flex items-center justify-between mb-4">
          <span className="text-xs font-mono uppercase tracking-wider text-slate-400">
            Конверсия в оплату
          </span>
          <div className="w-8 h-8 rounded-xl bg-amber-500/20 text-amber-400 flex items-center justify-center">
            <ArrowUpRight className="w-4 h-4" />
          </div>
        </div>
        <div className="text-2xl sm:text-3xl font-black font-mono text-white">
          {stats.conversion_rate.toFixed(1)}%
        </div>
        <div className="text-xs font-mono text-amber-300 mt-1">
          Выше среднего по EdTech (3.5%)
        </div>
      </div>

      {/* Pending Payouts */}
      <div className="bento-card-dark p-6 rounded-3xl border border-white/15">
        <div className="flex items-center justify-between mb-4">
          <span className="text-xs font-mono uppercase tracking-wider text-slate-400">
            Заявки на вывод (партнёры)
          </span>
          <div className="w-8 h-8 rounded-xl bg-rose-500/20 text-rose-400 flex items-center justify-center">
            <Wallet className="w-4 h-4" />
          </div>
        </div>
        <div className="text-2xl sm:text-3xl font-black font-mono text-white">
          {stats.pending_payouts_sum.toLocaleString("ru-RU")} ₽
        </div>
        <div className="text-xs font-mono text-rose-300 mt-1">
          {stats.pending_payouts_count} заявок требуют подтверждения
        </div>
      </div>
    </div>
  );
}
