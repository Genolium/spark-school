"use client";

import React, { useState, useEffect } from "react";
import { Clock, AlertCircle, ArrowUpRight, Flame, ShieldAlert, CheckCircle2 } from "lucide-react";
import { motion } from "framer-motion";

interface TimeLeft {
  days: number;
  hours: number;
  minutes: number;
  seconds: number;
  isExpired: boolean;
}

// Deadline: 20 ноября 2026 года в 23:59:59 по Московскому времени (UTC+3)
const DEADLINE_TARGET = new Date("2026-11-20T23:59:59+03:00").getTime();

function calculateTimeLeft(): TimeLeft {
  const now = new Date().getTime();
  const difference = DEADLINE_TARGET - now;

  if (difference <= 0) {
    return {
      days: 0,
      hours: 0,
      minutes: 0,
      seconds: 0,
      isExpired: true,
    };
  }

  return {
    days: Math.floor(difference / (1000 * 60 * 60 * 24)),
    hours: Math.floor((difference / (1000 * 60 * 60)) % 24),
    minutes: Math.floor((difference / 1000 / 60) % 60),
    seconds: Math.floor((difference / 1000) % 60),
    isExpired: false,
  };
}

interface DeadlineCountdownProps {
  variant?: "banner" | "card" | "hero";
  className?: string;
  onOpenPayment?: () => void;
}

export function DeadlineCountdown({
  variant = "card",
  className = "",
  onOpenPayment,
}: DeadlineCountdownProps) {
  const [mounted, setMounted] = useState(false);
  const [timeLeft, setTimeLeft] = useState<TimeLeft>(calculateTimeLeft());

  useEffect(() => {
    setMounted(true);
    setTimeLeft(calculateTimeLeft());

    const timer = setInterval(() => {
      setTimeLeft(calculateTimeLeft());
    }, 1000);

    return () => clearInterval(timer);
  }, []);

  // SSR hydration placeholder
  if (!mounted) {
    return (
      <div className={`p-5 rounded-2xl bento-card-dark border border-emerald-500/20 text-center ${className}`}>
        <div className="flex items-center justify-center gap-2 text-xs font-mono text-emerald-600 dark:text-emerald-400">
          <Clock className="w-4 h-4 animate-spin" />
          <span>Загрузка таймера дедлайна...</span>
        </div>
      </div>
    );
  }

  // EXPIRED STATE: Приём заявок закрыт
  if (timeLeft.isExpired) {
    return (
      <motion.div
        initial={{ opacity: 0, y: 10 }}
        animate={{ opacity: 1, y: 0 }}
        className={`p-6 sm:p-7 rounded-3xl bg-rose-500/10 dark:bg-rose-950/30 border-2 border-rose-500/30 text-slate-900 dark:text-white relative overflow-hidden shadow-xl ${className}`}
      >
        <div className="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4">
          <div className="flex items-start gap-3.5">
            <div className="w-10 h-10 rounded-2xl bg-rose-500/20 text-rose-600 dark:text-rose-400 flex items-center justify-center shrink-0 mt-0.5">
              <ShieldAlert className="w-5 h-5" />
            </div>
            <div>
              <div className="inline-flex items-center gap-2 px-2.5 py-0.5 rounded-full text-[11px] font-mono font-bold uppercase tracking-wider bg-rose-500/20 text-rose-700 dark:text-rose-300 border border-rose-500/30 mb-1.5">
                <span className="w-1.5 h-1.5 rounded-full bg-rose-500" />
                Приём заявок завершён
              </div>
              <h3 className="font-heading text-lg sm:text-xl font-bold text-slate-900 dark:text-white">
                Подача заявок на грант SPARK 2027 закрыта
              </h3>
              <p className="text-xs sm:text-sm text-slate-600 dark:text-slate-300 font-doc mt-1 max-w-xl leading-relaxed">
                Дедлайн приёма документов (20 ноября 23:59 МСК) истёк. Набор в текущий поток завершён.
                Напишите так называемому Илю в Telegram, чтобы подать заявку в лист ожидания следующего набора.
              </p>
            </div>
          </div>

          <a
            href="https://t.me/ilyan_vas"
            target="_blank"
            rel="noopener noreferrer"
            className="w-full sm:w-auto px-5 py-3 rounded-2xl bg-rose-600 hover:bg-rose-700 text-white font-mono text-xs font-bold flex items-center justify-center gap-2 transition-transform active:scale-95 shrink-0 shadow-lg"
          >
            <span>Написать в Telegram</span>
            <ArrowUpRight className="w-4 h-4" />
          </a>
        </div>
      </motion.div>
    );
  }

  // ACTIVE COUNTDOWN STATE
  const pad = (n: number) => String(n).padStart(2, "0");

  if (variant === "hero") {
    return (
      <div className={`inline-flex flex-col sm:flex-row items-center gap-3 p-2.5 sm:p-3 rounded-2xl bg-black/60 backdrop-blur-2xl border border-white/25 text-white shadow-2xl ${className}`}>
        <div className="flex items-center gap-2 px-2 text-xs font-mono">
          <span className="w-2 h-2 rounded-full bg-amber-400 animate-pulse" />
          <span className="text-amber-300 font-bold uppercase tracking-wider text-[11px]">
            Дедлайн: 20 ноября 23:59
          </span>
        </div>
        <div className="flex items-center gap-1.5 font-mono text-xs sm:text-sm">
          <div className="px-2 py-1 rounded-lg bg-white/10 font-bold">{pad(timeLeft.days)}д</div>
          <span className="text-white/40">:</span>
          <div className="px-2 py-1 rounded-lg bg-white/10 font-bold">{pad(timeLeft.hours)}ч</div>
          <span className="text-white/40">:</span>
          <div className="px-2 py-1 rounded-lg bg-white/10 font-bold">{pad(timeLeft.minutes)}м</div>
          <span className="text-white/40">:</span>
          <div className="px-2 py-1 rounded-lg bg-amber-500/20 text-amber-300 font-bold border border-amber-500/30">
            {pad(timeLeft.seconds)}с
          </div>
        </div>
      </div>
    );
  }

  return (
    <motion.div
      initial={{ opacity: 0, y: 15 }}
      whileInView={{ opacity: 1, y: 0 }}
      viewport={{ once: true }}
      transition={{ duration: 0.5 }}
      className={`p-6 sm:p-8 rounded-3xl bento-card-dark border-2 border-emerald-500/25 dark:border-emerald-500/30 bg-gradient-to-br from-emerald-500/[0.04] via-transparent to-teal-500/[0.04] relative overflow-hidden shadow-xl ${className}`}
    >
      {/* Decorative ambient glow */}
      <div className="absolute -top-12 -right-12 w-48 h-48 rounded-full bg-emerald-500/10 blur-3xl pointer-events-none" />

      <div className="flex flex-col lg:flex-row items-start lg:items-center justify-between gap-6 relative z-10">
        
        {/* Left: Info Title */}
        <div className="space-y-1.5 max-w-xl">
          <div className="inline-flex items-center gap-2 px-3 py-1 rounded-full text-xs font-mono font-bold uppercase tracking-wider bg-emerald-500/15 text-emerald-700 dark:text-emerald-300 border border-emerald-500/30">
            <Flame className="w-3.5 h-3.5 text-amber-500 animate-pulse" />
            <span>Официальный дедлайн подачи</span>
          </div>

          <h3 className="font-heading text-xl sm:text-2xl font-bold text-slate-900 dark:text-white tracking-tight">
            До закрытия приёма заявок: <span className="text-emerald-600 dark:text-emerald-400 font-extrabold">20 ноября в 23:59</span>
          </h3>

          <p className="text-xs sm:text-sm text-slate-600 dark:text-slate-300 font-doc leading-relaxed">
            После дедлайна регистрация на грантовую программу SPARK 2027 блокируется организаторами. Успей упаковать сильное резюме американского стандарта и победное эссе до окончания таймера.
          </p>
        </div>

        {/* Right: Digital Countdown Grid */}
        <div className="flex flex-col sm:flex-row items-stretch sm:items-center gap-4 w-full lg:w-auto">
          <div className="grid grid-cols-4 gap-2 sm:gap-3 text-center">
            
            {/* Days */}
            <div className="p-2.5 sm:p-3.5 rounded-2xl bg-white/80 dark:bg-black/40 border border-emerald-500/20 backdrop-blur-md shadow-inner">
              <div className="text-2xl sm:text-3xl font-black font-mono text-slate-900 dark:text-white tracking-tight">
                {pad(timeLeft.days)}
              </div>
              <div className="text-[10px] sm:text-xs font-mono uppercase text-slate-500 dark:text-slate-400 font-semibold mt-0.5">
                дней
              </div>
            </div>

            {/* Hours */}
            <div className="p-2.5 sm:p-3.5 rounded-2xl bg-white/80 dark:bg-black/40 border border-emerald-500/20 backdrop-blur-md shadow-inner">
              <div className="text-2xl sm:text-3xl font-black font-mono text-slate-900 dark:text-white tracking-tight">
                {pad(timeLeft.hours)}
              </div>
              <div className="text-[10px] sm:text-xs font-mono uppercase text-slate-500 dark:text-slate-400 font-semibold mt-0.5">
                часов
              </div>
            </div>

            {/* Minutes */}
            <div className="p-2.5 sm:p-3.5 rounded-2xl bg-white/80 dark:bg-black/40 border border-emerald-500/20 backdrop-blur-md shadow-inner">
              <div className="text-2xl sm:text-3xl font-black font-mono text-slate-900 dark:text-white tracking-tight">
                {pad(timeLeft.minutes)}
              </div>
              <div className="text-[10px] sm:text-xs font-mono uppercase text-slate-500 dark:text-slate-400 font-semibold mt-0.5">
                минут
              </div>
            </div>

            {/* Seconds */}
            <div className="p-2.5 sm:p-3.5 rounded-2xl bg-emerald-500/10 dark:bg-emerald-500/15 border border-emerald-500/30 backdrop-blur-md shadow-inner">
              <div className="text-2xl sm:text-3xl font-black font-mono text-emerald-600 dark:text-emerald-400 tracking-tight">
                {pad(timeLeft.seconds)}
              </div>
              <div className="text-[10px] sm:text-xs font-mono uppercase text-emerald-700 dark:text-emerald-300 font-semibold mt-0.5">
                секунд
              </div>
            </div>

          </div>

          {onOpenPayment && (
            <button
              type="button"
              onClick={onOpenPayment}
              className="btn-primary-mono px-5 py-3.5 text-xs sm:text-sm font-bold flex items-center justify-center gap-2 active:scale-95 shadow-lg group"
            >
              <span>Подать заявку</span>
              <ArrowUpRight className="w-4 h-4 transition-transform group-hover:translate-x-0.5 group-hover:-translate-y-0.5" />
            </button>
          )}
        </div>

      </div>
    </motion.div>
  );
}
