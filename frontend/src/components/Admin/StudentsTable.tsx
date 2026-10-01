"use client";

import React, { useState } from "react";
import { Search, UserCheck, Lock, Send, Check, Shield } from "lucide-react";
import { UserProfile } from "@/lib/api";

interface StudentsTableProps {
  students: UserProfile[];
  onToggleAccess: (userId: number, currentAccess: boolean) => void;
}

export function StudentsTable({ students, onToggleAccess }: StudentsTableProps) {
  const [query, setQuery] = useState("");

  const filtered = students.filter((s) => {
    const q = query.toLowerCase();
    return (
      s.username?.toLowerCase().includes(q) ||
      s.first_name?.toLowerCase().includes(q) ||
      s.last_name?.toLowerCase().includes(q) ||
      String(s.telegram_id).includes(q)
    );
  });

  return (
    <div className="bento-card-dark p-6 sm:p-8 rounded-[32px] border border-white/15 mb-8">
      {/* Header and Search */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 mb-6">
        <div>
          <h3 className="text-xl font-bold text-white">
            Управление студентами и ручной эквайринг
          </h3>
          <p className="text-xs font-mono text-slate-400 mt-1">
            Переключайте тумблер доступа при поступлении оплаты на карту самозанятого или через СБП
          </p>
        </div>

        <div className="relative w-full sm:w-72">
          <Search className="w-4 h-4 text-slate-400 absolute left-3.5 top-1/2 -translate-y-1/2" />
          <input
            type="text"
            placeholder="Поиск по @username, имени, ID..."
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            className="w-full pl-10 pr-4 py-2 text-xs font-mono bg-white/10 border border-white/15 rounded-xl text-white placeholder-slate-400 focus:outline-none focus:border-amber-400"
          />
        </div>
      </div>

      {/* Table */}
      <div className="overflow-x-auto">
        <table className="w-full text-left text-xs font-mono">
          <thead>
            <tr className="border-b border-white/10 text-slate-400 uppercase tracking-wider">
              <th className="pb-3 font-semibold">Студент</th>
              <th className="pb-3 font-semibold">Telegram ID</th>
              <th className="pb-3 font-semibold">Роль</th>
              <th className="pb-3 font-semibold">Статус доступа</th>
              <th className="pb-3 font-semibold text-right">Управление доступом</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-white/5">
            {filtered.map((s) => (
              <tr key={s.id} className="hover:bg-white/[0.02] transition-colors">
                <td className="py-4">
                  <div className="flex items-center gap-3">
                    <div className="w-8 h-8 rounded-full bg-white/10 text-white font-bold flex items-center justify-center text-xs shrink-0">
                      {s.first_name ? s.first_name[0] : "U"}
                    </div>
                    <div>
                      <div className="text-white font-bold text-sm">
                        {s.first_name} {s.last_name}
                      </div>
                      <div className="text-slate-400 flex items-center gap-1">
                        <span>@{s.username || "no_username"}</span>
                      </div>
                    </div>
                  </div>
                </td>

                <td className="py-4 text-slate-300">
                  {s.telegram_id}
                </td>

                <td className="py-4">
                  <span
                    className={`px-2 py-0.5 rounded-full text-[10px] font-bold ${
                      s.role === "admin"
                        ? "bg-amber-500/20 text-amber-300 border border-amber-500/30"
                        : "bg-white/10 text-slate-300"
                    }`}
                  >
                    {s.role === "admin" ? "Администратор" : "Студент"}
                  </span>
                </td>

                <td className="py-4">
                  {s.has_access ? (
                    <span className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-[11px] font-semibold bg-emerald-500/15 text-emerald-400 border border-emerald-500/25">
                      <UserCheck className="w-3.5 h-3.5" />
                      Активен
                    </span>
                  ) : (
                    <span className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-[11px] font-semibold bg-rose-500/15 text-rose-300 border border-rose-500/25">
                      <Lock className="w-3.5 h-3.5" />
                      Закрыт
                    </span>
                  )}
                </td>

                <td className="py-4 text-right">
                  <button
                    onClick={() => onToggleAccess(s.id, s.has_access)}
                    className={`px-4 py-2 rounded-xl text-xs font-bold transition-all ${
                      s.has_access
                        ? "bg-rose-500/20 text-rose-300 hover:bg-rose-500/30 border border-rose-500/30"
                        : "bg-emerald-500/20 text-emerald-400 hover:bg-emerald-500/30 border border-emerald-500/30"
                    }`}
                  >
                    {s.has_access ? "Отозвать доступ" : "Активировать доступ (6 900 ₽)"}
                  </button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
