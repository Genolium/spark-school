"use client";

import React, { useState } from "react";
import {
  BookOpen,
  CheckCircle2,
  AlertTriangle,
  Sparkles,
  ArrowRight,
  ShieldAlert,
  GraduationCap,
  FileText,
  Video,
  UserCheck,
  ChevronRight,
  Check,
} from "lucide-react";
import { trackFunnelStep } from "@/lib/analytics";

export function DemoLessonSection({ onOpenDiagnostic, onSelectPricing }: { onOpenDiagnostic?: () => void; onSelectPricing?: () => void }) {
  const [activeTab, setActiveTab] = useState<"lesson" | "feedback" | "outcomes">("lesson");

  const handleTabChange = (tab: "lesson" | "feedback" | "outcomes") => {
    setActiveTab(tab);
    trackFunnelStep("demo_lesson_view", { tab });
  };

  return (
    <section id="demo-lesson" className="w-full max-w-[1520px] mx-auto px-4 sm:px-6 py-16 scroll-mt-20">
      <div className="bento-card-dark p-6 sm:p-10 lg:p-12 rounded-[32px] border border-slate-200 dark:border-white/15 bg-white/95 dark:bg-[#12161f]/95 shadow-xl relative overflow-hidden backdrop-blur-xl">
        
        {/* Glow accent */}
        <div className="absolute top-0 right-1/4 w-96 h-48 bg-emerald-500/10 rounded-full blur-3xl pointer-events-none" />

        {/* Section Header */}
        <div className="max-w-3xl mb-8">
          <div className="inline-flex items-center gap-2 px-3.5 py-1.5 rounded-full bento-pill text-xs font-mono text-emerald-800 dark:text-emerald-300 mb-3">
            <BookOpen className="w-3.5 h-3.5 text-emerald-600 dark:text-emerald-400" />
            <span>Демонстрация программы и методики</span>
          </div>
          <h2 className="font-editorial text-2xl sm:text-4xl font-bold tracking-tight text-slate-900 dark:text-white">
            Что внутри: открытый урок и пример рецензии
          </h2>
          <p className="mt-2 text-xs sm:text-sm text-slate-600 dark:text-slate-300 font-doc leading-relaxed">
            Посмотрите реальный фрагмент вводного курса, формат работы куратора над заявкой и конкретный набор готовых материалов, который получает выпускник.
          </p>
        </div>

        {/* Tabs switcher */}
        <div className="flex flex-wrap items-center gap-2 mb-8 border-b border-slate-200 dark:border-white/10 pb-4">
          <button
            type="button"
            onClick={() => handleTabChange("lesson")}
            className={`px-4 py-2 rounded-xl text-xs sm:text-sm font-mono font-bold transition-all flex items-center gap-2 ${
              activeTab === "lesson"
                ? "bg-emerald-500 text-black shadow-md"
                : "bg-slate-100 hover:bg-slate-200 dark:bg-white/5 dark:hover:bg-white/10 text-slate-700 dark:text-slate-300"
            }`}
          >
            <BookOpen className="w-4 h-4" />
            <span>1. Вводный урок курса</span>
          </button>

          <button
            type="button"
            onClick={() => handleTabChange("feedback")}
            className={`px-4 py-2 rounded-xl text-xs sm:text-sm font-mono font-bold transition-all flex items-center gap-2 ${
              activeTab === "feedback"
                ? "bg-emerald-500 text-black shadow-md"
                : "bg-slate-100 hover:bg-slate-200 dark:bg-white/5 dark:hover:bg-white/10 text-slate-700 dark:text-slate-300"
            }`}
          >
            <Sparkles className="w-4 h-4" />
            <span>2. Разбор заявки куратором</span>
          </button>

          <button
            type="button"
            onClick={() => handleTabChange("outcomes")}
            className={`px-4 py-2 rounded-xl text-xs sm:text-sm font-mono font-bold transition-all flex items-center gap-2 ${
              activeTab === "outcomes"
                ? "bg-emerald-500 text-black shadow-md"
                : "bg-slate-100 hover:bg-slate-200 dark:bg-white/5 dark:hover:bg-white/10 text-slate-700 dark:text-slate-300"
            }`}
          >
            <GraduationCap className="w-4 h-4" />
            <span>3. Результат подготовки</span>
          </button>
        </div>

        {/* Tab 1: Intro Lesson */}
        {activeTab === "lesson" && (
          <div className="space-y-6">
            <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
              <div className="p-4 rounded-2xl bg-emerald-500/10 border border-emerald-500/25">
                <div className="text-xs font-mono font-bold text-emerald-800 dark:text-emerald-300 uppercase tracking-wider mb-1">
                  Грант $20,000
                </div>
                <div className="text-xs text-slate-600 dark:text-slate-300">
                  Покрывает перелеты в США и Астану, проживание в общежитии, meal plan, обучение в вузе и карту на личные расходы ($50/нед).
                </div>
              </div>

              <div className="p-4 rounded-2xl bg-teal-500/10 border border-teal-500/25">
                <div className="text-xs font-mono font-bold text-teal-800 dark:text-teal-300 uppercase tracking-wider mb-1">
                  Виза J-1 и Section 212(e)
                </div>
                <div className="text-xs text-slate-600 dark:text-slate-300">
                  Неиммиграционный статус Student Non-degree. Требует возвращения в родную страну на 2 года перед подачей на рабочие визы H/L.
                </div>
              </div>

              <div className="p-4 rounded-2xl bg-amber-500/10 border border-amber-500/25">
                <div className="text-xs font-mono font-bold text-amber-800 dark:text-amber-300 uppercase tracking-wider mb-1">
                  2 этапа отбора
                </div>
                <div className="text-xs text-slate-600 dark:text-slate-300">
                  1-й тур: онлайн-проверка в Submittable (резюме + эссе 500 слов + видео). 2-й тур: Zoom-интервью 20 минут + экзамен DET.
                </div>
              </div>
            </div>

            <div className="p-5 sm:p-6 rounded-2xl bg-slate-50 dark:bg-white/5 border border-slate-200 dark:border-white/10 space-y-4">
              <h3 className="text-base sm:text-lg font-bold text-slate-900 dark:text-white flex items-center gap-2">
                <ShieldAlert className="w-5 h-5 text-amber-500" />
                <span>3 фильтра отсева, на которых срезаются 90% заявок:</span>
              </h3>

              <div className="grid grid-cols-1 sm:grid-cols-3 gap-3 text-xs">
                <div className="p-3 rounded-xl bg-white dark:bg-black/30 border border-slate-200 dark:border-white/10">
                  <div className="font-bold text-rose-600 dark:text-rose-400 mb-1">1. Ловушка туриста</div>
                  <p className="text-slate-600 dark:text-slate-300">
                    Фразы <i>«Хочу подтянуть английский и увидеть Нью-Йорк»</i> бракуются сразу. Комиссия отбирает за civic engagement — пользу для своего окружения.
                  </p>
                </div>

                <div className="p-3 rounded-xl bg-white dark:bg-black/30 border border-slate-200 dark:border-white/10">
                  <div className="font-bold text-rose-600 dark:text-rose-400 mb-1">2. Стерильные тексты от ИИ</div>
                  <p className="text-slate-600 dark:text-slate-300">
                    Генерации без шероховатостей живой речи и цифр видны опытному ридеру за 30 секунд. Заявке нужен личный сюжет и фактура.
                  </p>
                </div>

                <div className="p-3 rounded-xl bg-white dark:bg-black/30 border border-slate-200 dark:border-white/10">
                  <div className="font-bold text-rose-600 dark:text-rose-400 mb-1">3. TAL-триггеры</div>
                  <p className="text-slate-600 dark:text-slate-300">
                    Technology Alert List: кибербезопасность, шифрование и критические технологии блокируются Госдепом. Фокусируемся на гражданских проектах.
                  </p>
                </div>
              </div>
            </div>
          </div>
        )}

        {/* Tab 2: Feedback Preview */}
        {activeTab === "feedback" && (
          <div className="space-y-4">
            <div className="text-xs font-mono font-semibold uppercase text-slate-500 dark:text-slate-400">
              Пример рецензирования черновика эссе куратором программы:
            </div>

            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              {/* Weak Draft */}
              <div className="p-5 rounded-2xl bg-rose-500/10 border border-rose-500/25 space-y-2">
                <div className="flex items-center gap-2 text-xs font-mono font-bold text-rose-700 dark:text-rose-400">
                  <AlertTriangle className="w-4 h-4" />
                  <span>Типичный черновик кандидата:</span>
                </div>
                <blockquote className="text-xs text-slate-700 dark:text-slate-300 italic border-l-2 border-rose-500/40 pl-3 py-1">
                  «Ever since high school, I have been passionate about technologies. I dream of visiting American universities, making international friends, and improving my English skills in real communication...»
                </blockquote>
                <div className="text-[11px] font-mono text-rose-800 dark:text-rose-300 bg-rose-500/10 p-2.5 rounded-xl border border-rose-500/20">
                  <b>Замечание куратора:</b> Срабатывает фильтр «туриста». Отсутствует нарративная ось (narrative spine), нет пользы для общества и конкретных измеримых фактов.
                </div>
              </div>

              {/* Polished Draft */}
              <div className="p-5 rounded-2xl bg-emerald-500/10 border border-emerald-500/25 space-y-2">
                <div className="flex items-center gap-2 text-xs font-mono font-bold text-emerald-700 dark:text-emerald-400">
                  <CheckCircle2 className="w-4 h-4" />
                  <span>Докрученный вариант после работы с куратором:</span>
                </div>
                <blockquote className="text-xs text-slate-700 dark:text-slate-300 italic border-l-2 border-emerald-500/40 pl-3 py-1">
                  «When our university department suffered from disorganized class cancellations, I coded a free notification bot now relied upon by 700+ engineering peers every morning...»
                </blockquote>
                <div className="text-[11px] font-mono text-emerald-800 dark:text-emerald-300 bg-emerald-500/10 p-2.5 rounded-xl border border-emerald-500/20">
                  <b>Результат ревизии:</b> Сформулирован Action Hook в первых 5 секундах, показан лидерский вклад и четкое позиционирование для комиссии.
                </div>
              </div>
            </div>
          </div>
        )}

        {/* Tab 3: Tangible Outcomes */}
        {activeTab === "outcomes" && (
          <div className="space-y-4">
            <div className="text-xs font-mono font-semibold uppercase text-slate-500 dark:text-slate-400">
              Комплект материалов, который студент получает к дедлайну:
            </div>

            <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-3.5">
              <div className="p-4 rounded-2xl bg-slate-50 dark:bg-white/5 border border-slate-200 dark:border-white/10">
                <FileText className="w-6 h-6 text-emerald-500 mb-2" />
                <div className="text-xs font-bold text-slate-900 dark:text-white">1-страничное US Resume</div>
                <p className="text-[11px] text-slate-600 dark:text-slate-400 mt-1">
                  По стандартам американских университетов с глаголами действия (Action Verbs) и метриками.
                </p>
              </div>

              <div className="p-4 rounded-2xl bg-slate-50 dark:bg-white/5 border border-slate-200 dark:border-white/10">
                <BookOpen className="w-6 h-6 text-teal-500 mb-2" />
                <div className="text-xs font-bold text-slate-900 dark:text-white">Эссе на 500 слов</div>
                <p className="text-[11px] text-slate-600 dark:text-slate-400 mt-1">
                  Глубокий ответ на официальный вопрос года с личным хуком и связкой с возвращением домой.
                </p>
              </div>

              <div className="p-4 rounded-2xl bg-slate-50 dark:bg-white/5 border border-slate-200 dark:border-white/10">
                <Video className="w-6 h-6 text-indigo-500 mb-2" />
                <div className="text-xs font-bold text-slate-900 dark:text-white">Видеопитч 80–100 сек</div>
                <p className="text-[11px] text-slate-600 dark:text-slate-400 mt-1">
                  Сценарий, драматургия первых секунд и уверенная подача без чтения с листа.
                </p>
              </div>

              <div className="p-4 rounded-2xl bg-slate-50 dark:bg-white/5 border border-slate-200 dark:border-white/10">
                <UserCheck className="w-6 h-6 text-amber-500 mb-2" />
                <div className="text-xs font-bold text-slate-900 dark:text-white">Симуляция Zoom 1-на-1</div>
                <p className="text-[11px] text-slate-600 dark:text-slate-400 mt-1">
                  45–60 минутный созвон с имитацией 20-минутного интервью с американской комиссией и разбором.
                </p>
              </div>
            </div>
          </div>
        )}

        {/* CTA Footer */}
        <div className="mt-8 pt-6 border-t border-slate-200 dark:border-white/10 flex flex-col sm:flex-row items-center justify-between gap-4">
          <div className="text-xs text-slate-500 dark:text-slate-400 font-mono">
            Курс и сопровождение ведет финалист программы SPARK 2026 (так называемый Иль).
          </div>
          <div className="flex items-center gap-3 w-full sm:w-auto">
            {onOpenDiagnostic && (
              <button
                type="button"
                onClick={() => {
                  trackFunnelStep("diagnostic_start");
                  onOpenDiagnostic();
                }}
                className="flex-1 sm:flex-none px-4 py-2.5 rounded-xl bg-slate-100 hover:bg-slate-200 dark:bg-white/10 dark:hover:bg-white/20 text-slate-900 dark:text-white text-xs font-mono font-semibold transition-all cursor-pointer text-center"
              >
                Пройти диагностику ➔
              </button>
            )}
            <a
              href="#pricing"
              onClick={() => {
                trackFunnelStep("pricing_select_tier");
                if (onSelectPricing) onSelectPricing();
              }}
              className="flex-1 sm:flex-none px-5 py-2.5 rounded-xl bg-emerald-500 hover:bg-emerald-400 text-black text-xs font-mono font-bold transition-all shadow-md active:scale-95 text-center cursor-pointer"
            >
              Выбрать тариф
            </a>
          </div>
        </div>

      </div>
    </section>
  );
}
