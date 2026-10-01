import React from "react";
import Link from "next/link";
import { Home, HelpCircle } from "lucide-react";

export const metadata = {
  title: "Страница не найдена",
  description: "Запрошенная страница не существует или была перемещена. Перейдите на главную страницу акселератора так называемый SPARK.",
};

export default function NotFound() {
  return (
    <main className="min-h-screen bg-[var(--bg-base)] text-[var(--text-main)] flex items-center justify-center p-4 sm:p-6 relative overflow-hidden font-sans transition-colors duration-300">
      {/* Background Ambient Glows */}
      <div className="absolute top-1/4 left-1/2 -translate-x-1/2 -translate-y-1/2 w-[500px] h-[500px] bg-amber-500/10 rounded-full blur-[140px] pointer-events-none" />
      <div className="absolute bottom-1/4 right-1/4 w-[400px] h-[400px] bg-sky-500/10 rounded-full blur-[140px] pointer-events-none" />

      <div className="w-full max-w-xl text-center relative z-10">
        {/* Bento Surface Card */}
        <div className="bento-card-dark p-8 sm:p-12 rounded-[36px] border border-emerald-500/15 dark:border-white/15 shadow-2xl relative overflow-hidden">
          {/* Subtle Top Accent */}
          <div className="inline-flex items-center gap-2 px-3.5 py-1.5 rounded-full bento-pill text-xs font-mono text-amber-600 dark:text-amber-400 mb-6 border border-amber-500/20">
            <HelpCircle className="w-3.5 h-3.5" />
            <span>Ошибка 404 • Ресурс не найден</span>
          </div>

          {/* Large Stylized 404 Number */}
          <div className="text-7xl sm:text-9xl font-black font-mono tracking-tighter text-slate-900 dark:text-white drop-shadow-[0_10px_35px_rgba(0,0,0,0.08)] dark:drop-shadow-[0_10px_35px_rgba(245,173,70,0.25)] mb-4">
            404
          </div>

          {/* Single Semantic H1 */}
          <h1 className="font-editorial text-2xl sm:text-3xl font-bold text-slate-900 dark:text-white mb-4 tracking-tight">
            Страница потерялась в пути
          </h1>

          <p className="text-sm sm:text-base text-slate-600 dark:text-slate-300 leading-relaxed font-doc mb-8 max-w-md mx-auto">
            Возможно, ссылка устарела или адрес был введён с опечаткой. Перейдите на главную страницу акселератора.
          </p>

          {/* Single Primary Action Button */}
          <div className="flex justify-center">
            <Link
              href="/"
              className="btn-primary-mono px-8 py-3.5 text-sm flex items-center justify-center gap-2 rounded-2xl active:scale-95 transition-all shadow-xl font-bold"
            >
              <Home className="w-4 h-4" />
              <span>На главную</span>
            </Link>
          </div>
        </div>

        {/* Footer Brand Note */}
        <p className="text-xs font-mono text-slate-500 dark:text-slate-400 mt-6">
          «так называемый SPARK» &copy; 2026–2027 • Все права защищены
        </p>
      </div>
    </main>
  );
}
