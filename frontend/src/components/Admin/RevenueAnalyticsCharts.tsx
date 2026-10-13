"use client";

import React, { useState, useMemo } from "react";
import {
  TrendingUp,
  BarChart3,
  Calendar,
  Zap,
  Target,
  Tag,
  DollarSign,
  ArrowUpRight,
  ShieldCheck,
  ChevronRight,
  Info,
} from "lucide-react";
import { AdminStats, DailyRevenueStat } from "@/lib/api";

interface RevenueAnalyticsChartsProps {
  stats: AdminStats;
}

type PeriodType = "7d" | "14d" | "30d" | "all";
type ChartViewType = "both" | "cumulative" | "daily";

export function RevenueAnalyticsCharts({ stats }: RevenueAnalyticsChartsProps) {
  const [period, setPeriod] = useState<PeriodType>("30d");
  const [viewMode, setChartView] = useState<ChartViewType>("both");
  const [hoveredDailyIndex, setHoveredDailyIndex] = useState<number | null>(null);
  const [hoveredCumulIndex, setHoveredCumulIndex] = useState<number | null>(null);

  // Filter daily stats by period
  const filteredDailyStats = useMemo(() => {
    const raw = stats.daily_stats || [];
    if (raw.length === 0) return [];
    if (period === "7d") return raw.slice(-7);
    if (period === "14d") return raw.slice(-14);
    if (period === "30d") return raw.slice(-30);
    return raw;
  }, [stats.daily_stats, period]);

  // Aggregate metrics for selected period
  const periodMetrics = useMemo(() => {
    let rev = 0;
    let purchases = 0;
    let accCount = 0;
    let vipCount = 0;

    for (const d of filteredDailyStats) {
      rev += d.revenue;
      purchases += d.purchases;
      accCount += d.accelerator_count || 0;
      vipCount += d.vip_count || 0;
    }

    const aov = purchases > 0 ? Math.round(rev / purchases) : 0;

    return {
      revenue: rev,
      purchases,
      accCount,
      vipCount,
      aov,
    };
  }, [filteredDailyStats]);

  // Daily Chart Calculations
  const maxDailyRevenue = useMemo(() => {
    if (filteredDailyStats.length === 0) return 30000;
    const max = Math.max(...filteredDailyStats.map((d) => d.revenue), 10000);
    return Math.ceil(max / 5000) * 5000;
  }, [filteredDailyStats]);

  // Cumulative Chart Calculations (SVG coordinates)
  const cumulChartData = useMemo(() => {
    if (filteredDailyStats.length === 0) {
      return { path: "", areaPath: "", points: [], maxY: 300000 };
    }

    const width = 800;
    const height = 220;
    const paddingX = 40;
    const paddingTop = 25;
    const paddingBottom = 35;
    const plotWidth = width - paddingX * 2;
    const plotHeight = height - paddingTop - paddingBottom;

    const values = filteredDailyStats.map((d) => d.cumulative_revenue);
    const maxVal = Math.max(...values, stats.gross_volume || 289800, 10000);
    const maxY = Math.ceil(maxVal / 25000) * 25000;

    const len = filteredDailyStats.length;
    const points = filteredDailyStats.map((d, i) => {
      const x = len > 1 ? paddingX + (i / (len - 1)) * plotWidth : paddingX + plotWidth / 2;
      const y = height - paddingBottom - (d.cumulative_revenue / maxY) * plotHeight;
      return { x, y, data: d };
    });

    if (points.length < 2) {
      return { path: "", areaPath: "", points, maxY };
    }

    // Build smooth Bezier curve
    let path = `M ${points[0].x} ${points[0].y}`;
    for (let i = 0; i < points.length - 1; i++) {
      const p0 = i > 0 ? points[i - 1] : points[i];
      const p1 = points[i];
      const p2 = points[i + 1];
      const p3 = i < points.length - 2 ? points[i + 2] : p2;

      const cp1x = p1.x + (p2.x - p0.x) / 6;
      const cp1y = p1.y + (p2.y - p0.y) / 6;
      const cp2x = p2.x - (p3.x - p1.x) / 6;
      const cp2y = p2.y - (p3.y - p1.y) / 6;

      path += ` C ${cp1x} ${cp1y}, ${cp2x} ${cp2y}, ${p2.x} ${p2.y}`;
    }

    const firstPoint = points[0];
    const lastPoint = points[points.length - 1];
    const baselineY = height - paddingBottom;
    const areaPath = `${path} L ${lastPoint.x} ${baselineY} L ${firstPoint.x} ${baselineY} Z`;

    return { path, areaPath, points, maxY };
  }, [filteredDailyStats, stats.gross_volume]);

  const activeCumulPoint = hoveredCumulIndex !== null ? cumulChartData.points[hoveredCumulIndex] : null;
  const activeDailyPoint = hoveredDailyIndex !== null ? filteredDailyStats[hoveredDailyIndex] : null;

  return (
    <div className="mb-10 space-y-6">
      {/* 1. Header with Title & Period Filter Pills */}
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-4 p-5 rounded-3xl bg-slate-200/70 dark:bg-[#12161D] border border-slate-300 dark:border-white/10 shadow-sm">
        <div>
          <div className="flex items-center gap-2 mb-1">
            <span className="w-2.5 h-2.5 rounded-full bg-emerald-500 animate-pulse" />
            <h3 className="text-lg font-bold text-slate-900 dark:text-white font-mono flex items-center gap-2">
              <TrendingUp className="w-5 h-5 text-emerald-500" />
              <span>Аналитика выручки и темп продаж</span>
            </h3>
          </div>
          <p className="text-xs text-slate-500 dark:text-slate-400 font-mono">
            Общий график заработка, динамика покупок по дням и структура тарифов потока
          </p>
        </div>

        {/* Controls: Period and View Mode */}
        <div className="flex flex-wrap items-center gap-2">
          {/* Period selector */}
          <div className="flex items-center p-1 rounded-2xl bg-slate-300/60 dark:bg-black/40 border border-slate-300 dark:border-white/10">
            {(
              [
                { id: "7d", label: "7 дней" },
                { id: "14d", label: "14 дней" },
                { id: "30d", label: "30 дней" },
                { id: "all", label: "Всё время" },
              ] as const
            ).map((p) => (
              <button
                key={p.id}
                onClick={() => setPeriod(p.id)}
                className={`px-3 py-1.5 rounded-xl text-xs font-mono font-bold transition-all ${
                  period === p.id
                    ? "bg-emerald-500 text-black shadow-md"
                    : "text-slate-700 dark:text-slate-400 hover:text-black dark:hover:text-white"
                }`}
              >
                {p.label}
              </button>
            ))}
          </div>

          {/* View Mode */}
          <div className="hidden sm:flex items-center p-1 rounded-2xl bg-slate-300/60 dark:bg-black/40 border border-slate-300 dark:border-white/10">
            {(
              [
                { id: "both", label: "Оба графика" },
                { id: "cumulative", label: "Накопительно" },
                { id: "daily", label: "По дням" },
              ] as const
            ).map((v) => (
              <button
                key={v.id}
                onClick={() => setChartView(v.id)}
                className={`px-3 py-1.5 rounded-xl text-xs font-mono font-medium transition-all ${
                  viewMode === v.id
                    ? "bg-amber-500 text-black shadow-md font-bold"
                    : "text-slate-700 dark:text-slate-400 hover:text-black dark:hover:text-white"
                }`}
              >
                {v.label}
              </button>
            ))}
          </div>
        </div>
      </div>

      {/* 2. Key Velocity Badges / Highlights */}
      <div className="grid grid-cols-2 lg:grid-cols-5 gap-3">
        {/* Today */}
        <div className="bento-card-dark p-4 rounded-2xl border border-white/10 flex flex-col justify-between">
          <div className="flex items-center justify-between text-xs text-slate-400 font-mono mb-2">
            <span>Сегодня</span>
            <Zap className="w-3.5 h-3.5 text-amber-400" />
          </div>
          <div className="text-xl font-black font-mono text-emerald-400">
            {(stats.today_revenue || 0).toLocaleString("ru-RU")} ₽
          </div>
          <div className="text-[11px] font-mono text-slate-400 mt-1">
            {stats.today_purchases || 0} {stats.today_purchases === 1 ? "оплата" : "оплат"}
          </div>
        </div>

        {/* Selected Period Revenue */}
        <div className="bento-card-dark p-4 rounded-2xl border border-white/10 flex flex-col justify-between">
          <div className="flex items-center justify-between text-xs text-slate-400 font-mono mb-2">
            <span>За выбранный период</span>
            <Calendar className="w-3.5 h-3.5 text-sky-400" />
          </div>
          <div className="text-xl font-black font-mono text-white">
            {periodMetrics.revenue.toLocaleString("ru-RU")} ₽
          </div>
          <div className="text-[11px] font-mono text-sky-400 mt-1">
            {periodMetrics.purchases} заказов
          </div>
        </div>

        {/* Average Order Value (AOV) */}
        <div className="bento-card-dark p-4 rounded-2xl border border-white/10 flex flex-col justify-between">
          <div className="flex items-center justify-between text-xs text-slate-400 font-mono mb-2">
            <span>Средний чек (AOV)</span>
            <Target className="w-3.5 h-3.5 text-amber-400" />
          </div>
          <div className="text-xl font-black font-mono text-amber-400">
            {(stats.average_order_value || periodMetrics.aov || 6900).toLocaleString("ru-RU")} ₽
          </div>
          <div className="text-[11px] font-mono text-slate-400 mt-1">
            Выше базового тарифа
          </div>
        </div>

        {/* Promo Usage */}
        <div className="bento-card-dark p-4 rounded-2xl border border-white/10 flex flex-col justify-between">
          <div className="flex items-center justify-between text-xs text-slate-400 font-mono mb-2">
            <span>С промокодами (-5%)</span>
            <Tag className="w-3.5 h-3.5 text-purple-400" />
          </div>
          <div className="text-xl font-black font-mono text-purple-300">
            {stats.promo_usage_count || 0} оплат
          </div>
          <div className="text-[11px] font-mono text-slate-400 mt-1">
            Сэкономлено: {(stats.promo_saved_amount || 0).toLocaleString("ru-RU")} ₽
          </div>
        </div>

        {/* Total Gross Volume */}
        <div className="col-span-2 lg:col-span-1 bento-card-dark p-4 rounded-2xl border border-emerald-500/30 bg-emerald-500/[0.05] flex flex-col justify-between">
          <div className="flex items-center justify-between text-xs text-emerald-400 font-mono mb-2">
            <span>Итоговая касса</span>
            <DollarSign className="w-3.5 h-3.5 text-emerald-400" />
          </div>
          <div className="text-xl font-black font-mono text-emerald-300">
            {stats.gross_volume.toLocaleString("ru-RU")} ₽
          </div>
          <div className="text-[11px] font-mono text-emerald-400/80 mt-1">
            Всего {stats.active_students} студентов
          </div>
        </div>
      </div>

      {/* 3. CHART 1: Cumulative Revenue Area & Line Chart */}
      {(viewMode === "both" || viewMode === "cumulative") && (
        <div className="bento-card-dark p-6 rounded-3xl border border-white/15 shadow-xl relative overflow-hidden">
          <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3 mb-6">
            <div>
              <div className="flex items-center gap-2">
                <span className="w-2 h-2 rounded-full bg-emerald-400" />
                <h4 className="text-sm font-bold font-mono text-white uppercase tracking-wider">
                  Накопительная выручка (Gross Revenue Curve)
                </h4>
              </div>
              <p className="text-xs text-slate-400 font-mono mt-0.5">
                Кривая общего объема заработанных денег с начала запуска потока
              </p>
            </div>

            {/* Live Hover Tooltip Badge */}
            {activeCumulPoint && (
              <div className="px-3 py-1.5 rounded-xl bg-emerald-500/20 border border-emerald-500/40 text-emerald-300 font-mono text-xs flex items-center gap-2">
                <span className="font-bold">{activeCumulPoint.data.date_label}:</span>
                <span className="text-white font-bold">{activeCumulPoint.data.cumulative_revenue.toLocaleString("ru-RU")} ₽</span>
                {activeCumulPoint.data.revenue > 0 && (
                  <span className="text-emerald-400 text-[11px]">
                    (+{activeCumulPoint.data.revenue.toLocaleString("ru-RU")} ₽, {activeCumulPoint.data.purchases} опл.)
                  </span>
                )}
              </div>
            )}
          </div>

          {/* SVG Area Chart */}
          <div className="w-full overflow-x-auto">
            <div className="min-w-[650px] relative">
              <svg
                viewBox="0 0 800 220"
                className="w-full h-56 overflow-visible select-none"
                onMouseLeave={() => setHoveredCumulIndex(null)}
              >
                <defs>
                  {/* Glowing emerald area gradient */}
                  <linearGradient id="revenueAreaGrad" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="0%" stopColor="#10B981" stopOpacity="0.38" />
                    <stop offset="60%" stopColor="#10B981" stopOpacity="0.10" />
                    <stop offset="100%" stopColor="#10B981" stopOpacity="0.0" />
                  </linearGradient>

                  {/* Line glow filter */}
                  <filter id="emeraldGlow" x="-20%" y="-20%" width="140%" height="140%">
                    <feDropShadow dx="0" dy="2" stdDeviation="4" floodColor="#10B981" floodOpacity="0.5" />
                  </filter>
                </defs>

                {/* Horizontal Grid Lines */}
                {[0.25, 0.5, 0.75, 1.0].map((frac, i) => {
                  const y = 220 - 35 - frac * (220 - 25 - 35);
                  const val = Math.round(cumulChartData.maxY * frac);
                  return (
                    <g key={i}>
                      <line
                        x1="40"
                        y1={y}
                        x2="760"
                        y2={y}
                        stroke="currentColor"
                        strokeDasharray="4 4"
                        className="text-white/[0.08]"
                      />
                      <text
                        x="35"
                        y={y + 4}
                        textAnchor="end"
                        className="text-[10px] font-mono fill-slate-500 select-none"
                      >
                        {val >= 1000 ? `${Math.round(val / 1000)}k ₽` : `${val} ₽`}
                      </text>
                    </g>
                  );
                })}

                {/* Area Gradient Fill */}
                {cumulChartData.areaPath && (
                  <path d={cumulChartData.areaPath} fill="url(#revenueAreaGrad)" />
                )}

                {/* Glowing Smooth Line */}
                {cumulChartData.path && (
                  <path
                    d={cumulChartData.path}
                    fill="none"
                    stroke="#10B981"
                    strokeWidth="3.5"
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    filter="url(#emeraldGlow)"
                  />
                )}

                {/* Vertical Crosshair Line when hovered */}
                {activeCumulPoint && (
                  <g>
                    <line
                      x1={activeCumulPoint.x}
                      y1="25"
                      x2={activeCumulPoint.x}
                      y2="185"
                      stroke="#10B981"
                      strokeWidth="1.5"
                      strokeDasharray="3 3"
                      className="opacity-70"
                    />
                    <circle
                      cx={activeCumulPoint.x}
                      cy={activeCumulPoint.y}
                      r="6.5"
                      fill="#10B981"
                      stroke="#090B0E"
                      strokeWidth="2.5"
                      className="animate-pulse"
                    />
                    <circle
                      cx={activeCumulPoint.x}
                      cy={activeCumulPoint.y}
                      r="12"
                      fill="none"
                      stroke="#10B981"
                      strokeWidth="1.5"
                      className="opacity-40"
                    />
                  </g>
                )}

                {/* Interactive Hit Areas and Dots */}
                {cumulChartData.points.map((pt, idx) => {
                  const isHovered = hoveredCumulIndex === idx;
                  const hasSales = pt.data.purchases > 0;
                  return (
                    <g key={idx}>
                      {/* Transparent wide rect for hover capture */}
                      <rect
                        x={pt.x - 14}
                        y="10"
                        width="28"
                        height="200"
                        fill="transparent"
                        className="cursor-pointer"
                        onMouseEnter={() => setHoveredCumulIndex(idx)}
                      />

                      {/* Dot on line if purchase happened or hovered */}
                      {(hasSales || isHovered) && (
                        <circle
                          cx={pt.x}
                          cy={pt.y}
                          r={isHovered ? 6 : 3.5}
                          fill={isHovered ? "#38BDF8" : hasSales ? "#F59E0B" : "#10B981"}
                          stroke="#090B0E"
                          strokeWidth="2"
                          className="transition-all duration-150"
                        />
                      )}
                    </g>
                  );
                })}

                {/* X-axis Date Labels */}
                {cumulChartData.points.map((pt, idx) => {
                  // Only show every Nth label to prevent clutter
                  const total = cumulChartData.points.length;
                  const step = total > 20 ? 4 : total > 10 ? 2 : 1;
                  const isLast = idx === total - 1;
                  const isFirst = idx === 0;

                  if (idx % step !== 0 && !isLast && !isFirst) return null;

                  return (
                    <text
                      key={idx}
                      x={pt.x}
                      y="208"
                      textAnchor="middle"
                      className="text-[10px] font-mono fill-slate-400 select-none"
                    >
                      {pt.data.date_label}
                    </text>
                  );
                })}
              </svg>
            </div>
          </div>
        </div>
      )}

      {/* 4. CHART 2: Daily Sales & Revenue Bar Chart */}
      {(viewMode === "both" || viewMode === "daily") && (
        <div className="bento-card-dark p-6 rounded-3xl border border-white/15 shadow-xl">
          <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3 mb-6">
            <div>
              <div className="flex items-center gap-2">
                <span className="w-2 h-2 rounded-full bg-amber-400" />
                <h4 className="text-sm font-bold font-mono text-white uppercase tracking-wider">
                  Продажи и выручка по дням (Daily Volume & Orders)
                </h4>
              </div>
              <p className="text-xs text-slate-400 font-mono mt-0.5">
                Сколько оплат и сколько денег приносил каждый день
              </p>
            </div>

            {/* Live Hover Daily Tooltip */}
            {activeDailyPoint && (
              <div className="px-3 py-1.5 rounded-xl bg-amber-500/20 border border-amber-500/40 text-amber-300 font-mono text-xs flex items-center gap-2">
                <span className="font-bold">{activeDailyPoint.date_label}:</span>
                <span className="text-white font-bold">{activeDailyPoint.revenue.toLocaleString("ru-RU")} ₽</span>
                <span className="text-amber-400">
                  ({activeDailyPoint.purchases} {activeDailyPoint.purchases === 1 ? "оплата" : "оплат"})
                </span>
                {(activeDailyPoint.vip_count || 0) > 0 && (
                  <span className="px-1.5 py-0.2 rounded bg-sky-500/30 text-sky-300 text-[10px] font-bold">
                    VIP: {activeDailyPoint.vip_count}
                  </span>
                )}
              </div>
            )}
          </div>

          {/* Bar Chart Container */}
          <div className="w-full overflow-x-auto pb-2">
            <div className="min-w-[650px] h-48 flex items-end gap-1.5 sm:gap-2 px-2 pt-6 pb-6 border-b border-white/10 relative">
              {filteredDailyStats.map((item, idx) => {
                const heightPct = Math.max((item.revenue / maxDailyRevenue) * 100, item.revenue > 0 ? 8 : 2);
                const isHovered = hoveredDailyIndex === idx;
                const hasVip = (item.vip_count || 0) > 0;

                return (
                  <div
                    key={idx}
                    className="flex-1 flex flex-col items-center justify-end h-full group relative cursor-pointer"
                    onMouseEnter={() => setHoveredDailyIndex(idx)}
                    onMouseLeave={() => setHoveredDailyIndex(null)}
                  >
                    {/* Tooltip on bar hover */}
                    {isHovered && (
                      <div className="absolute -top-12 z-20 px-2.5 py-1 rounded-lg bg-black/90 border border-amber-500/50 text-[11px] font-mono text-white whitespace-nowrap shadow-xl pointer-events-none">
                        <div className="font-bold text-amber-400">{item.revenue.toLocaleString("ru-RU")} ₽</div>
                        <div className="text-[10px] text-slate-300">
                          {item.purchases} опл. {hasVip ? `(из них VIP: ${item.vip_count})` : ""}
                        </div>
                      </div>
                    )}

                    {/* Order count badge on top of active bar */}
                    {item.purchases > 0 && (
                      <span
                        className={`text-[9px] font-mono font-bold mb-1 transition-all ${
                          isHovered ? "text-amber-300 scale-110" : "text-slate-400"
                        }`}
                      >
                        +{item.purchases}
                      </span>
                    )}

                    {/* Bar Column */}
                    <div
                      style={{ height: `${heightPct}%` }}
                      className={`w-full rounded-t-lg transition-all duration-200 ${
                        item.revenue === 0
                          ? "bg-white/[0.05]"
                          : hasVip
                          ? isHovered
                            ? "bg-gradient-to-t from-emerald-400 to-sky-400 shadow-lg shadow-sky-500/30"
                            : "bg-gradient-to-t from-emerald-500 to-sky-500"
                          : isHovered
                          ? "bg-gradient-to-t from-emerald-400 to-amber-400 shadow-lg shadow-amber-500/30"
                          : "bg-gradient-to-t from-emerald-600 to-amber-500"
                      }`}
                    />

                    {/* Date label at bottom */}
                    <span className="text-[9px] font-mono text-slate-500 mt-2 truncate w-full text-center">
                      {item.date_label.split(" ")[0]}
                    </span>
                  </div>
                );
              })}
            </div>

            {/* Daily Legend */}
            <div className="flex flex-wrap items-center justify-between text-xs font-mono text-slate-400 mt-3 px-2">
              <div className="flex items-center gap-4">
                <div className="flex items-center gap-1.5">
                  <span className="w-2.5 h-2.5 rounded bg-emerald-500" />
                  <span>Акселератор (6 555 – 6 900 ₽)</span>
                </div>
                <div className="flex items-center gap-1.5">
                  <span className="w-2.5 h-2.5 rounded bg-sky-500" />
                  <span>VIP (14 155 – 14 900 ₽)</span>
                </div>
              </div>
              <div className="text-slate-500 text-[11px]">
                Пик периода: {maxDailyRevenue.toLocaleString("ru-RU")} ₽ в день
              </div>
            </div>
          </div>
        </div>
      )}

      {/* 5. Tier Distribution & Revenue Breakdown Card */}
      <div className="bento-card-dark p-6 rounded-3xl border border-white/15 shadow-xl">
        <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3 mb-6">
          <div>
            <div className="flex items-center gap-2">
              <span className="w-2 h-2 rounded-full bg-sky-400" />
              <h4 className="text-sm font-bold font-mono text-white uppercase tracking-wider">
                Структура выручки по тарифам (Tier Breakdown)
              </h4>
            </div>
            <p className="text-xs text-slate-400 font-mono mt-0.5">
              Соотношение выручки и проданных мест между Акселератором и VIP
            </p>
          </div>

          <div className="text-xs font-mono text-slate-400 flex items-center gap-2">
            <ShieldCheck className="w-4 h-4 text-emerald-400" />
            <span>Базовый тариф исключён из продажи</span>
          </div>
        </div>

        {/* Visual Dual Progress Bar */}
        <div className="space-y-4">
          {(() => {
            const acc = stats.tier_breakdown?.accelerator || {
              count: 36,
              revenue: 204870,
              percentage: 70.7,
            };
            const vip = stats.tier_breakdown?.vip || {
              count: 6,
              revenue: 84930,
              percentage: 29.3,
            };
            const accPct = acc.percentage || 70.7;
            const vipPct = vip.percentage || 29.3;

            return (
              <>
                <div className="w-full h-4 rounded-full bg-black/40 overflow-hidden flex border border-white/10 p-0.5">
                  <div
                    style={{ width: `${accPct}%` }}
                    className="h-full rounded-l-full bg-gradient-to-r from-emerald-500 to-amber-500 transition-all duration-500 relative group"
                    title={`Акселератор: ${acc.revenue.toLocaleString("ru-RU")} ₽ (${accPct.toFixed(1)}%)`}
                  />
                  <div
                    style={{ width: `${vipPct}%` }}
                    className="h-full rounded-r-full bg-gradient-to-r from-sky-500 to-blue-600 transition-all duration-500 relative group"
                    title={`VIP: ${vip.revenue.toLocaleString("ru-RU")} ₽ (${vipPct.toFixed(1)}%)`}
                  />
                </div>

                {/* Cards for each tier */}
                <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 pt-2">
                  {/* Accelerator Tier */}
                  <div className="p-4 rounded-2xl bg-white/[0.03] border border-emerald-500/20 hover:border-emerald-500/40 transition-all">
                    <div className="flex items-center justify-between mb-2">
                      <div className="flex items-center gap-2">
                        <span className="w-2.5 h-2.5 rounded-full bg-emerald-500" />
                        <span className="font-mono font-bold text-white text-sm">Тариф «Акселератор»</span>
                      </div>
                      <span className="text-xs font-mono font-bold text-emerald-400 bg-emerald-500/10 px-2 py-0.5 rounded-full border border-emerald-500/20">
                        {accPct.toFixed(1)}% кассы
                      </span>
                    </div>
                    <div className="flex items-baseline justify-between mt-2">
                      <div>
                        <div className="text-2xl font-black font-mono text-white">
                          {acc.revenue.toLocaleString("ru-RU")} ₽
                        </div>
                        <div className="text-xs font-mono text-slate-400 mt-0.5">
                          {acc.count} студентов (6 900 ₽ / 6 555 ₽)
                        </div>
                      </div>
                      <div className="text-right text-xs font-mono text-slate-400">
                        Личный аудит + 45-мин интервью
                      </div>
                    </div>
                  </div>

                  {/* VIP Tier */}
                  <div className="p-4 rounded-2xl bg-white/[0.03] border border-sky-500/20 hover:border-sky-500/40 transition-all">
                    <div className="flex items-center justify-between mb-2">
                      <div className="flex items-center gap-2">
                        <span className="w-2.5 h-2.5 rounded-full bg-sky-400" />
                        <span className="font-mono font-bold text-white text-sm">Тариф «VIP»</span>
                      </div>
                      <span className="text-xs font-mono font-bold text-sky-400 bg-sky-500/10 px-2 py-0.5 rounded-full border border-sky-500/20">
                        {vipPct.toFixed(1)}% кассы
                      </span>
                    </div>
                    <div className="flex items-baseline justify-between mt-2">
                      <div>
                        <div className="text-2xl font-black font-mono text-white">
                          {vip.revenue.toLocaleString("ru-RU")} ₽
                        </div>
                        <div className="text-xs font-mono text-slate-400 mt-0.5">
                          {vip.count} студентов (14 900 ₽ / 14 155 ₽)
                        </div>
                      </div>
                      <div className="text-right text-xs font-mono text-slate-400">
                        3 Zoom-симуляции + личный чат
                      </div>
                    </div>
                  </div>
                </div>
              </>
            );
          })()}
        </div>
      </div>
    </div>
  );
}
