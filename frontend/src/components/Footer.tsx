"use client";

import React from "react";
import Link from "next/link";
import { Send, Shield, FileText, ArrowUp } from "lucide-react";

export function Footer() {
  const scrollToTop = () => {
    window.scrollTo({ top: 0, behavior: "smooth" });
  };

  const handleNavClick = (e: React.MouseEvent<HTMLAnchorElement>, href: string) => {
    if (href.startsWith("/#") && typeof window !== "undefined" && window.location.pathname === "/") {
      e.preventDefault();
      const id = href.replace("/#", "");
      const el = document.getElementById(id);
      if (el) {
        el.scrollIntoView({ behavior: "smooth" });
        window.history.pushState(null, "", href);
      }
    }
  };

  return (
    <footer className="relative w-full max-w-[1520px] mx-auto px-3 sm:px-6 pt-16 pb-16 sm:pb-20 border-t border-emerald-500/15 dark:border-white/10 mt-16">
      <div className="grid grid-cols-1 md:grid-cols-12 gap-10 items-start mb-12">
        
        {/* Brand & Mission (Span 5) */}
        <div className="md:col-span-5">
          <Link href="/" className="flex items-center mb-5 group inline-flex">
            <img
              src="/logos/logo-text-light-row.png"
              alt="так называемый SPARK"
              width={178}
              height={32}
              loading="lazy"
              decoding="async"
              className="h-7 sm:h-8 w-auto object-contain dark:hidden block transition-opacity hover:opacity-90"
            />
            <img
              src="/logos/logo-text-dark-row.png"
              alt="так называемый SPARK"
              width={178}
              height={32}
              loading="lazy"
              decoding="async"
              className="h-7 sm:h-8 w-auto object-contain hidden dark:block transition-opacity hover:opacity-90"
            />
          </Link>

          <p className="text-xs sm:text-sm text-slate-600 dark:text-slate-400 leading-relaxed max-w-sm mb-6">
            Практический акселератор подготовки к грантовым программам академического обмена США. Личный менторинг, симуляция интервью и разбор победивших заявок.
          </p>

          {/* <div className="flex items-center gap-3">
            <a
              href="https://t.me/spark_prep_bot"
              target="_blank"
              rel="noreferrer"
              className="inline-flex items-center gap-2 px-4 py-2 rounded-full bento-pill text-xs font-mono text-slate-800 dark:text-white hover:bg-emerald-500/10 dark:hover:bg-white/20 transition-all"
            >
              <Send className="w-3.5 h-3.5 text-emerald-600 dark:text-sky-400" />
              <span>@spark_prep_bot</span>
            </a>
          </div> */}
        </div>

        {/* Navigation Quick Links (Span 3) */}
        <div className="md:col-span-3">
          <div className="text-xs font-mono uppercase tracking-wider text-slate-500 dark:text-slate-400 font-bold mb-4">
            Навигация
          </div>
          <ul className="space-y-2 text-xs sm:text-sm text-slate-600 dark:text-slate-300 font-medium">
            <li>
              <Link
                href="/#grant"
                onClick={(e) => handleNavClick(e, "/#grant")}
                className="hover:text-emerald-600 dark:hover:text-white transition-colors"
              >
                Программа акселератора
              </Link>
            </li>
            <li>
              <Link
                href="/#hard-truth"
                onClick={(e) => handleNavClick(e, "/#hard-truth")}
                className="hover:text-emerald-600 dark:hover:text-white transition-colors"
              >
                Ошибки 90% кандидатов
              </Link>
            </li>
            <li>
              <Link
                href="/#roadmap"
                onClick={(e) => handleNavClick(e, "/#roadmap")}
                className="hover:text-emerald-600 dark:hover:text-white transition-colors"
              >
                Маршрут подготовки
              </Link>
            </li>
            <li>
              <Link
                href="/#map"
                onClick={(e) => handleNavClick(e, "/#map")}
                className="hover:text-emerald-600 dark:hover:text-white transition-colors"
              >
                Пул кампусов США (24)
              </Link>
            </li>
            <li>
              <Link
                href="/#author"
                onClick={(e) => handleNavClick(e, "/#author")}
                className="hover:text-emerald-600 dark:hover:text-white transition-colors"
              >
                Автор курса
              </Link>
            </li>
            <li>
              <Link
                href="/#pricing"
                onClick={(e) => handleNavClick(e, "/#pricing")}
                className="hover:text-emerald-600 dark:hover:text-white transition-colors"
              >
                Стоимость участия
              </Link>
            </li>
          </ul>
        </div>

        {/* Legal Details (Span 4) */}
        <div className="md:col-span-4">
          <div className="text-xs font-mono uppercase tracking-wider text-slate-500 dark:text-slate-400 font-bold mb-4">
            Юридическая информация
          </div>
          <div className="text-xs font-doc text-slate-600 dark:text-slate-400 space-y-1.5 mb-6">
            <div>Самозанятый Васюнин Илья Олегович</div>
            <div>ИНН: 780739313219</div>
            <div>Email поддержки: vas.ilyan@icloud.com</div>
          </div>

          <div className="flex flex-col gap-2 text-xs font-doc text-slate-600 dark:text-slate-400">
            <Link href="/offer" className="hover:text-emerald-600 dark:hover:text-white transition-colors underline underline-offset-2">
              Публичная оферта на оказание услуг
            </Link>
            <Link href="/privacy" className="hover:text-emerald-600 dark:hover:text-white transition-colors underline underline-offset-2">
              Политика конфиденциальности данных
            </Link>
          </div>
        </div>

      </div>

      {/* Mandatory Program Disclaimer */}
      <div className="p-5 rounded-2xl bento-card-dark border border-emerald-500/15 dark:border-white/10 text-[11px] sm:text-xs text-slate-600 dark:text-slate-400 leading-relaxed font-doc mb-8">
        <div className="text-slate-900 dark:text-white font-semibold mb-1 flex items-center gap-1.5">
          <Shield className="w-3.5 h-3.5 text-amber-500" />
          <span>Официальный дисклеймер программы:</span>
        </div>
        «Проект «так называемый SPARK» является независимым частным образовательным проектом. Проект не является официальным представителем программы SPARK, American Councils или Государственного департамента США. Все упоминания торговых марок, программ и университетов используются исключительно в информационных целях для описания академического опыта автора».
      </div>

      {/* Bottom Bar */}
      <div className="flex flex-col sm:flex-row items-center justify-between gap-4 pt-6 border-t border-emerald-500/10 dark:border-white/5 text-xs font-mono text-slate-500">
        <div>
          © {new Date().getFullYear()} «так называемый SPARK». Все права защищены.
        </div>

        <button
          onClick={scrollToTop}
          className="flex items-center gap-1.5 text-slate-500 hover:text-emerald-600 dark:text-slate-400 dark:hover:text-white transition-colors"
        >
          <span>Наверх</span>
          <ArrowUp className="w-3.5 h-3.5" />
        </button>
      </div>
    </footer>
  );
}
