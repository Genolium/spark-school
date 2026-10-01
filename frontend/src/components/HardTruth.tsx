"use client";

import React from "react";
import { BlurImage } from "./BlurImage";
import { motion } from "framer-motion";
import { ArrowRight, AlertCircle, CheckCircle2 } from "lucide-react";

export function HardTruth() {
  const auditMatrix = [
    {
      index: "01",
      deliverable: "Эссе (500 слов)",
      stage: "1-й тур • Смысловой фильтр",
      metric: "85% отсева",
      image: "/criteria-essay.jpg",
      rejectionReason: "Шаблонный текст: «хочу улучшить английский и увидеть Нью-Йорк». Отсутствие личной миссии, конкретных результатов и связи с программой.",
      acceleratorStandard: "Правильная структура, распаковка реального бэкграунда, удержание внимания комиссии с первой строки и соответствие ключевым ценностям программы",
    },
    {
      index: "02",
      deliverable: "Видеовизитка (2 мин)",
      stage: "1-й тур • Подача и энергия",
      metric: "Первые 5 секунд",
      image: "/criteria-video.jpg",
      rejectionReason: "Монотонное чтение с экрана на фоне белой стены. Отсутствие зрительного контакта, перегруз сухими фактами из резюме.",
      acceleratorStandard: "Динамичная композиция на смартфон, грамотный сторителлинг, естественная речь и строгие тайминги.",
    },
    {
      index: "03",
      deliverable: "Резюме (Гарвардский формат)",
      stage: "1-й тур • Профиль кандидата",
      metric: "10 сек на скрининг",
      image: "/criteria-resume.jpg",
      rejectionReason: "Хаос в стуктуре резюме, сухое перечисление проектов без обязанностей и измеримых достижений. CV на 50 страниц из хобби, любимых цитат и политических взглядов.",
      acceleratorStandard: "Одностраничный международный формат резюме по стандартам Гарварда. Только важная информация и прикладные результаты.",
    },
    {
      index: "04",
      deliverable: "Zoom-интервью (45 мин)",
      stage: "2-й тур • Финальный отбор",
      metric: "Решает судьбу $20,000",
      image: "/criteria-zoom.jpg",
      rejectionReason: "Ступор при каверзных вопросах, заученные ответы и потеря нити диалога под стрессом.",
      acceleratorStandard: "Техники для ответов на любые поведенческие кейсы. Отработка 30+ стресс-вопросов на 45-минутной персональной симуляции 1-на-1.",
    },
  ];

  return (
    <section id="hard-truth" className="relative w-full max-w-[1520px] mx-auto px-3 sm:px-6 py-16 sm:py-24 scroll-mt-20 sm:scroll-mt-24">
      {/* Section Header */}
      <div className="max-w-3xl mb-12 sm:mb-16">
        <h2 className="font-editorial text-3xl sm:text-5xl lg:text-6xl font-bold tracking-tight text-slate-900 dark:text-white leading-[1.1]">
          Критерии отбора SPARK:{" "}
          <span className="text-emerald-600 dark:text-emerald-400 font-serif italic">почему 95% заявок отсеивают в первом туре</span>
        </h2>
        <p className="mt-4 text-base text-slate-600 dark:text-slate-300 leading-relaxed font-normal">
          Американская приёмная комиссия оценивает не идеальную академическую грамматику, а лидерский потенциал, культурный контекст и ценность твоего опыта. Разница между финалистом и отсеянным кандидатом видна в деталях каждого документа.
        </p>
      </div>

      {/* Structured Admissions Audit Matrix Grid */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        {auditMatrix.map((item, idx) => (
          <motion.div
            key={idx}
            initial={{ opacity: 0, y: 20 }}
            whileInView={{ opacity: 1, y: 0 }}
            viewport={{ once: true }}
            transition={{ duration: 0.45, delay: idx * 0.08 }}
            className="bento-card-dark p-6 sm:p-7 flex flex-col justify-between rounded-2xl group border border-emerald-500/15 dark:border-white/15 shadow-[0_10px_30px_rgba(16,185,129,0.06)] dark:shadow-none"
          >
            <div>
              {/* Header: Deliverable & Stage */}
              <div className="flex items-center justify-between pb-4 border-b border-emerald-500/15 dark:border-white/10 mb-4">
                <div>
                  <div className="text-[11px] font-mono text-emerald-800 dark:text-slate-400 uppercase tracking-wider mb-1">
                    {item.stage}
                  </div>
                  <h3 className="font-editorial text-xl sm:text-2xl font-bold text-slate-900 dark:text-white">
                    {item.deliverable}
                  </h3>
                </div>
                <div className="text-right">
                  <span className="px-2.5 py-1 rounded-md bg-emerald-500/10 dark:bg-white/[0.06] border border-emerald-500/20 dark:border-white/10 text-emerald-800 dark:text-slate-200 text-xs font-mono font-medium block">
                    {item.metric}
                  </span>
                </div>
              </div>

              {/* Visual Illustration Preview */}
              <div className="relative w-full aspect-[4/3] rounded-xl overflow-hidden border border-emerald-500/20 dark:border-white/15 mb-5 shadow-sm bg-slate-100 dark:bg-[#0e1015]">
                <BlurImage
                  src={item.image}
                  alt={item.deliverable}
                  fill
                  loading="lazy"
                  quality={90}
                  className="object-cover object-center group-hover:scale-[1.02] transition-transform duration-500"
                  sizes="(max-width: 768px) 100vw, 600px"
                />
              </div>

              {/* Comparison Matrix: Rejection vs проект «так называемый SPARK» */}
              <div className="space-y-3 font-doc">
                {/* 95% Rejection Reason */}
                <div className="p-3.5 rounded-xl bg-rose-500/[0.06] border border-rose-500/20">
                  <div className="text-[11px] font-mono text-rose-700 dark:text-rose-300 font-semibold uppercase tracking-wider mb-1 flex items-center gap-1.5">
                    <AlertCircle className="w-3.5 h-3.5 text-rose-500 dark:text-rose-400 shrink-0" />
                    <span>Типичная заявка (95% отсева)</span>
                  </div>
                  <p className="text-xs sm:text-sm text-slate-700 dark:text-slate-300 leading-relaxed">
                    {item.rejectionReason}
                  </p>
                </div>

                {/* Стандарт проекта «так называемый SPARK» */}
                <div className="p-3.5 rounded-xl bg-emerald-500/[0.08] border border-emerald-500/25">
                  <div className="text-[11px] font-mono text-emerald-800 dark:text-emerald-300 font-semibold uppercase tracking-wider mb-1 flex items-center gap-1.5">
                    <CheckCircle2 className="w-3.5 h-3.5 text-emerald-600 dark:text-emerald-400 shrink-0" />
                    <span>Стандарт проекта «так называемый SPARK»</span>
                  </div>
                  <p className="text-xs sm:text-sm text-slate-900 dark:text-slate-200 leading-relaxed font-medium">
                    {item.acceleratorStandard}
                  </p>
                </div>
              </div>
            </div>

            {/* Bottom Footer Indicator */}
            <div className="pt-3.5 mt-5 border-t border-emerald-500/15 dark:border-white/10 flex items-center justify-between text-xs font-mono text-slate-500 dark:text-slate-400">
              <span className="text-slate-700 dark:text-slate-300">Раздел 0{idx + 1} программы</span>
              <span className="text-slate-700 dark:text-slate-300 flex items-center gap-1">
                <span>Шаблоны + личный аудит</span>
                <ArrowRight className="w-3 h-3 text-slate-400" />
              </span>
            </div>
          </motion.div>
        ))}
      </div>
    </section>
  );
}
