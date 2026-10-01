"use client";

import React, { useState } from "react";
import { Check, X, Copy, CheckCheck, Wallet, ArrowUpRight } from "lucide-react";
import { PayoutRequest } from "@/lib/api";

interface PayoutsTableProps {
  payouts: PayoutRequest[];
  onApprove: (id: number) => void;
  onReject: (id: number) => void;
}

export function PayoutsTable({ payouts, onApprove, onReject }: PayoutsTableProps) {
  const [copiedId, setCopiedId] = useState<number | null>(null);

  const handleCopy = (id: number, text: string) => {
    navigator.clipboard.writeText(text);
    setCopiedId(id);
    setTimeout(() => setCopiedId(null), 2000);
  };

  return (
    <div className="bento-card-dark p-6 sm:p-8 rounded-[32px] border border-white/15">
      <div className="flex items-center justify-between mb-6">
        <div>
          <h3 className="text-xl font-bold text-white flex items-center gap-2">
            <Wallet className="w-5 h-5 text-amber-400" />
            <span>Заявки на вывод реферальных средств (Партнёры 15%)</span>
          </h3>
          <p className="text-xs font-mono text-slate-400 mt-1">
            Скопируйте реквизиты в один клик и сделайте перевод через СБП в приложении банка
          </p>
        </div>
      </div>

      <div className="overflow-x-auto">
        <table className="w-full text-left text-xs font-mono">
          <thead>
            <tr className="border-b border-white/10 text-slate-400 uppercase tracking-wider">
              <th className="pb-3 font-semibold">ID / Дата</th>
              <th className="pb-3 font-semibold">Сумма</th>
              <th className="pb-3 font-semibold">Реквизиты получателя (СБП)</th>
              <th className="pb-3 font-semibold">Статус</th>
              <th className="pb-3 font-semibold text-right">Действия</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-white/5">
            {payouts.map((p) => (
              <tr key={p.id} className="hover:bg-white/[0.02] transition-colors">
                <td className="py-4 text-slate-300">
                  <div className="font-bold text-white">#{p.id}</div>
                  <div className="text-[11px] text-slate-400">{p.created_at}</div>
                </td>

                <td className="py-4 font-black text-white text-sm">
                  {p.amount.toLocaleString("ru-RU")} ₽
                </td>

                <td className="py-4">
                  <div className="flex items-center gap-2 max-w-md">
                    <span className="text-slate-200 truncate">{p.payment_details}</span>
                    <button
                      onClick={() => handleCopy(p.id, p.payment_details)}
                      className="p-1.5 rounded-lg bg-white/10 hover:bg-white/20 text-slate-300 hover:text-white transition-all shrink-0"
                      title="Скопировать реквизиты"
                    >
                      {copiedId === p.id ? (
                        <CheckCheck className="w-3.5 h-3.5 text-emerald-400" />
                      ) : (
                        <Copy className="w-3.5 h-3.5" />
                      )}
                    </button>
                  </div>
                </td>

                <td className="py-4">
                  {p.status === "pending" && (
                    <span className="px-2.5 py-1 rounded-full text-[10px] font-bold bg-amber-500/20 text-amber-300 border border-amber-500/30">
                      На рассмотрении
                    </span>
                  )}
                  {p.status === "paid" && (
                    <span className="px-2.5 py-1 rounded-full text-[10px] font-bold bg-emerald-500/20 text-emerald-400 border border-emerald-500/30">
                      Выплачено ✓
                    </span>
                  )}
                  {p.status === "rejected" && (
                    <span className="px-2.5 py-1 rounded-full text-[10px] font-bold bg-rose-500/20 text-rose-300 border border-rose-500/30">
                      Отклонено
                    </span>
                  )}
                </td>

                <td className="py-4 text-right">
                  {p.status === "pending" ? (
                    <div className="flex items-center justify-end gap-2">
                      <button
                        onClick={() => onApprove(p.id)}
                        className="px-3 py-1.5 rounded-xl bg-emerald-500/20 text-emerald-400 hover:bg-emerald-500/30 border border-emerald-500/30 font-bold transition-all"
                      >
                        Подтвердить
                      </button>
                      <button
                        onClick={() => onReject(p.id)}
                        className="px-3 py-1.5 rounded-xl bg-rose-500/20 text-rose-300 hover:bg-rose-500/30 border border-rose-500/30 font-bold transition-all"
                      >
                        Отклонить
                      </button>
                    </div>
                  ) : (
                    <span className="text-slate-400 text-[11px]">—</span>
                  )}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
