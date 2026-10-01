"use client";

import React from "react";
import { BlurImage } from "./BlurImage";
import { motion } from "framer-motion";

export function AuthorSection() {
  return (
    <section id="author" className="relative w-full max-w-[1520px] mx-auto px-3 sm:px-6 py-16 sm:py-24 scroll-mt-20 sm:scroll-mt-24">
      {/* Section Header */}
      <div className="max-w-3xl mb-12 sm:mb-16">
        <div className="inline-flex items-center gap-2 px-3.5 py-1.5 rounded-full bento-pill text-xs font-mono text-emerald-800 dark:text-emerald-300 mb-4">
          <span className="w-2 h-2 rounded-full bg-emerald-500" />
          <span>Автор и ментор акселератора</span>
        </div>
        <h2 className="font-editorial text-3xl sm:text-5xl font-bold tracking-tight text-slate-900 dark:text-white leading-[1.1]">
          Практический опыт <br />
          <span className="text-emerald-600 dark:text-emerald-400 font-serif italic">из первых рук финалиста</span>
        </h2>
        <p className="mt-4 text-base text-slate-600 dark:text-slate-300 leading-relaxed font-normal">
          Никакой сухой академической теории. Только проверенные на практике техники студента, выигравшего грант $20,000 с первого раза.
        </p>
      </div>

      {/* Main Author Bento Master Container */}
      <div className="bento-card-dark p-6 sm:p-10 lg:p-12 rounded-2xl sm:rounded-3xl border border-emerald-500/20 dark:border-white/20 relative overflow-hidden shadow-[0_12px_35px_rgba(16,185,129,0.06)] dark:shadow-none">
        
        <div className="grid grid-cols-1 lg:grid-cols-12 gap-10 items-center">
          
          {/* Left Column: Editorial Photo with Hotspot Micro-Chips */}
          <div className="lg:col-span-5 relative">
            <div className="relative w-full aspect-[4/5] rounded-2xl overflow-hidden border border-emerald-500/20 dark:border-white/20 shadow-2xl">
              <BlurImage
                src="/ilya-author.jpg"
                alt="так называемый Иль — Финалист программы SPARK 2026, грант $20,000"
                fill
                loading="lazy"
                quality={100}
                unoptimized
                className="object-cover object-top"
                sizes="(max-width: 1024px) 100vw, 40vw"
              />
              
              {/* Subtle Gradient Overlays */}
              <div className="absolute inset-0 bg-gradient-to-t from-black/80 via-transparent to-black/30" />
            </div>
          </div>

          {/* Right Column: Bio, Story, Metrics, and Authentic Quote */}
          <div className="lg:col-span-7 flex flex-col justify-between">
            
            <div>
              {/* Contrast Tag: Voenmeh ➔ Wyoming */}
              <div className="inline-flex items-center gap-2 px-3 py-1 rounded-full tag-neutral text-xs font-mono font-medium mb-4">
                <span>Военмех (СПб) ➔ University of Wyoming (США)</span>
              </div>

              <h3 className="font-editorial text-3xl sm:text-4xl font-bold text-slate-900 dark:text-white mb-3">
                так называемый Иль
              </h3>
              
              <div className="text-xs sm:text-sm font-mono text-slate-600 dark:text-slate-300 mb-6 flex flex-wrap items-center gap-3">
                <span>Финалист SPARK &apos;26</span>
                <span className="text-slate-400 dark:text-white/30">•</span>
                <span>Полный грант Госдепа США $20,000</span>
                <span className="text-slate-400 dark:text-white/30">•</span>
                <span className="text-emerald-700 dark:text-emerald-400 font-semibold">J-1 Visa Approved</span>
              </div>

              {/* Highlighted Bio Narrative with Strong Emphasis */}
              <div className="font-doc p-6 sm:p-7 rounded-3xl bg-emerald-500/[0.04] border border-emerald-500/15 dark:bg-white/[0.04] dark:border-white/15 mb-8 relative space-y-3.5">
                <p className="text-base sm:text-lg text-slate-900 dark:text-white font-medium leading-relaxed">
                  Обычный студент БГТУ &quot;Военмех&quot; из Санкт-Петербурга без дорогих репетиторов.
                </p>
                <p className="text-sm sm:text-base text-slate-700 dark:text-slate-200 leading-relaxed font-normal">
                  Прошёл конкурс программы SPARK, доказав американской комиссии ценность своего бэкграунда, получил 135 баллов по Duolingo English Test и улетел учиться в США.
                </p>
                <div className="pt-1 flex items-center gap-2 text-xs sm:text-sm font-mono text-emerald-700 dark:text-emerald-400 font-semibold">
                  <span>📍 На программе посетил 10 американских штатов от Калифорнии до Нью-Йорка</span>
                </div>
              </div>
            </div>

            {/* Key Fact Numbers in Space Grotesk */}
            <div className="grid grid-cols-3 gap-4 pt-6 border-t border-emerald-500/15 dark:border-white/10">
              <div>
                <div className="text-xl sm:text-2xl font-black font-mono text-slate-900 dark:text-white">
                  $20,000
                </div>
                <div className="text-[11px] font-mono text-slate-500 dark:text-slate-400 mt-0.5">
                  Выигранный грант
                </div>
              </div>

              <div>
                <div className="text-xl sm:text-2xl font-black font-mono text-slate-900 dark:text-white">
                  C1 - 135 DET
                </div>
                <div className="text-[11px] font-mono text-slate-500 dark:text-slate-400 mt-0.5">
                  Duolingo English Test
                </div>
              </div>

              <div>
                <div className="text-xl sm:text-2xl font-black font-mono text-slate-900 dark:text-white">
                  10 штатов
                </div>
                <div className="text-[11px] font-mono text-slate-500 dark:text-slate-400 mt-0.5">
                  От Калифорнии до Нью-Йорка
                </div>
              </div>
            </div>

          </div>

        </div>

      </div>
    </section>
  );
}
