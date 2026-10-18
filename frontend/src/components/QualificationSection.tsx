"use client";

import React from "react";
import { motion } from "framer-motion";
import { XCircle, CheckCircle2, ArrowUpRight, Flame, ShieldAlert } from "lucide-react";

interface QualificationSectionProps {
  onSelectTier?: () => void;
}

export function QualificationSection({ onSelectTier }: QualificationSectionProps) {
  const disqualifiers = [
    {
      title: "Ждёте, что эссе напишут за вас",
      desc: "Если вы рассчитываете на генерацию через ChatGPT или думаете, что куратор сделает работу за вас — вы гарантированно срежетесь на интервью. Мы учим распаковывать ваш реальный опыт, но писать и думать предстоит вам.",
    },
    {
      title: "Не готовы переписывать черновик 3–4 раза",
      desc: "Если честный разбор слабых мест и критика вызывают обиду, нам не по пути. Победная заявка рождается в процессе безжалостных итераций и вычитки каждого предложения.",
    },
    {
      title: "Планируете сесть за заявку за 2 часа до дедлайна",
      desc: "Отбор в SPARK не терпит спешки. Подача «на удачу» в последний вечер — это гарантированный отсев на первом же этапе скрининга.",
    },
    {
      title: "Боитесь говорить о реальных трудностях и факапах",
      desc: "Приёмная комиссия мгновенно чувствует фальшь и шаблонные лозунги «успешного успеха». Здесь ценят зрелость, искренность и выводы, сделанные из собственных ошибок.",
    },
  ];

  const qualifiers = [
    {
      title: "Вы студент 18–21 года с большими амбициями",
      desc: "Вы понимаете, что лето в США на кампусе американского университета и грант $20,000 — это редчайший социальный лифт, способный развернуть всю вашу дальнейшую карьеру на 180°.",
    },
    {
      title: "Вы готовы пахать и докручивать материалы",
      desc: "Вам нужна не пустая похвала, а жесткий построчный аудит эссе, резюме и видеовизитки от финалиста прошлого отбора, знающего все критерии американской комиссии.",
    },
    {
      title: "Вы хотите отрепетировать настоящее 20-минутное интервью",
      desc: "Отработать 30+ стресс-вопросов на английском языке на 45–60 минутном персональном созвоне 1-на-1 с симуляцией и подробной обратной связью, чтобы на реальном собеседовании чувствовать абсолютную уверенность.",
    },
    {
      title: "Вы нацелены побеждать, а не «просто поучаствовать»",
      desc: "Вы готовы вложить силы и время, чтобы ваша заявка оказалась на голову сильнее сотен других претендентов со всего СНГ.",
    },
  ];

  const handleScrollToPricing = () => {
    if (onSelectTier) {
      onSelectTier();
      return;
    }
    if (typeof window !== "undefined") {
      const el = document.getElementById("pricing");
      if (el) {
        el.scrollIntoView({ behavior: "smooth" });
      }
    }
  };

  return (
    <section id="qualification" className="relative w-full max-w-[1520px] mx-auto px-3 sm:px-6 py-16 sm:py-24 scroll-mt-20 sm:scroll-mt-24">
      {/* Section Header */}
      <div className="max-w-3xl mx-auto text-center mb-12 sm:mb-16">
        <div className="inline-flex items-center gap-2 px-3.5 py-1.5 rounded-full bento-pill text-xs font-mono text-rose-700 dark:text-rose-300 mb-4 border border-rose-500/25">
          <ShieldAlert className="w-3.5 h-3.5 text-rose-500" />
          <span>Честный фильтр отбора</span>
        </div>
        <h2 className="font-editorial text-3xl sm:text-5xl lg:text-6xl font-bold tracking-tight text-slate-900 dark:text-white leading-[1.1]">
          Кому этот курс{" "}
          <span className="text-rose-600 dark:text-rose-400 font-serif italic">точно НЕ подойдет</span>
        </h2>
        <p className="mt-4 text-base text-slate-600 dark:text-slate-300 leading-relaxed font-normal">
          Мы ценим ваше время и свою репутацию. Пожалуйста, внимательно прочитайте этот блок перед тем, как оплачивать доступ к материалам.
        </p>
      </div>

      {/* 2-Column Polar Filter Grid */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6 lg:gap-8 max-w-6xl mx-auto">
        
        {/* Left Column: Disqualifiers (Red / Warning) */}
        <motion.div
          initial={{ opacity: 0, x: -20 }}
          whileInView={{ opacity: 1, x: 0 }}
          viewport={{ once: true }}
          transition={{ duration: 0.5 }}
          className="bento-card-dark p-6 sm:p-8 rounded-3xl border border-rose-500/30 dark:border-rose-500/20 bg-rose-500/[0.02] dark:bg-rose-500/[0.04] flex flex-col justify-between"
        >
          <div>
            <div className="flex items-center gap-3 mb-6 pb-4 border-b border-rose-500/20">
              <div className="w-10 h-10 rounded-2xl bg-rose-500/10 text-rose-600 dark:text-rose-400 flex items-center justify-center shrink-0">
                <XCircle className="w-5 h-5 stroke-[2.5]" />
              </div>
              <div>
                <span className="text-[11px] font-mono text-rose-600 dark:text-rose-400 uppercase tracking-wider block font-semibold">
                  Стоп-факторы
                </span>
                <h3 className="font-editorial text-xl sm:text-2xl font-bold text-slate-900 dark:text-white">
                  Не приходите на курс, если:
                </h3>
              </div>
            </div>

            <div className="space-y-5">
              {disqualifiers.map((item, idx) => (
                <div key={idx} className="flex items-start gap-3.5">
                  <span className="w-6 h-6 rounded-full bg-rose-500/10 text-rose-600 dark:text-rose-400 flex items-center justify-center font-mono text-xs font-bold shrink-0 mt-0.5">
                    {idx + 1}
                  </span>
                  <div>
                    <h4 className="font-bold text-sm sm:text-base text-slate-900 dark:text-white mb-1">
                      {item.title}
                    </h4>
                    <p className="font-doc text-xs sm:text-sm text-slate-600 dark:text-slate-300 leading-relaxed">
                      {item.desc}
                    </p>
                  </div>
                </div>
              ))}
            </div>
          </div>

          <div className="mt-8 pt-5 border-t border-rose-500/20 text-xs font-mono text-rose-700 dark:text-rose-300">
            ⚠ Мы возвращаем 100% средств, если видим, что студент не готов включиться в реальную работу.
          </div>
        </motion.div>

        {/* Right Column: Qualifiers (Emerald / Success) */}
        <motion.div
          initial={{ opacity: 0, x: 20 }}
          whileInView={{ opacity: 1, x: 0 }}
          viewport={{ once: true }}
          transition={{ duration: 0.5 }}
          className="bento-card-dark p-6 sm:p-8 rounded-3xl border border-emerald-500/40 dark:border-emerald-500/30 bg-emerald-500/[0.03] dark:bg-emerald-500/[0.05] flex flex-col justify-between shadow-[0_15px_40px_rgba(16,185,129,0.08)]"
        >
          <div>
            <div className="flex items-center gap-3 mb-6 pb-4 border-b border-emerald-500/20">
              <div className="w-10 h-10 rounded-2xl bg-emerald-500/15 text-emerald-600 dark:text-emerald-400 flex items-center justify-center shrink-0">
                <CheckCircle2 className="w-5 h-5 stroke-[2.5]" />
              </div>
              <div>
                <span className="text-[11px] font-mono text-emerald-700 dark:text-emerald-400 uppercase tracking-wider block font-semibold">
                  Идеальный профиль
                </span>
                <h3 className="font-editorial text-xl sm:text-2xl font-bold text-slate-900 dark:text-white">
                  Мы сработаемся на 100%, если:
                </h3>
              </div>
            </div>

            <div className="space-y-5">
              {qualifiers.map((item, idx) => (
                <div key={idx} className="flex items-start gap-3.5">
                  <span className="w-6 h-6 rounded-full bg-emerald-500/15 text-emerald-700 dark:text-emerald-400 flex items-center justify-center font-mono text-xs font-bold shrink-0 mt-0.5">
                    {idx + 1}
                  </span>
                  <div>
                    <h4 className="font-bold text-sm sm:text-base text-slate-900 dark:text-white mb-1">
                      {item.title}
                    </h4>
                    <p className="font-doc text-xs sm:text-sm text-slate-600 dark:text-slate-300 leading-relaxed">
                      {item.desc}
                    </p>
                  </div>
                </div>
              ))}
            </div>
          </div>

          <div className="mt-8 pt-5 border-t border-emerald-500/20">
            <button
              type="button"
              onClick={handleScrollToPricing}
              className="btn-primary-mono w-full py-4 text-xs sm:text-sm font-bold flex items-center justify-center gap-2 group shadow-lg active:scale-95"
            >
              <Flame className="w-4 h-4 fill-current text-amber-300" />
              <span>Я готов к серьезной работе — Выбрать тариф</span>
              <ArrowUpRight className="w-4 h-4" />
            </button>
          </div>
        </motion.div>

      </div>
    </section>
  );
}
