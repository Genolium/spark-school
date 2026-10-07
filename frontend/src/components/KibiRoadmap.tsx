"use client";

import React, { useState } from "react";
import { BlurImage } from "./BlurImage";
import { motion, AnimatePresence } from "framer-motion";
import { CheckCircle2, ChevronRight, Plane, Compass, FileCheck, Video, Users, ShieldAlert, Check } from "lucide-react";

export function KibiRoadmap() {
  const [activeStep, setActiveStep] = useState(0);

  const steps = [
    {
      index: "01",
      icon: Compass,
      title: "Архитектура легенды",
      subtitle: "Фундамент и позиционирование",
      tag: "Stage 01 • Анализ",
      tagClass: "bento-pill text-slate-300",
      image: "/stage-01.webp",
      shortDesc: "Распаковка твоего бэкграунда, поиск уникального угла подачи и миссии, которая отзовётся у американской комиссии.",
      deliverables: [
        "Аудит твоих сильных сторон и увлечений",
        "Карта соответствия ценностям Госдепа США",
        "Устранение синдрома самозванца: как упаковать свой опыт",
      ],
      result: "Чёткое позиционирование твоей кандидатуры, выделяющее тебя среди 2000+ подающихся на программу.",
    },
    {
      index: "02",
      icon: FileCheck,
      title: "Пакет документов",
      subtitle: "Резюме и эссе",
      tag: "Stage 02 • Копирайтинг",
      tagClass: "bento-pill text-slate-300",
      image: "/stage-02.webp",
      shortDesc: "Создание документов американского стандарта. Никаких шаблонных фраз и сухого перечисления оценок.",
      deliverables: [
        "Резюме по международным стандартам",
        "Победное эссе на 500 слов с захватывающим хуком и сторителлингом",
        "Разбор реальных черновиков и финальной заявки финалиста 2026",
      ],
      result: "Готовый, вычитанный пакет документов, который комиссия дочитывает до конца с интересом.",
    },
    {
      index: "03",
      icon: Video,
      title: "Видеовизитка",
      subtitle: "Режиссура и хук 5 секунд",
      tag: "Stage 03 • Продакшн",
      tagClass: "bento-pill text-slate-300",
      image: "/stage-03.webp",
      shortDesc: "Съемка и монтаж видеовизитки на обычный смартфон. Удержание внимания комиссии с первых 5 секунд.",
      deliverables: [
        "Сценарная сетка и покадровый план ролика",
        "Постановка света, ракурса и чистого звука без дорогой аппаратуры",
        "Живая динамика: как говорить на камеру уверенно и естественно",
      ],
      result: "Видеовизитка, вызывающая мгновенную симпатию комиссии.",
    },
    {
      index: "04",
      icon: Users,
      title: "Live Mock-Interview",
      subtitle: "Боевая Zoom-симуляция 45 минут",
      tag: "Stage 04 • Практика",
      tagClass: "bento-pill text-slate-300",
      image: "/stage-04.webp",
      shortDesc: "Полноценная репетиция собеседования на программу в формате 1-на-1 с разбором.",
      deliverables: [
        "Подборка вопросов прошлых лет, разбитых на группы",
        "Отработка всех видов вопросов от интервьюеров",
        "Универсальный гайд по ответам на интервью",
      ],
      result: "Полное отсутствие страха перед экраном и готовые отточенные ответы на английском.",
    },
    {
      index: "05",
      icon: ShieldAlert,
      title: "Подача заявки на визу",
      subtitle: "DS-160, Посольство, Вылет в США",
      tag: "Stage 05 • Финал",
      tagClass: "bento-pill text-slate-300",
      image: "/stage-05.webp",
      shortDesc: "Пошаговый план прохождения американского консульства, получение визы J-1 и логистика перелёта на кампус.",
      deliverables: [
        "Безошибочное заполнение визовой анкеты DS-160",
        "Интервью для консульства: как отвечать визовому офицеру",
        "Чек-лист сборов, сим-карта, банковские карты и вылет в США",
      ],
      result: "Паспорт с вклеенной визой J-1 и билет на самолёт до твоего американского университета.",
    },
  ];

  return (
    <section id="roadmap" className="relative w-full max-w-[1520px] mx-auto px-3 sm:px-6 py-16 sm:py-24 scroll-mt-20 sm:scroll-mt-24">
      {/* Section Header */}
      <div className="flex flex-col md:flex-row md:items-end justify-between mb-12 sm:mb-16 gap-6">
        <div className="max-w-3xl">
          <div className="inline-flex items-center gap-2 px-3.5 py-1.5 rounded-full bento-pill text-xs font-mono text-emerald-800 dark:text-emerald-300 mb-4">
            <Plane className="w-3.5 h-3.5 text-sky-500 dark:text-sky-400" />
            <span>Маршрут программы • 5 этапов</span>
          </div>
          <h2 className="font-editorial text-3xl sm:text-5xl lg:text-6xl font-bold tracking-tight text-slate-900 dark:text-white leading-[1.1]">
            Маршрут от нуля <br />
            <span className="text-emerald-600 dark:text-emerald-400 font-serif italic">до вылета в США</span>
          </h2>
          <p className="mt-4 text-base text-slate-600 dark:text-slate-300 leading-relaxed font-normal">
            5 последовательных этапов курса с полным сопровождением через всю программу.
          </p>
        </div>

        {/* Status Pill */}
        <div className="flex items-center gap-2 text-xs font-mono text-emerald-800 dark:text-emerald-300 bento-pill px-4 py-2 self-start md:self-end">
          <span className="w-2 h-2 rounded-full bg-emerald-500" />
          <span>Выбирай этап для просмотра деталей</span>
        </div>
      </div>

      {/* ========================================================================= */}
      {/* KIBI STYLE DASHED FLIGHT PATH CHECKPOINTS (HORIZONTAL ROW)               */}
      {/* ========================================================================= */}
      <div className="relative mb-10">
        {/* Subtle dashed flight line */}
        <div className="hidden lg:block absolute top-7 left-12 right-12 h-0.5 border-t border-dashed border-emerald-500/25 dark:border-white/20 z-0" />

        <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-5 gap-3 sm:gap-4 relative z-10">
          {steps.map((st, idx) => {
            const isSelected = activeStep === idx;
            return (
              <button
                key={idx}
                onClick={() => setActiveStep(idx)}
                className={`bento-card-dark p-4 sm:p-5 text-left transition-all relative overflow-hidden flex flex-col justify-between rounded-xl ${
                  isSelected
                    ? "bg-emerald-500/[0.08] border-emerald-500/50 shadow-md ring-1 ring-emerald-500/20 dark:bg-white/[0.12] dark:border-white/40"
                    : "hover:bg-emerald-500/[0.03] hover:border-emerald-500/30 dark:hover:bg-white/[0.05] dark:hover:border-white/20"
                }`}
              >
                {/* Active indicator bar */}
                {isSelected && (
                  <div className="absolute top-0 left-0 right-0 h-1 bg-emerald-600 dark:bg-white" />
                )}

                <div className="flex items-center justify-between mb-4">
                  <span className="w-8 h-8 rounded-full bg-emerald-500/10 dark:bg-white/10 flex items-center justify-center font-mono font-bold text-xs text-emerald-800 dark:text-white">
                    {st.index}
                  </span>
                </div>

                <div>
                  <div className="text-sm sm:text-base font-bold text-slate-900 dark:text-white mb-1 line-clamp-1">
                    {st.title}
                  </div>
                  <div className="text-xs text-slate-600 dark:text-slate-400 line-clamp-1">
                    {st.subtitle}
                  </div>
                </div>
              </button>
            );
          })}
        </div>
      </div>

      {/* ========================================================================= */}
      {/* ACTIVE CHECKPOINT INSPECTOR CARD                                          */}
      {/* ========================================================================= */}
      <AnimatePresence mode="wait">
        <motion.div
          key={activeStep}
          initial={{ opacity: 0, y: 15 }}
          animate={{ opacity: 1, y: 0 }}
          exit={{ opacity: 0, y: -15 }}
          transition={{ duration: 0.35 }}
          className="bento-card-dark p-6 sm:p-10 border border-emerald-500/20 dark:border-white/20 rounded-2xl sm:rounded-3xl shadow-[0_12px_35px_rgba(16,185,129,0.06)] dark:shadow-none"
        >
          <div className="grid grid-cols-1 lg:grid-cols-12 gap-8 items-start">
            
            {/* Left Column: Details */}
            <div className="lg:col-span-7">
              <div className="flex flex-wrap items-center gap-3 mb-4">
                <span className={`px-3 py-1 rounded-full text-xs font-mono font-semibold ${steps[activeStep].tagClass}`}>
                  {steps[activeStep].tag}
                </span>
                <span className="text-xs font-mono text-slate-500 dark:text-slate-400">
                  Шаг {steps[activeStep].index} из 05
                </span>
              </div>

              <h3 className="font-editorial text-2xl sm:text-4xl font-bold text-slate-900 dark:text-white mb-2">
                {steps[activeStep].title}
              </h3>
              <p className="text-sm sm:text-base font-medium text-slate-700 dark:text-slate-300 mb-6">
                {steps[activeStep].subtitle}
              </p>

              <p className="font-doc text-sm text-slate-600 dark:text-slate-300 leading-relaxed mb-8">
                {steps[activeStep].shortDesc}
              </p>

              {/* Deliverables Checklist */}
              <div className="space-y-3 font-doc">
                <div className="text-xs font-mono uppercase tracking-wider text-slate-500 dark:text-slate-400 font-bold mb-2">
                  Что входит в этот этап:
                </div>
                {steps[activeStep].deliverables.map((item, dIdx) => (
                  <div key={dIdx} className="flex items-start gap-3 text-xs sm:text-sm text-slate-800 dark:text-slate-200">
                    <CheckCircle2 className="w-4 h-4 text-emerald-600 dark:text-emerald-400 mt-0.5 shrink-0" />
                    <span>{item}</span>
                  </div>
                ))}
              </div>
            </div>

            {/* Right Column: Key Takeaway Bento Micro-Module */}
            <div className="lg:col-span-5 flex flex-col gap-4">
              <div className="bento-card-glass p-6 sm:p-7 rounded-2xl border border-emerald-500/20 dark:border-white/20">
                <div className="flex items-center gap-2 text-xs font-mono text-emerald-800 dark:text-slate-200 mb-3 font-semibold">
                  <Check className="w-4 h-4 text-emerald-600 dark:text-emerald-400" />
                  <span>Твой результат после спринта:</span>
                </div>
                <p className="font-doc text-sm sm:text-base text-emerald-950 dark:text-white font-medium leading-relaxed">
                  {steps[activeStep].result}
                </p>
              </div>

              {/* Visual Outcome for the active step */}
              <div className="relative rounded-2xl overflow-hidden border border-emerald-500/20 dark:border-white/20 aspect-[4/3] shadow-lg group bg-slate-100 dark:bg-[#0e1015]">
                <BlurImage
                  src={steps[activeStep].image}
                  alt={steps[activeStep].title}
                  fill
                  loading="lazy"
                  quality={90}
                  className="object-cover object-center group-hover:scale-[1.02] transition-transform duration-500"
                  sizes="(max-width: 1024px) 100vw, 450px"
                />
              </div>

              {/* Navigation buttons to move to next/prev checkpoint */}
              <div className="flex items-center justify-between gap-4 pt-2">
                <button
                  disabled={activeStep === 0}
                  onClick={() => setActiveStep((prev) => Math.max(0, prev - 1))}
                  className="px-5 py-2.5 rounded-full bento-pill text-xs font-mono text-slate-700 dark:text-slate-300 disabled:opacity-30 disabled:pointer-events-none hover:text-emerald-700 dark:hover:text-white"
                >
                  ← Предыдущий шаг
                </button>

                <button
                  disabled={activeStep === steps.length - 1}
                  onClick={() => setActiveStep((prev) => Math.min(steps.length - 1, prev + 1))}
                  className="btn-primary-mono px-5 py-2.5 text-xs font-mono flex items-center gap-1.5 disabled:opacity-30 disabled:pointer-events-none"
                >
                  <span>Следующий шаг</span>
                  <ChevronRight className="w-3.5 h-3.5" />
                </button>
              </div>
            </div>

          </div>
        </motion.div>
      </AnimatePresence>
    </section>
  );
}
