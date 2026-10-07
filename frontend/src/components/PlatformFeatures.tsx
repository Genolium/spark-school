"use client";

import React from "react";
import { motion } from "framer-motion";
import { Send, FileText, Video, Award, BookmarkCheck } from "lucide-react";

export function PlatformFeatures() {
  return (
    <section id="features" className="relative w-full max-w-[1520px] mx-auto px-3 sm:px-6 py-16 sm:py-24 scroll-mt-20 sm:scroll-mt-24">
      {/* Section Header */}
      <div className="max-w-3xl mb-12 sm:mb-16">
        <div className="inline-flex items-center gap-2 px-3.5 py-1.5 rounded-full bento-pill text-xs font-mono text-emerald-800 dark:text-emerald-300 mb-4">
          <span className="w-2 h-2 rounded-full bg-emerald-500" />
          <span>Архитектура продукта</span>
        </div>
        <h2 className="font-editorial text-3xl sm:text-5xl lg:text-6xl font-bold tracking-tight text-slate-900 dark:text-white leading-[1.1]">
          Всё, что нужно для победы, <br />
          <span className="text-emerald-600 dark:text-emerald-400 font-serif italic">в закрытом комьюнити</span>
        </h2>
        <p className="mt-4 text-base text-slate-600 dark:text-slate-300 leading-relaxed font-normal">
          Закрытый Telegram-канал по приглашению: структурированная база знаний, видеоразборы, победные архивы и закрытый чат участников.
        </p>
      </div>

      {/* Asymmetrical Spatial Bento Grid */}
      <div className="grid grid-cols-1 md:grid-cols-12 gap-6">
        
        {/* Module 1: Closed Telegram Channel & Community (Large span 7) */}
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          whileInView={{ opacity: 1, y: 0 }}
          viewport={{ once: true }}
          transition={{ duration: 0.5 }}
          className="md:col-span-7 bento-card-dark p-7 sm:p-10 flex flex-col justify-between relative overflow-hidden group rounded-2xl border border-emerald-500/20 dark:border-white/20 shadow-[0_12px_35px_rgba(16,185,129,0.06)] dark:shadow-none"
        >
          <div className="flex items-center justify-between mb-8">
            <span className="bento-pill px-3 py-1 rounded-full text-xs font-mono text-emerald-800 dark:text-slate-300">
              01 • Приватный канал + Чат
            </span>
            <div className="w-10 h-10 rounded-xl bg-emerald-500/10 text-emerald-700 dark:bg-white/10 dark:text-white flex items-center justify-center">
              <Send className="w-5 h-5" />
            </div>
          </div>

          <div>
            <h3 className="font-editorial text-2xl sm:text-3xl font-bold text-slate-900 dark:text-white mb-3">
              Закрытый Telegram-канал и комьюнити
            </h3>
            <p className="font-doc text-sm sm:text-base text-slate-600 dark:text-slate-300 leading-relaxed max-w-xl mb-6">
              Концентрированные гайды, видеоразборы, пошаговые инструкции и чек-листы готовности каждого документа.
            </p>

            {/* Clean Typographic Spec Pills */}
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-3 text-xs font-mono text-slate-600 dark:text-slate-300">
              <div className="p-3.5 rounded-xl bg-emerald-500/[0.04] border border-emerald-500/15 dark:bg-white/[0.04] dark:border-white/10 flex items-center justify-between">
                <span className="text-slate-500 dark:text-slate-400">Формат</span>
                <span className="text-slate-900 dark:text-white font-medium">Закрытый TG-канал</span>
              </div>
              <div className="p-3.5 rounded-xl bg-emerald-500/[0.04] border border-emerald-500/15 dark:bg-white/[0.04] dark:border-white/10 flex items-center justify-between">
                <span className="text-slate-500 dark:text-slate-400">Доступ</span>
                <span className="text-slate-900 dark:text-white font-medium">Приватный инвайт</span>
              </div>
              <div className="p-3.5 rounded-xl bg-emerald-500/[0.04] border border-emerald-500/15 dark:bg-white/[0.04] dark:border-white/10 flex items-center justify-between">
                <span className="text-slate-500 dark:text-slate-400">Материалы</span>
                <span className="text-slate-900 dark:text-white font-medium">DOCX / PDF шаблоны</span>
              </div>
              <div className="p-3.5 rounded-xl bg-emerald-500/[0.04] border border-emerald-500/15 dark:bg-white/[0.04] dark:border-white/10 flex items-center justify-between">
                <span className="text-slate-500 dark:text-slate-400">Комьюнити</span>
                <span className="text-slate-900 dark:text-white font-medium">Чат участников</span>
              </div>
            </div>
          </div>

          <div className="mt-8 pt-6 border-t border-emerald-500/15 dark:border-white/10 hidden sm:flex items-center justify-between text-xs font-mono text-slate-500 dark:text-slate-400">
            <span>Доступ с Mac, Windows, iPhone, Android через Telegram</span>
            <span className="text-emerald-600 dark:text-emerald-400 font-medium">● 100% Mobile Ready</span>
          </div>
        </motion.div>

        {/* Module 2: Real Winning Materials (Span 5) */}
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          whileInView={{ opacity: 1, y: 0 }}
          viewport={{ once: true }}
          transition={{ duration: 0.5, delay: 0.1 }}
          className="md:col-span-5 bento-card-dark p-7 sm:p-10 flex flex-col justify-between relative overflow-hidden group rounded-2xl border border-emerald-500/20 dark:border-white/20 shadow-[0_12px_35px_rgba(16,185,129,0.06)] dark:shadow-none"
        >
          <div className="flex items-center justify-between mb-8">
            <span className="bento-pill px-3 py-1 rounded-full text-xs font-mono text-emerald-800 dark:text-slate-300">
              02 • Архивы финалиста
            </span>
            <div className="w-10 h-10 rounded-xl bg-emerald-500/10 text-emerald-700 dark:bg-white/10 dark:text-white flex items-center justify-center">
              <FileText className="w-5 h-5" />
            </div>
          </div>

          <div>
            <h3 className="font-editorial text-2xl sm:text-3xl font-bold text-slate-900 dark:text-white mb-3">
              Реальные победные материалы
            </h3>
            <p className="font-doc text-sm text-slate-600 dark:text-slate-300 leading-relaxed mb-6">
              Полные материалы победителя программы SPARK 2026.
            </p>

            <div className="space-y-2 text-xs font-mono">
              <div className="p-3 rounded-xl bg-emerald-500/[0.04] border border-emerald-500/15 dark:bg-white/5 dark:border-white/10 flex items-center justify-between text-slate-800 dark:text-slate-200">
                <span>Шаблоны материалов</span>
                <span className="text-emerald-700 dark:text-emerald-400 font-bold">US Standard</span>
              </div>
              <div className="p-3 rounded-xl bg-emerald-500/[0.04] border border-emerald-500/15 dark:bg-white/5 dark:border-white/10 flex items-center justify-between text-slate-800 dark:text-slate-200">
                <span>100% Score</span>
                <span className="text-emerald-700 dark:text-emerald-400 font-bold">Verified</span>
              </div>
            </div>
          </div>

          <div className="mt-8 pt-6 border-t border-emerald-500/15 dark:border-white/10 text-xs font-mono text-slate-500 dark:text-slate-400">
            Никакой воды: бери структуру и адаптируй под свой опыт.
          </div>
        </motion.div>

        {/* Module 3: Exclusive Real Interview Analysis (Span 4) */}
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          whileInView={{ opacity: 1, y: 0 }}
          viewport={{ once: true }}
          transition={{ duration: 0.5, delay: 0.15 }}
          className="md:col-span-4 bento-card-dark p-7 sm:p-8 flex flex-col justify-between group rounded-2xl border border-emerald-500/20 dark:border-white/20 shadow-[0_12px_35px_rgba(16,185,129,0.06)] dark:shadow-none"
        >
          <div className="flex items-center justify-between mb-6">
            <span className="bento-pill px-3 py-1 rounded-full text-xs font-mono text-emerald-800 dark:text-slate-300">
              03 • Эксклюзив
            </span>
            <div className="w-9 h-9 rounded-xl bg-emerald-500/10 text-emerald-700 dark:bg-white/10 dark:text-white flex items-center justify-center">
              <Video className="w-4 h-4" />
            </div>
          </div>

          <div>
            <h3 className="font-editorial text-xl font-bold text-slate-900 dark:text-white mb-2">
              Разбор реального интервью
            </h3>
            <p className="font-doc text-xs sm:text-sm text-slate-600 dark:text-slate-300 leading-relaxed">
              Видеозапись собеседования с американскими интервьюерами с покадровым таймлайн-разбором стресс-вопросов и удачных формулировок.
            </p>
          </div>

          <div className="mt-6 pt-4 border-t border-emerald-500/15 dark:border-white/10 flex items-center justify-between text-xs font-mono text-slate-600 dark:text-slate-300">
            <span>Живой разбор 1-на-1</span>
            <span className="text-slate-900 dark:text-white font-medium">45 мин Zoom</span>
          </div>
        </motion.div>

        {/* Module 4: Duolingo 130+ Framework (Span 4) */}
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          whileInView={{ opacity: 1, y: 0 }}
          viewport={{ once: true }}
          transition={{ duration: 0.5, delay: 0.2 }}
          className="md:col-span-4 bento-card-dark p-7 sm:p-8 flex flex-col justify-between group rounded-2xl border border-emerald-500/20 dark:border-white/20 shadow-[0_12px_35px_rgba(16,185,129,0.06)] dark:shadow-none"
        >
          <div className="flex items-center justify-between mb-6">
            <span className="bento-pill px-3 py-1 rounded-full text-xs font-mono text-emerald-800 dark:text-slate-300">
              04 • Языковой экзамен
            </span>
            <div className="w-9 h-9 rounded-xl bg-emerald-500/10 text-emerald-700 dark:bg-white/10 dark:text-white flex items-center justify-center">
              <Award className="w-4 h-4" />
            </div>
          </div>

          <div>
            <h3 className="font-editorial text-xl font-bold text-slate-900 dark:text-white mb-2">
              Гайд по Duolingo на 130+
            </h3>
            <p className="font-doc text-xs sm:text-sm text-slate-600 dark:text-slate-300 leading-relaxed">
              Алгоритмы сдачи языкового экзамена от профессионального приглашённого преподавателя английского.
            </p>
          </div>

          <div className="mt-6 pt-4 border-t border-emerald-500/15 dark:border-white/10 flex items-center justify-between text-xs font-mono text-slate-600 dark:text-slate-300">
            <span>Результат автора:</span>
            <span className="text-slate-900 dark:text-white font-bold font-mono">135 / 160 (C1)</span>
          </div>
        </motion.div>

        {/* Module 5: Bonus Track INSPIRE (Span 4) */}
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          whileInView={{ opacity: 1, y: 0 }}
          viewport={{ once: true }}
          transition={{ duration: 0.5, delay: 0.25 }}
          className="md:col-span-4 bento-card-dark p-7 sm:p-8 flex flex-col justify-between group rounded-2xl border border-emerald-500/20 dark:border-white/20 shadow-[0_12px_35px_rgba(16,185,129,0.06)] dark:shadow-none"
        >
          <div className="flex items-center justify-between mb-6">
            <span className="bento-pill px-3 py-1 rounded-full text-xs font-mono text-emerald-800 dark:text-slate-300">
              05 • Бонусный трек
            </span>
            <div className="w-9 h-9 rounded-xl bg-emerald-500/10 text-emerald-700 dark:bg-white/10 dark:text-white flex items-center justify-center">
              <BookmarkCheck className="w-4 h-4" />
            </div>
          </div>

          <div>
            <h3 className="font-editorial text-xl font-bold text-slate-900 dark:text-white mb-2">
              Грантовый трек INSPIRE
            </h3>
            <p className="font-doc text-xs sm:text-sm text-slate-600 dark:text-slate-300 leading-relaxed">
              Бонусный трек для тех, кто хочет удвоить шансы и податься сразу на две программы академического обмена США (доступен за доп. плату).
            </p>
          </div>

          <div className="mt-6 pt-4 border-t border-emerald-500/15 dark:border-white/10 flex items-center justify-between text-xs font-mono text-slate-600 dark:text-slate-300">
            <span>2 программы в 1 комьюнити</span>
            <span className="text-slate-500 dark:text-slate-400">За доп. плату</span>
          </div>
        </motion.div>

      </div>
    </section>
  );
}
