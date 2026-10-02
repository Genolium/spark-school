"use client";

import React, { useState } from "react";
import { motion, AnimatePresence } from "framer-motion";
import { ChevronDown, HelpCircle } from "lucide-react";

export function FaqSection() {
  const [openIndex, setOpenIndex] = useState<number | null>(0);

  const faqs = [
    {
      q: "Какой у меня должен быть уровень английского для старта?",
      a: "Достаточно уверенного базового понимания (B1–B2). Первый тур отбора оценивает не академическую грамматику, а смыслы, твои проекты и мотивацию. К этапу Duolingo DET ты подготовишься по нашим закрытым алгоритмам сдачи дома.",
    },
    {
      q: "А если я не выиграю грант SPARK?",
      a: "Ты гарантированно получишь упакованное резюме американского формата по стандартам Гарварда, сильное персональное эссе и боевой навык прохождения интервью на английском. Эти активы откроют тебе двери в международные IT-стажировки, европейские гранты (Erasmus+) и грантовые программы следующего сезона.",
    },
    {
      q: "Каковы условия «Обратной гарантии риска» (100% возврат или перенос)?",
      a: "Если ты полностью проходишь все учебные модули акселератора, сдаешь домашние задания на проверку куратору, но приёмная комиссия SPARK не принимает твою заявку к рассмотрению — мы возвращаем 100% стоимости тарифа или бесплатно переносим твоё участие на следующий отборочный поток. Мы уверены в методологии и берем риск на себя.",
    },
    {
      q: "Вы напишете заявку и эссе за меня?",
      a: "Категорически нет. Американская приёмная комиссия отсматривает тысячи анкет и мгновенно отсекает чужие тексты и сгенерированный ИИ-копипаст. Мы даём проверенную структуру, препарируем твои черновики, указываем на логические дыры и помогаем раскрыть твой уникальный голос.",
    },
    {
      q: "Как быстро я получу доступ к каналу и материалам после оплаты?",
      a: "Мгновенно в автоматическом режиме. Сразу после подтверждения оплаты вы получите персональную ссылку-приглашение в закрытый Telegram-канал с материалами и закрытый чат участников.",
    },
    {
      q: "Безопасна ли оплата и формируется ли официальный чек?",
      a: "Абсолютно. Все платежи проходят через защищённый эквайринг (СБП, банковские карты РФ). Оплата официально фискализируется через Федеральную налоговую службу РФ (Самозанятый Иль О. В., ИНН), чек отправляется на указанный контакт.",
    },
    {
      q: "Сколько времени в день требуется на подготовку?",
      a: "Материалы разбиты на короткие, прикладные пошаговые руководства и видеоразборы. Достаточно уделять 40–60 минут в день на протяжении 2–3 недель. Если до дедлайна осталось мало времени, всю программу можно пройти в ускоренном режиме за 5–7 дней.",
    },
  ];

  return (
    <section id="faq" className="relative w-full max-w-[1520px] mx-auto px-3 sm:px-6 py-16 sm:py-24 scroll-mt-20 sm:scroll-mt-24">
      {/* Section Header */}
      <div className="max-w-3xl mb-12 sm:mb-16">
        <div className="inline-flex items-center gap-2 px-3.5 py-1.5 rounded-full bento-pill text-xs font-mono text-slate-700 dark:text-emerald-300 mb-4">
          <HelpCircle className="w-3.5 h-3.5 text-emerald-600 dark:text-emerald-400" />
          <span>Частые вопросы</span>
        </div>
        <h2 className="font-editorial text-3xl sm:text-5xl lg:text-6xl font-bold tracking-tight text-slate-900 dark:text-white leading-[1.1]">
          Ответы на ключевые вопросы <br />
          <span className="text-emerald-600 dark:text-emerald-400 font-serif italic">и снятие сомнений</span>
        </h2>
        <p className="mt-4 text-base text-slate-600 dark:text-slate-300 leading-relaxed font-normal">
          Всё, что важно знать о формате обучения, гарантиях, требованиях к языку и процессе сопровождения.
        </p>
      </div>

      {/* Accordion List */}
      <div className="max-w-4xl space-y-4">
        {faqs.map((faq, idx) => {
          const isOpen = openIndex === idx;
          return (
            <div
              key={idx}
              className={`rounded-3xl bento-card-dark transition-all border overflow-hidden ${
                isOpen
                  ? "border-emerald-500/40 bg-emerald-500/[0.04] dark:border-white/30 dark:bg-white/[0.08]"
                  : "border-emerald-500/15 hover:border-emerald-500/30 dark:border-white/10 dark:hover:border-white/20"
              }`}
            >
              <button
                onClick={() => setOpenIndex(isOpen ? null : idx)}
                className="w-full p-6 sm:p-7 text-left flex items-center justify-between gap-4 focus:outline-none"
              >
                <span className="font-editorial text-lg sm:text-xl font-bold text-slate-900 dark:text-white leading-snug">
                  {faq.q}
                </span>
                <div
                  className={`w-8 h-8 rounded-full bg-emerald-500/10 dark:bg-white/10 flex items-center justify-center shrink-0 text-emerald-700 dark:text-white transition-all duration-300 ${
                    isOpen ? "rotate-180 bg-emerald-600 text-white dark:bg-white dark:text-[#121316]" : ""
                  }`}
                >
                  <ChevronDown className="w-4 h-4 stroke-[2.5]" />
                </div>
              </button>

              <AnimatePresence>
                {isOpen && (
                  <motion.div
                    initial={{ height: 0, opacity: 0 }}
                    animate={{ height: "auto", opacity: 1 }}
                    exit={{ height: 0, opacity: 0 }}
                    transition={{ duration: 0.25 }}
                  >
                    <div className="px-6 pb-6 sm:px-7 sm:pb-7 text-sm sm:text-base text-slate-600 dark:text-slate-300 leading-relaxed border-t border-emerald-500/10 dark:border-white/5 pt-4 font-doc">
                      {faq.a}
                    </div>
                  </motion.div>
                )}
              </AnimatePresence>
            </div>
          );
        })}
      </div>
    </section>
  );
}
