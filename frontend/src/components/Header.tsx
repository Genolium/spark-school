"use client";

import React, { useState, useEffect } from "react";
import Link from "next/link";
import { ArrowUpRight, Send, Menu, X, Sun, Moon } from "lucide-react";
import { useTheme } from "@/context/ThemeContext";

interface HeaderProps {
  onOpenPayment?: () => void;
  onOpenLeadMagnet?: () => void;
}

export function Header({ onOpenPayment }: HeaderProps) {
  const { theme, toggleTheme } = useTheme();
  const [scrolled, setScrolled] = useState(false);
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false);

  useEffect(() => {
    const handleScroll = () => {
      setScrolled(window.scrollY > 30);
    };
    window.addEventListener("scroll", handleScroll);
    return () => window.removeEventListener("scroll", handleScroll);
  }, []);

  const navLinks = [
    { label: "Программа", href: "/#grant" },
    { label: "Маршрут", href: "/#roadmap" },
    { label: "Кампусы", href: "/#map" },
    { label: "Автор", href: "/#author" },
    { label: "Тариф", href: "/#pricing" },
    // { label: "Партнёрам", href: "/partner" },
  ];

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
    <header className="fixed top-4 sm:top-6 left-1/2 -translate-x-1/2 z-50 w-[94%] max-w-5xl">
      <nav
        className={`bento-nav-pill px-3.5 sm:px-6 py-2.5 sm:py-3 rounded-full flex items-center justify-between transition-all duration-300 ${
          scrolled ? "shadow-[0_25px_60px_rgba(0,0,0,0.7)]" : ""
        }`}
      >
        {/* Left Side: Brand Row Logo & Desktop Nav */}
        <div className="flex items-center gap-5 lg:gap-8">
          <Link href="/" className="flex items-center shrink-0">
            {/* Mobile: Clean square spark logo */}
            <img
              src="/logos/logo.svg"
              alt="так называемый SPARK"
              width={32}
              height={32}
              decoding="async"
              className="h-7 w-7 sm:h-8 sm:w-8 object-contain md:hidden block transition-transform hover:scale-105 active:scale-95"
            />
            {/* Desktop: Horizontal text row brand lockup */}
            <img
              src="/logos/logo-text-light-row.png"
              alt="так называемый SPARK"
              width={178}
              height={28}
              decoding="async"
              className="h-7 w-auto object-contain dark:hidden hidden md:block transition-opacity hover:opacity-90"
            />
            <img
              src="/logos/logo-text-dark-row.png"
              alt="так называемый SPARK"
              width={178}
              height={28}
              decoding="async"
              className="h-7 w-auto object-contain hidden dark:md:block transition-opacity hover:opacity-90"
            />
          </Link>

          {/* Desktop Navigation Links */}
          <div className="hidden md:flex items-center gap-5 lg:gap-6 text-xs lg:text-[13px] font-medium text-slate-600 dark:text-slate-300">
            {navLinks.map((link) => (
              <Link
                key={link.label}
                href={link.href}
                onClick={(e) => handleNavClick(e, link.href)}
                className="hover:text-emerald-600 dark:hover:text-emerald-400 transition-colors duration-200"
              >
                {link.label}
              </Link>
            ))}
          </div>
        </div>

        {/* Right Action Trio */}
        <div className="flex items-center gap-2 sm:gap-2.5">
          {/* <a
            href="https://t.me/spark_prep_bot"
            target="_blank"
            rel="noreferrer"
            className="hidden sm:inline-flex items-center gap-1.5 px-3.5 py-2 rounded-full bento-pill text-xs font-medium text-slate-700 dark:text-slate-200 hover:text-emerald-700 dark:hover:text-emerald-300 hover:bg-emerald-500/15 dark:hover:bg-emerald-500/20 transition-all"
          >
            <Send className="w-3.5 h-3.5 text-sky-500 dark:text-sky-400" />
            <span>TG Бот</span>
          </a> */}

          {/* Theme Switcher Toggle */}
          <button
            onClick={toggleTheme}
            aria-label={theme === "dark" ? "Включить светлую тему" : "Включить тёмную тему"}
            className="w-8 h-8 rounded-full flex items-center justify-center bento-pill transition-transform hover:scale-105 active:scale-95 shrink-0"
            title={theme === "dark" ? "Переключить на светлую тему" : "Переключить на тёмную тему"}
          >
            {theme === "dark" ? (
              <Sun className="w-4 h-4 text-amber-300" />
            ) : (
              <Moon className="w-4 h-4 text-emerald-700" />
            )}
          </button>

          <a
            href="https://t.me/ilyan_vas"
            target="_blank"
            rel="noopener noreferrer"
            className="btn-primary-mono px-3.5 sm:px-5 py-2 text-xs sm:text-[13px] flex items-center gap-1.5 active:scale-95 shadow-lg shrink-0"
          >
            <span>Занять место</span>
            <ArrowUpRight className="w-3.5 h-3.5 stroke-[2.5]" />
          </a>

          {/* Mobile Menu Trigger Button */}
          <button
            onClick={() => setMobileMenuOpen(!mobileMenuOpen)}
            className="md:hidden flex items-center gap-1.5 px-2.5 sm:px-3 py-1.5 rounded-full bento-pill text-xs font-medium text-slate-700 dark:text-slate-200 hover:text-emerald-600 dark:hover:text-white shrink-0"
            aria-label="Меню"
          >
            {mobileMenuOpen ? <X className="w-4 h-4" /> : <Menu className="w-4 h-4" />}
            <span className="hidden xs:inline">Меню</span>
          </button>
        </div>
      </nav>

      {/* Mobile Drawer Menu */}
      {mobileMenuOpen && (
        <div className="md:hidden mt-3 p-4 rounded-3xl bento-card-dark border border-emerald-500/20 dark:border-white/20 shadow-2xl flex flex-col gap-3">
          {navLinks.map((link) => (
            <Link
              key={link.label}
              href={link.href}
              onClick={(e) => {
                handleNavClick(e, link.href);
                setMobileMenuOpen(false);
              }}
              className="px-3 py-2 text-sm font-medium text-slate-800 dark:text-slate-200 hover:text-emerald-600 dark:hover:text-white hover:bg-emerald-500/10 dark:hover:bg-white/5 rounded-xl transition-all"
            >
              {link.label}
            </Link>
          ))}
          <div className="pt-2 border-t border-emerald-500/15 dark:border-white/10 flex flex-col gap-2">
            <button
              onClick={toggleTheme}
              className="flex items-center justify-between p-2.5 rounded-full bento-pill text-xs font-medium"
            >
              <span>Тема оформления:</span>
              <span className="flex items-center gap-1.5 font-bold">
                {theme === "dark" ? (
                  <>
                    <Sun className="w-3.5 h-3.5 text-amber-300" /> Тёмная
                  </>
                ) : (
                  <>
                    <Moon className="w-3.5 h-3.5 text-emerald-700" /> Светлая
                  </>
                )}
              </span>
            </button>
            {/* <a
              href="https://t.me/spark_prep_bot"
              target="_blank"
              rel="noreferrer"
              className="flex items-center justify-center gap-2 p-2.5 rounded-full bento-pill text-xs font-medium text-white"
            >
              <Send className="w-3.5 h-3.5 text-sky-400" />
              <span>Перейти в Telegram-бота</span>
            </a> */}
          </div>
        </div>
      )}
    </header>
  );
}
