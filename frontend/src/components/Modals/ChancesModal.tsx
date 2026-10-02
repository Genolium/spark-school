"use client";

import React, { useState } from "react";
import { motion, AnimatePresence } from "framer-motion";
import { X, Calculator, CheckCircle2, AlertTriangle, ArrowRight, RotateCcw, Send, Sparkles } from "lucide-react";

interface ChancesModalProps {
  isOpen: boolean;
  onClose: () => void;
}

export function ChancesModal({ isOpen, onClose }: ChancesModalProps) {
  const [step, setStep] = useState<1 | 2 | 3 | 4>(1);
  const [ageCitizenshipValid, setAgeCitizenshipValid] = useState<boolean | null>(null);
  const [universityYear, setUniversityYear] = useState<"1-2" | "3" | "senior" | null>(null);
  const [englishLevel, setEnglishLevel] = useState<"b1" | "b2" | "c1" | null>(null);

  const getScoreData = () => {
    switch (englishLevel) {
      case "c1":
        return {
          percent: 96,
          label: "Максимальные шансы (Elite Zone)",
          color: "text-emerald-500",
          strokeColor: "#10b981",
          description: "У тебя сильная языковая база! При грамотной упаковке истории в эссе и подготовке к поведенческому интервью твоя заявка войдёт в топ-5% отбора.",
        };
      case "b2":
        return {
          percent: 88,
          label: "Отличные шансы (High Potential)",
          color: "text-teal-500",
          strokeColor: "#14b8a6",
          description: "Отличный уровень для уверенного прохождения собеседования с американской комиссией. Главный фокус — отточить структуру Google XYZ резюме.",
        };
      case "b1":
      default:
        return {
          percent: 75,
          label: "Хороший потенциал (Needs Practice)",
          color: "text-amber-500",
          strokeColor: "#f59e0b",
          description: "Базового уровня достаточно для старта! Комиссия оценивает идеи и лидерский потенциал. В проекте «так называемый SPARK» поможем подготовить скрипты ответов.",
        };
    }
  };

  const handleReset = () => {
    setStep(1);
    setAgeCitizenshipValid(null);
    setUniversityYear(null);
    setEnglishLevel(null);
  };

  const scoreData = getScoreData();

  if (!isOpen) return null;

  return (
    <AnimatePresence>
      <div className="fixed inset-0 z-50 flex items-center justify-center p-3 sm:p-4 overflow-y-auto">
        {/* Backdrop */}
        <motion.div
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          exit={{ opacity: 0 }}
          onClick={onClose}
          className="fixed inset-0 bg-black/75 backdrop-blur-md"
        />

        {/* Modal Window */}
        <motion.div
          initial={{ opacity: 0, scale: 0.95, y: 15 }}
          animate={{ opacity: 1, scale: 1, y: 0 }}
          exit={{ opacity: 0, scale: 0.95, y: 15 }}
          transition={{ duration: 0.25 }}
          className="relative w-full max-w-2xl p-6 sm:p-8 rounded-3xl border border-emerald-500/25 dark:border-white/20 shadow-2xl z-10 bg-white dark:bg-[#12161f] text-slate-900 dark:text-white my-auto max-h-[92vh] overflow-y-auto"
        >
          {/* Close Button */}
          <button
            type="button"
            onClick={onClose}
            className="sticky top-0 float-right -mt-1 -mr-1 w-8 h-8 rounded-full bg-slate-100 hover:bg-slate-200 dark:bg-white/10 dark:hover:bg-white/20 flex items-center justify-center text-slate-600 dark:text-slate-300 hover:text-slate-950 dark:hover:text-white border border-slate-200 dark:border-white/10 transition-all z-20 shadow-sm cursor-pointer"
            aria-label="Закрыть"
          >
            <X className="w-4 h-4" />
          </button>

          {/* Ambient Top Glow */}
          <div className="absolute -top-20 left-1/2 -translate-x-1/2 w-80 h-40 bg-emerald-500/15 rounded-full blur-3xl pointer-events-none" />

          {/* Header */}
          <div className="text-center max-w-xl mx-auto mb-6 pr-6 sm:pr-0 relative z-10">
            <div className="inline-flex items-center gap-2 px-3.5 py-1.5 rounded-full bento-pill text-xs font-mono text-emerald-800 dark:text-emerald-300 mb-3">
              <Calculator className="w-3.5 h-3.5 text-emerald-600 dark:text-emerald-400" />
              <span>Интерактивный экспресс-тест</span>
            </div>
            <h2 className="font-editorial text-2xl sm:text-3xl font-bold tracking-tight text-slate-900 dark:text-white">
              Оцени свои шансы на грант $20,000
            </h2>
            <p className="mt-1.5 text-xs sm:text-sm text-slate-600 dark:text-slate-300 font-doc">
              3 вопроса • Без лишней бюрократии • Мгновенный расчет вероятности победы
            </p>

            {/* Step Progress Bar */}
            <div className="flex items-center justify-center gap-2 mt-4">
              {[1, 2, 3].map((s) => (
                <div
                  key={s}
                  className={`h-1.5 rounded-full transition-all duration-300 ${
                    step === s
                      ? "w-10 bg-emerald-600 dark:bg-emerald-400"
                      : step > s
                      ? "w-6 bg-emerald-500/40"
                      : "w-6 bg-slate-200 dark:bg-white/10"
                  }`}
                />
              ))}
            </div>
          </div>

          {/* Step 1: Age & Citizenship */}
          <AnimatePresence mode="wait">
            {step === 1 && (
              <motion.div
                key="step-1"
                initial={{ opacity: 0, y: 15 }}
                animate={{ opacity: 1, y: 0 }}
                exit={{ opacity: 0, y: -15 }}
                transition={{ duration: 0.3 }}
                className="max-w-lg mx-auto text-center relative z-10"
              >
                <div className="text-xs font-mono font-semibold uppercase text-emerald-700 dark:text-emerald-400 mb-2">
                  Шаг 1 из 3: Возраст и гражданство
                </div>
                <h3 className="text-base sm:text-xl font-bold text-slate-900 dark:text-white mb-2">
                  Тебе от 18 до 21 года, ты гражданин РФ и проживаешь в России?
                </h3>
                <p className="text-xs text-slate-600 dark:text-slate-300 mb-6">
                  Официальные критерии Госдепартамента США для летней грантовой программы обмена.
                </p>

                <div className="grid grid-cols-1 sm:grid-cols-2 gap-3.5">
                  <button
                    type="button"
                    onClick={() => {
                      setAgeCitizenshipValid(true);
                      setStep(2);
                    }}
                    className="p-4 rounded-2xl border border-emerald-300 hover:border-emerald-500 bg-emerald-50 hover:bg-emerald-100/80 dark:bg-emerald-500/10 dark:hover:bg-emerald-500/20 dark:border-emerald-500/30 text-emerald-950 dark:text-white font-semibold text-sm transition-all flex items-center justify-center gap-2 group active:scale-95 cursor-pointer shadow-sm"
                  >
                    <CheckCircle2 className="w-4 h-4 text-emerald-600 dark:text-emerald-400 group-hover:scale-110 transition-transform" />
                    <span>Да, всё верно</span>
                  </button>

                  <button
                    type="button"
                    onClick={() => {
                      setAgeCitizenshipValid(false);
                    }}
                    className="p-4 rounded-2xl border border-slate-200 hover:border-rose-400 bg-slate-50 hover:bg-slate-100 dark:bg-white/5 dark:border-white/10 text-slate-700 dark:text-slate-300 font-medium text-sm transition-all active:scale-95 cursor-pointer"
                  >
                    <span>Нет, не подхожу</span>
                  </button>
                </div>

                {ageCitizenshipValid === false && (
                  <motion.div
                    initial={{ opacity: 0, height: 0 }}
                    animate={{ opacity: 1, height: "auto" }}
                    className="mt-6 p-4 rounded-2xl bg-amber-50 border border-amber-200 text-amber-900 dark:bg-amber-500/10 dark:border-amber-500/25 dark:text-slate-200 text-left text-xs font-doc"
                  >
                    <div className="flex items-center gap-2 font-bold text-amber-700 dark:text-amber-400 mb-1">
                      <AlertTriangle className="w-4 h-4" />
                      <span>Ограничение программы</span>
                    </div>
                    По правилам программы участник должен быть гражданином РФ в возрасте от 18 до 21 года на момент стажировки. Но ты можешь разобрать другие возможности или задать вопрос куратору!
                    <div className="mt-3 flex gap-2">
                      <a
                        href="https://t.me/ilyan_vas"
                        target="_blank"
                        rel="noreferrer"
                        className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-xl bg-amber-600 hover:bg-amber-700 text-white font-mono text-[11px] font-bold cursor-pointer"
                      >
                        <Send className="w-3 h-3" /> Написать куратору
                      </a>
                      <button
                        type="button"
                        onClick={handleReset}
                        className="px-3 py-1.5 rounded-xl bg-slate-200 hover:bg-slate-300 dark:bg-white/10 text-[11px] font-mono text-slate-800 dark:text-slate-300 cursor-pointer"
                      >
                        Сбросить
                      </button>
                    </div>
                  </motion.div>
                )}
              </motion.div>
            )}

            {/* Step 2: University Year */}
            {step === 2 && (
              <motion.div
                key="step-2"
                initial={{ opacity: 0, y: 15 }}
                animate={{ opacity: 1, y: 0 }}
                exit={{ opacity: 0, y: -15 }}
                transition={{ duration: 0.3 }}
                className="max-w-lg mx-auto text-center relative z-10"
              >
                <div className="text-xs font-mono font-semibold uppercase text-emerald-700 dark:text-emerald-400 mb-2">
                  Шаг 2 из 3: Курс обучения
                </div>
                <h3 className="text-base sm:text-xl font-bold text-slate-900 dark:text-white mb-2">
                  На каком ты сейчас курсе в вузе?
                </h3>
                <p className="text-xs text-slate-600 dark:text-slate-300 mb-6">
                  Требование визы J-1: обязательство вернуться минимум на 1 академический год в РФ.
                </p>

                <div className="space-y-3">
                  <button
                    type="button"
                    onClick={() => {
                      setUniversityYear("1-2");
                      setStep(3);
                    }}
                    className="w-full p-4 rounded-2xl border border-slate-200 hover:border-emerald-500 bg-slate-50 hover:bg-emerald-50/70 dark:bg-white/5 dark:border-white/10 dark:hover:bg-emerald-500/10 text-slate-900 dark:text-white font-medium text-sm transition-all flex items-center justify-between group active:scale-95 cursor-pointer shadow-xs"
                  >
                    <span className="font-semibold">1–2 курс (бакалавриат / специалитет)</span>
                    <ArrowRight className="w-4 h-4 text-emerald-600 dark:text-emerald-400 group-hover:translate-x-1 transition-transform" />
                  </button>

                  <button
                    type="button"
                    onClick={() => {
                      setUniversityYear("3");
                      setStep(3);
                    }}
                    className="w-full p-4 rounded-2xl border border-slate-200 hover:border-emerald-500 bg-slate-50 hover:bg-emerald-50/70 dark:bg-white/5 dark:border-white/10 dark:hover:bg-emerald-500/10 text-slate-900 dark:text-white font-medium text-sm transition-all flex items-center justify-between group active:scale-95 cursor-pointer shadow-xs"
                  >
                    <span className="font-semibold">3 курс (ИЛИ остается еще год учёбы)</span>
                    <ArrowRight className="w-4 h-4 text-emerald-600 dark:text-emerald-400 group-hover:translate-x-1 transition-transform" />
                  </button>

                  <button
                    type="button"
                    onClick={() => {
                      setUniversityYear("senior");
                    }}
                    className="w-full p-4 rounded-2xl border border-slate-200 hover:border-amber-400 bg-slate-50 hover:bg-slate-100 dark:bg-white/5 dark:border-white/10 text-slate-700 dark:text-slate-300 font-medium text-sm transition-all flex items-center justify-between group active:scale-95 cursor-pointer"
                  >
                    <span>Выпускной курс (заканчиваю летом)</span>
                    <span className="text-xs text-amber-700 dark:text-amber-400 font-mono font-semibold">Нюанс J-1</span>
                  </button>
                </div>

                {universityYear === "senior" && (
                  <motion.div
                    initial={{ opacity: 0, height: 0 }}
                    animate={{ opacity: 1, height: "auto" }}
                    className="mt-6 p-4 rounded-2xl bg-amber-50 border border-amber-200 text-amber-900 dark:bg-amber-500/10 dark:border-amber-500/25 dark:text-slate-200 text-left text-xs font-doc"
                  >
                    <div className="flex items-center gap-2 font-bold text-amber-700 dark:text-amber-400 mb-1">
                      <AlertTriangle className="w-4 h-4" />
                      <span>Ограничение для выпускников</span>
                    </div>
                    По правилам визы J-1 студенты выпускного курса без планов на российскую магистратуру не могут участвовать. Но если ты поступаешь в магистратуру в РФ — участие разрешено! Напиши куратору, чтобы разобрать твою ситуацию.
                    <div className="mt-3 flex gap-2">
                      <a
                        href="https://t.me/ilyan_vas"
                        target="_blank"
                        rel="noreferrer"
                        className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-xl bg-amber-600 hover:bg-amber-700 text-white font-mono text-[11px] font-bold cursor-pointer"
                      >
                        <Send className="w-3 h-3" /> Разобрать случай с куратором
                      </a>
                      <button
                        type="button"
                        onClick={() => setStep(3)}
                        className="px-3 py-1.5 rounded-xl bg-slate-200 hover:bg-slate-300 dark:bg-white/10 text-[11px] font-mono text-slate-800 dark:text-slate-300 cursor-pointer"
                      >
                        Всё равно продолжить
                      </button>
                    </div>
                  </motion.div>
                )}
              </motion.div>
            )}

            {/* Step 3: English Level */}
            {step === 3 && (
              <motion.div
                key="step-3"
                initial={{ opacity: 0, y: 15 }}
                animate={{ opacity: 1, y: 0 }}
                exit={{ opacity: 0, y: -15 }}
                transition={{ duration: 0.3 }}
                className="max-w-lg mx-auto text-center relative z-10"
              >
                <div className="text-xs font-mono font-semibold uppercase text-emerald-700 dark:text-emerald-400 mb-2">
                  Шаг 3 из 3: Уровень английского языка
                </div>
                <h3 className="text-base sm:text-xl font-bold text-slate-900 dark:text-white mb-2">
                  Оцени свой разговорный и письменный английский
                </h3>
                <p className="text-xs text-slate-600 dark:text-slate-300 mb-6">
                  Выберите примерный уровень владения языком на текущий момент.
                </p>

                <div className="space-y-3">
                  <button
                    type="button"
                    onClick={() => {
                      setEnglishLevel("b1");
                      setStep(4);
                    }}
                    className="w-full p-4 rounded-2xl border border-slate-200 hover:border-emerald-500 bg-slate-50 hover:bg-emerald-50/70 dark:bg-white/5 dark:border-white/10 dark:hover:bg-emerald-500/10 text-slate-900 dark:text-white font-medium text-sm transition-all flex items-center justify-between group active:scale-95 cursor-pointer shadow-xs"
                  >
                    <div className="text-left">
                      <div className="font-bold flex items-center gap-2">
                        <span>🥉 B1 (Intermediate)</span>
                      </div>
                      <div className="text-xs text-slate-600 dark:text-slate-400 mt-0.5">
                        Понимаю суть, могу изъясняться простыми фразами
                      </div>
                    </div>
                    <span className="text-xs font-mono text-amber-700 dark:text-amber-400 font-bold">75% шансов</span>
                  </button>

                  <button
                    type="button"
                    onClick={() => {
                      setEnglishLevel("b2");
                      setStep(4);
                    }}
                    className="w-full p-4 rounded-2xl border border-slate-200 hover:border-teal-500 bg-slate-50 hover:bg-teal-50/70 dark:bg-white/5 dark:border-white/10 dark:hover:bg-teal-500/10 text-slate-900 dark:text-white font-medium text-sm transition-all flex items-center justify-between group active:scale-95 cursor-pointer shadow-xs"
                  >
                    <div className="text-left">
                      <div className="font-bold flex items-center gap-2">
                        <span>🥈 B2 (Upper-Intermediate)</span>
                      </div>
                      <div className="text-xs text-slate-600 dark:text-slate-400 mt-0.5">
                        Свободно общаюсь на общие темы, смотрю видео без субтитров
                      </div>
                    </div>
                    <span className="text-xs font-mono text-teal-700 dark:text-teal-400 font-bold">88% шансов</span>
                  </button>

                  <button
                    type="button"
                    onClick={() => {
                      setEnglishLevel("c1");
                      setStep(4);
                    }}
                    className="w-full p-4 rounded-2xl border border-emerald-300 hover:border-emerald-600 bg-emerald-50/80 hover:bg-emerald-100/80 dark:bg-emerald-500/10 dark:border-emerald-500/30 dark:hover:bg-emerald-500/20 text-slate-900 dark:text-white font-medium text-sm transition-all flex items-center justify-between group active:scale-95 cursor-pointer shadow-xs"
                  >
                    <div className="text-left">
                      <div className="font-bold flex items-center gap-2">
                        <span>🥇 C1 (Advanced)</span>
                      </div>
                      <div className="text-xs text-slate-600 dark:text-slate-400 mt-0.5">
                        Свободный беглый язык, аргументация и профессиональные темы
                      </div>
                    </div>
                    <span className="text-xs font-mono text-emerald-700 dark:text-emerald-400 font-bold">96% шансов</span>
                  </button>
                </div>
              </motion.div>
            )}

            {/* Step 4: Results & Animated Gauge */}
            {step === 4 && (
              <motion.div
                key="step-4"
                initial={{ opacity: 0, scale: 0.95 }}
                animate={{ opacity: 1, scale: 1 }}
                exit={{ opacity: 0, scale: 0.95 }}
                transition={{ duration: 0.4 }}
                className="max-w-lg mx-auto text-center relative z-10"
              >
                {/* Circular Gauge */}
                <div className="relative w-32 h-32 mx-auto mb-5 flex items-center justify-center">
                  <svg className="w-full h-full -rotate-90" viewBox="0 0 120 120">
                    <circle
                      cx="60"
                      cy="60"
                      r="50"
                      className="stroke-slate-200 dark:stroke-white/10"
                      strokeWidth="10"
                      fill="transparent"
                    />
                    <motion.circle
                      cx="60"
                      cy="60"
                      r="50"
                      stroke={scoreData.strokeColor}
                      strokeWidth="10"
                      strokeDasharray={2 * Math.PI * 50}
                      initial={{ strokeDashoffset: 2 * Math.PI * 50 }}
                      animate={{
                        strokeDashoffset:
                          2 * Math.PI * 50 - (2 * Math.PI * 50 * scoreData.percent) / 100,
                      }}
                      transition={{ duration: 1.2, ease: "easeOut" }}
                      strokeLinecap="round"
                      fill="transparent"
                    />
                  </svg>
                  <div className="absolute inset-0 flex flex-col items-center justify-center">
                    <motion.span
                      initial={{ opacity: 0, scale: 0.5 }}
                      animate={{ opacity: 1, scale: 1 }}
                      transition={{ delay: 0.3, duration: 0.5 }}
                      className="text-2xl sm:text-3xl font-black font-mono text-slate-900 dark:text-white"
                    >
                      {scoreData.percent}%
                    </motion.span>
                    <span className="text-[10px] font-mono text-slate-500 uppercase tracking-wider">
                      шанс победы
                    </span>
                  </div>
                </div>

                {/* Status Badge */}
                <div className="inline-flex items-center gap-1.5 px-3 py-1 rounded-full bento-pill text-xs font-mono font-bold text-slate-800 dark:text-slate-100 mb-3">
                  <Sparkles className="w-3.5 h-3.5 text-amber-500" />
                  <span>{scoreData.label}</span>
                </div>

                <h3 className="text-lg sm:text-xl font-bold text-slate-900 dark:text-white mb-2">
                  Ты проходишь базовые фильтры отбора!
                </h3>

                <p className="text-xs sm:text-sm text-slate-600 dark:text-slate-300 font-doc mb-6 leading-relaxed">
                  {scoreData.description}
                </p>

                <div className="flex flex-col sm:flex-row items-center justify-center gap-3">
                  <a
                    href={`https://t.me/spark_prep_bot?start=calc_${englishLevel || "b1"}_${scoreData.percent}`}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="btn-primary-mono w-full sm:w-auto px-6 py-3.5 text-xs sm:text-sm font-bold flex items-center justify-center gap-2 group shadow-xl cursor-pointer"
                  >
                    <Send className="w-4 h-4" />
                    <span>Разобрать заявку ➔</span>
                  </a>

                  <button
                    type="button"
                    onClick={handleReset}
                    className="px-4 py-3 rounded-xl bg-slate-100 hover:bg-slate-200 border border-slate-200 text-xs font-mono font-semibold text-slate-700 hover:text-slate-950 dark:bg-white/10 dark:hover:bg-white/15 dark:border-white/10 dark:text-slate-300 dark:hover:text-white flex items-center gap-1.5 transition-all cursor-pointer"
                  >
                    <RotateCcw className="w-3.5 h-3.5" />
                    <span>Пройти заново</span>
                  </button>
                </div>
              </motion.div>
            )}
          </AnimatePresence>
        </motion.div>
      </div>
    </AnimatePresence>
  );
}
