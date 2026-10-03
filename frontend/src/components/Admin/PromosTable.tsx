"use client";

import React, { useState } from "react";
import { PromoCode, api } from "@/lib/api";
import { Tag, Plus, CheckCircle, XCircle, Search, RefreshCw, Copy, Check } from "lucide-react";

interface PromosTableProps {
  promos: PromoCode[];
  onRefresh: () => void;
}

export function PromosTable({ promos, onRefresh }: PromosTableProps) {
  const [search, setSearch] = useState("");
  const [copiedCode, setCopiedCode] = useState<string | null>(null);
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [editingId, setEditingId] = useState<number | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<PromoCode | null>(null);

  // Form states
  const [newCode, setNewCode] = useState("");
  const [newOwner, setNewOwner] = useState("");
  const [newDiscount, setNewDiscount] = useState(5);
  const [newReward, setNewReward] = useState(1000);
  const [creating, setCreating] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const filtered = promos.filter(
    (p) =>
      p.code.toLowerCase().includes(search.toLowerCase()) ||
      (p.owner_username && p.owner_username.toLowerCase().includes(search.toLowerCase()))
  );

  const handleCopy = (code: string) => {
    navigator.clipboard.writeText(code);
    setCopiedCode(code);
    setTimeout(() => setCopiedCode(null), 2000);
  };

  const handleToggle = async (id: number) => {
    await api.toggleAdminPromoCode(id);
    onRefresh();
  };

  const openCreate = () => {
    setEditingId(null);
    setNewCode("");
    setNewOwner("");
    setNewDiscount(5);
    setNewReward(1000);
    setError(null);
    setShowCreateModal(true);
  };

  const openEdit = (p: PromoCode) => {
    setEditingId(p.id);
    setNewCode(p.code);
    setNewOwner(p.owner_username || "");
    setNewDiscount(p.discount_percent || 5);
    setNewReward(p.reward_amount);
    setError(null);
    setShowCreateModal(true);
  };

  const handleDelete = async () => {
    if (!deleteTarget) return;
    await api.deleteAdminPromoCode(deleteTarget.id);
    setDeleteTarget(null);
    onRefresh();
  };

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!newCode.trim()) return;
    setCreating(true);
    setError(null);

    const payload = {
      code: newCode.trim().toUpperCase(),
      owner_username: newOwner.trim(),
      discount_percent: newDiscount,
      reward_amount: newReward,
    };
    const res = editingId
      ? await api.updateAdminPromoCode(editingId, payload)
      : await api.createAdminPromoCode(payload);

    setCreating(false);
    if (res.success) {
      setNewCode("");
      setNewOwner("");
      setEditingId(null);
      setShowCreateModal(false);
      onRefresh();
    } else {
      setError(res.error || "Не удалось сохранить промокод");
    }
  };


  return (
    <div className="space-y-6">
      {/* Top Bar: Search + Create Button */}
      <div className="flex flex-col sm:flex-row items-stretch sm:items-center justify-between gap-4 p-4 rounded-2xl bg-white/70 dark:bg-white/[0.03] border border-slate-200 dark:border-white/10 shadow-sm">
        <div className="relative flex-1 max-w-md">
          <Search className="w-4 h-4 absolute left-3.5 top-1/2 -translate-y-1/2 text-slate-400" />
          <input
            type="text"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder="Поиск по коду или амбассадору..."
            className="w-full pl-10 pr-4 py-2 text-xs font-mono bg-slate-50 dark:bg-black/40 border border-slate-200 dark:border-white/10 rounded-xl text-slate-900 dark:text-white placeholder-slate-400 dark:placeholder-slate-500 focus:outline-none focus:border-amber-500"
          />
        </div>

        <button
          onClick={openCreate}
          className="px-4 py-2 rounded-xl bg-amber-500 hover:bg-amber-400 text-black text-xs font-mono font-bold flex items-center justify-center gap-2 transition-all shadow-md active:scale-95"
        >
          <Plus className="w-4 h-4" />
          <span>Создать промокод</span>
        </button>
      </div>

      {/* Promos Table */}
      <div className="overflow-x-auto rounded-2xl border border-slate-200 dark:border-white/10 bg-white dark:bg-black/30 shadow-sm">
        <table className="w-full text-left text-xs font-mono">
          <thead>
            <tr className="border-b border-slate-200 dark:border-white/10 bg-slate-50 dark:bg-white/[0.02] text-slate-600 dark:text-slate-400 font-semibold">
              <th className="py-3 px-4">Код скидки</th>
              <th className="py-3 px-4">Скидка студенту</th>
              <th className="py-3 px-4">Амбассадор / Партнёр</th>
              <th className="py-3 px-4">Использований</th>
              <th className="py-3 px-4">Вознаграждение</th>
              <th className="py-3 px-4">Статус</th>
              <th className="py-3 px-4 text-right">Действие</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-slate-100 dark:divide-white/5 text-slate-800 dark:text-slate-300">
            {filtered.length === 0 ? (
              <tr>
                <td colSpan={7} className="py-8 text-center text-slate-500 font-mono">
                  Промокоды не найдены
                </td>
              </tr>
            ) : (
              filtered.map((promo) => (
                <tr key={promo.id} className="hover:bg-slate-50 dark:hover:bg-white/[0.02] transition-colors">
                  <td className="py-3.5 px-4">
                    <div className="flex items-center gap-2">
                      <span className="font-bold text-amber-600 dark:text-amber-400 bg-amber-500/10 border border-amber-500/25 px-2 py-0.5 rounded">
                        {promo.code}
                      </span>
                      <button
                        onClick={() => handleCopy(promo.code)}
                        className="text-slate-400 hover:text-slate-700 dark:hover:text-white p-1 rounded transition-colors"
                        title="Скопировать"
                      >
                        {copiedCode === promo.code ? (
                          <Check className="w-3.5 h-3.5 text-emerald-500" />
                        ) : (
                          <Copy className="w-3.5 h-3.5" />
                        )}
                      </button>
                    </div>
                  </td>
                  <td className="py-3.5 px-4 font-bold text-emerald-600 dark:text-emerald-400">
                    -{promo.discount_percent || 5}% (345 ₽)
                  </td>
                  <td className="py-3.5 px-4">
                    {promo.owner_username ? (
                      <span className="text-sky-600 dark:text-sky-400 font-medium">@{promo.owner_username}</span>
                    ) : (
                      <span className="text-slate-400">—</span>
                    )}
                  </td>
                  <td className="py-3.5 px-4">
                    <span className="px-2 py-0.5 rounded-full bg-slate-100 dark:bg-white/5 border border-slate-200 dark:border-white/10 font-bold text-slate-800 dark:text-white">
                      {promo.uses_count}
                    </span>
                  </td>
                  <td className="py-3.5 px-4 text-amber-700 dark:text-amber-300 font-semibold">
                    {promo.reward_amount.toLocaleString("ru-RU")} ₽
                  </td>
                  <td className="py-3.5 px-4">
                    {promo.is_active ? (
                      <span className="inline-flex items-center gap-1 text-[11px] text-emerald-600 dark:text-emerald-400 font-bold">
                        <CheckCircle className="w-3.5 h-3.5" />
                        <span>Активен</span>
                      </span>
                    ) : (
                      <span className="inline-flex items-center gap-1 text-[11px] text-slate-400 dark:text-slate-500">
                        <XCircle className="w-3.5 h-3.5" />
                        <span>Отключен</span>
                      </span>
                    )}
                  </td>
                  <td className="py-3.5 px-4 text-right">
                    <div className="flex items-center justify-end gap-2">
                      <button
                        onClick={() => handleToggle(promo.id)}
                        className={`px-2.5 py-1 rounded text-[11px] font-bold transition-all border ${
                          promo.is_active
                            ? "bg-rose-500/10 text-rose-600 dark:text-rose-300 border-rose-500/20 hover:bg-rose-500/20"
                            : "bg-emerald-500/10 text-emerald-600 dark:text-emerald-300 border-emerald-500/20 hover:bg-emerald-500/20"
                        }`}
                      >
                        {promo.is_active ? "Деактивировать" : "Активировать"}
                      </button>
                      <button
                        onClick={() => openEdit(promo)}
                        className="px-2.5 py-1 rounded text-[11px] font-bold border border-slate-300 dark:border-white/10 text-slate-700 dark:text-slate-300 hover:bg-slate-100 dark:hover:bg-white/10"
                      >
                        Редактировать
                      </button>
                      <button
                        onClick={() => setDeleteTarget(promo)}
                        className="px-2.5 py-1 rounded text-[11px] font-bold border border-rose-500/30 text-rose-600 dark:text-rose-300 hover:bg-rose-500/10"
                      >
                        Удалить
                      </button>
                    </div>
                  </td>

                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>

      {/* Modal: Create Promo Code */}
      {showCreateModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/60 backdrop-blur-sm animate-in fade-in">
          <div className="bg-white dark:bg-[#12161D] border border-slate-200 dark:border-white/20 rounded-2xl max-w-md w-full p-6 space-y-4 shadow-2xl text-slate-900 dark:text-white">
            <div className="flex items-center justify-between border-b border-slate-200 dark:border-white/10 pb-3">
              <div className="flex items-center gap-2">
                <Tag className="w-4 h-4 text-amber-500" />
                <h3 className="font-editorial text-lg font-bold text-slate-900 dark:text-white">{editingId ? "Редактирование промокода" : "Новый промокод"}</h3>
              </div>
              <button
                onClick={() => setShowCreateModal(false)}
                className="text-slate-400 hover:text-slate-700 dark:hover:text-white"
              >
                ✕
              </button>
            </div>

            {error && (
              <div className="p-3 bg-rose-500/10 border border-rose-500/30 text-rose-600 dark:text-rose-300 rounded-xl text-xs font-mono">
                {error}
              </div>
            )}

            <form onSubmit={handleCreate} className="space-y-4 text-xs font-mono">
              <div>
                <label className="block text-slate-600 dark:text-slate-400 mb-1">Код промокода (на латинице):</label>
                <input
                  type="text"
                  required
                  value={newCode}
                  onChange={(e) => setNewCode(e.target.value.toUpperCase())}
                  placeholder="НАПРИМЕР: ALEX5 или BESTSTUDENT"
                  className="w-full px-3 py-2 bg-slate-50 dark:bg-black/50 border border-slate-300 dark:border-white/10 rounded-xl text-slate-900 dark:text-white font-bold uppercase focus:outline-none focus:border-amber-500"
                />
              </div>

              <div>
                <label className="block text-slate-600 dark:text-slate-400 mb-1">Telegram амбассадора / владельца:</label>
                <input
                  type="text"
                  value={newOwner}
                  onChange={(e) => setNewOwner(e.target.value)}
                  placeholder="@username партнера"
                  className="w-full px-3 py-2 bg-slate-50 dark:bg-black/50 border border-slate-300 dark:border-white/10 rounded-xl text-slate-900 dark:text-white focus:outline-none focus:border-amber-500"
                />
              </div>

              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="block text-slate-600 dark:text-slate-400 mb-1">Скидка клиенту (%):</label>
                  <input
                    type="number"
                    min="1"
                    max="50"
                    value={newDiscount}
                    onChange={(e) => setNewDiscount(parseInt(e.target.value) || 5)}
                    className="w-full px-3 py-2 bg-slate-50 dark:bg-black/50 border border-slate-300 dark:border-white/10 rounded-xl text-slate-900 dark:text-white focus:outline-none focus:border-amber-500"
                  />
                  <span className="text-[10px] text-emerald-600 dark:text-emerald-400 mt-1 block">
                    Цена: {(6900 * (1 - newDiscount / 100)).toFixed(0)} ₽
                  </span>
                </div>
                <div>
                  <label className="block text-slate-600 dark:text-slate-400 mb-1">Выплата партнеру (₽):</label>
                  <input
                    type="number"
                    min="100"
                    step="50"
                    value={newReward}
                    onChange={(e) => setNewReward(parseFloat(e.target.value) || 1000)}
                    className="w-full px-3 py-2 bg-slate-50 dark:bg-black/50 border border-slate-300 dark:border-white/10 rounded-xl text-slate-900 dark:text-white focus:outline-none focus:border-amber-500"
                  />
                </div>
              </div>

              <div className="flex items-center justify-end gap-2 pt-2">
                <button
                  type="button"
                  onClick={() => setShowCreateModal(false)}
                  className="px-4 py-2 rounded-xl text-slate-500 hover:text-slate-900 dark:text-slate-400 dark:hover:text-white"
                >
                  Отмена
                </button>
                <button
                  type="submit"
                  disabled={creating}
                  className="px-4 py-2 rounded-xl bg-amber-500 hover:bg-amber-400 text-black font-bold disabled:opacity-50"
                >
                  {creating ? "Сохранение..." : editingId ? "Сохранить изменения" : "Создать промокод"}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {deleteTarget && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/60 backdrop-blur-sm">
          <div className="bg-white dark:bg-[#12161D] border border-slate-200 dark:border-white/20 rounded-2xl max-w-sm w-full p-6 space-y-4 shadow-2xl text-slate-900 dark:text-white">
            <h3 className="text-lg font-bold">Удалить промокод?</h3>
            <p className="text-xs font-mono text-slate-500 dark:text-slate-400">
              Промокод {deleteTarget.code} будет удалён без возможности восстановления.
            </p>
            <div className="flex gap-3 text-xs font-mono">
              <button
                onClick={() => setDeleteTarget(null)}
                className="flex-1 px-4 py-2 rounded-xl border border-slate-200 dark:border-white/10"
              >
                Отмена
              </button>
              <button
                onClick={handleDelete}
                className="flex-1 px-4 py-2 rounded-xl bg-rose-500 hover:bg-rose-400 text-white font-bold"
              >
                Удалить
              </button>
            </div>
          </div>
        </div>
      )}
    </div>

  );
}
