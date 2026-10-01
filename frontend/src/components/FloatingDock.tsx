"use client";

import React from "react";
import Link from "next/link";
import { BookOpen, Map, CreditCard, Send } from "lucide-react";

export function FloatingDock() {
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

  const dockItems: Array<{
    label: string;
    href: string;
    icon: React.ComponentType<{ className?: string }>;
    external?: boolean;
  }> = [
    { label: "Канал", href: "/#features", icon: BookOpen },
    { label: "Маршрут", href: "/#roadmap", icon: Map },
    { label: "Тариф", href: "/#pricing", icon: CreditCard },
    // { label: "Бот", href: "https://t.me/spark_prep_bot", icon: Send, external: true },
  ];

  return (
    <div className="fixed bottom-5 left-1/2 -translate-x-1/2 z-40 sm:hidden w-[90%] max-w-sm">
      <nav className="bento-nav-pill px-3 py-2 rounded-full flex items-center justify-around shadow-[0_12px_40px_rgba(5,150,105,0.15)] dark:shadow-[0_20px_50px_rgba(0,0,0,0.8)] border border-emerald-500/20 dark:border-white/20 bg-white/95 dark:bg-[rgba(18,22,31,0.95)] backdrop-blur-xl">
        {dockItems.map((item, idx) => {
          const Icon = item.icon;
          return item.external ? (
            <a
              key={idx}
              href={item.href}
              target="_blank"
              rel="noreferrer"
              className="flex flex-col items-center gap-1 py-1 px-3 text-slate-600 dark:text-slate-300 hover:text-emerald-600 dark:hover:text-white transition-colors"
            >
              <div className="w-7 h-7 rounded-full bg-emerald-500/10 dark:bg-white/10 flex items-center justify-center text-emerald-700 dark:text-white">
                <Icon className="w-3.5 h-3.5" />
              </div>
              <span className="text-[10px] font-mono font-medium">{item.label}</span>
            </a>
          ) : (
            <Link
              key={idx}
              href={item.href}
              onClick={(e) => handleNavClick(e, item.href)}
              className="flex flex-col items-center gap-1 py-1 px-3 text-slate-600 dark:text-slate-300 hover:text-emerald-600 dark:hover:text-white transition-colors"
            >
              <div className="w-7 h-7 rounded-full bg-emerald-500/10 dark:bg-white/10 flex items-center justify-center text-emerald-700 dark:text-white">
                <Icon className="w-3.5 h-3.5" />
              </div>
              <span className="text-[10px] font-mono font-medium">{item.label}</span>
            </Link>
          );
        })}
      </nav>
    </div>
  );
}
