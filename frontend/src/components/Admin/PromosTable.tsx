"use client";

import React, { useState } from "react";
import { PromoCode, api } from "@/lib/api";
import {
  Tag,
  Plus,
  CheckCircle,
  XCircle,
  Search,
  Copy,
  Check,
  TrendingUp,
  Users,
  Coins,
  ShieldAlert,
  Percent,
} from "lucide-react";

interface PromosTableProps {
  promos: PromoCode[];
  onRefresh: () => void;
}

export function PromosTable({ promos, onRefresh }: PromosTableProps) {
  const [search, setSearch] = useState("");
  const [filterActive, setFilterActive] = useState<"all" | "active" | "inactive">("all");
  const [copiedCode, setCopiedCode] = useState<string | null>(null);
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [editingId, setEditingId] = useState<number | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<PromoCode | null>(null);
  const [studentsModalPromo, setStudentsModalPromo] = useState<PromoCode | null>(null);

  // Form states
  const [newCode, setNewCode] = useState("");
  const [newOwner, setNewOwner] = useState("");
  const [newDiscount, setNewDiscount] = useState(5);
  const [newReward, setNewReward] = useState(1000);
  const [creating, setCreating] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [coursePricing, setCoursePricing] = useState({ accelerator_price: 10900 });

  React.useEffect(() => {
    api.getCoursePricing().then((p) => {
      if (p && p.accelerator_price > 0) {
        setCoursePricing({ accelerator_price: p.accelerator_price });
      }
    }).catch(() => {});
  }, []);

  // Aggregated analytics calculation. Revenue comes only from the payment
  // ledger returned by the backend; never estimate it from activations.
  const totalActivations = promos.reduce(
    (sum, p) => sum + (p.activation_count ?? p.students?.length ?? 0),
    0
  );
  const totalPaidUses = promos.reduce((sum, p) => sum + (p.paid_uses || 0), 0);
  const totalRevenue = promos.reduce((sum, p) => sum + (p.total_revenue || 0), 0);
  const totalRewards = promos.reduce(
    (sum, p) => sum + (p.paid_uses || 0) * (p.reward_amount || 0),
    0
  );
  const activeCount = promos.filter((p) => p.is_active).length;

  const filtered = promos.filter((p) => {
    const matchesSearch =
      p.code.toLowerCase().includes(search.toLowerCase()) ||
      (p.owner_username && p.owner_username.toLowerCase().includes(search.toLowerCase()));
    if (!matchesSearch) return false;
    if (filterActive === "active") return p.is_active;
    if (filterActive === "inactive") return !p.is_active;
    return true;
  });

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
      owner_username: newOwner.trim().replace(/^@/, ""),
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
      {/* 1. Analytics & KPI Cards */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        {/* Total Promo Revenue */}
        <div className="p-5 rounded-2xl bg-white/80 dark:bg-black/40 border border-slate-200 dark:border-white/10 shadow-sm backdrop-blur-md relative overflow-hidden">
          <div className="flex items-center justify-between">
            <span className="text-[11px] font-mono text-slate-500 uppercase tracking-wider">
              Выручка по промокодам
            </span>
            <div className="w-8 h-8 rounded-xl bg-emerald-500/10 text-emerald-500 flex items-center justify-center">
              <TrendingUp className="w-4 h-4" />
            </div>
          </div>
          <div className="mt-3 text-2xl font-bold font-mono text-slate-900 dark:text-white">
            {totalRevenue.toLocaleString("ru-RU")} ₽
          </div>
          <p className="text-[11px] font-mono text-slate-500 mt-1">
            Сумма оплат с применёнными скидками
          </p>
        </div>

        {/* Total Activations */}
        <div className="p-5 rounded-2xl bg-white/80 dark:bg-black/40 border border-slate-200 dark:border-white/10 shadow-sm backdrop-blur-md relative overflow-hidden">
          <div className="flex items-center justify-between">
            <span className="text-[11px] font-mono text-slate-500 uppercase tracking-wider">
              Активаций промокода
            </span>
            <div className="w-8 h-8 rounded-xl bg-amber-500/10 text-amber-500 flex items-center justify-center">
              <Users className="w-4 h-4" />
            </div>
          </div>
          <div className="mt-3 text-2xl font-bold font-mono text-slate-900 dark:text-white">
            {totalActivations} <span className="text-sm font-normal text-slate-400">раз</span>
          </div>
          <p className="text-[11px] font-mono text-slate-500 mt-1">
            Оплачено с промокодом: {totalPaidUses}
          </p>
        </div>

        {/* Total Partner Commission Liability */}
        <div className="p-5 rounded-2xl bg-white/80 dark:bg-black/40 border border-slate-200 dark:border-white/10 shadow-sm backdrop-blur-md relative overflow-hidden">
          <div className="flex items-center justify-between">
            <span className="text-[11px] font-mono text-slate-500 uppercase tracking-wider">
              Комиссии амбассадорам
            </span>
            <div className="w-8 h-8 rounded-xl bg-indigo-500/10 text-indigo-500 flex items-center justify-center">
              <Coins className="w-4 h-4" />
            </div>
          </div>
          <div className="mt-3 text-2xl font-bold font-mono text-indigo-600 dark:text-indigo-400">
            {totalRewards.toLocaleString("ru-RU")} ₽
          </div>
          <p className="text-[11px] font-mono text-slate-500 mt-1">
            Суммарно начислено за привлечение
          </p>
        </div>

        {/* Active Promos Ratio */}
        <div className="p-5 rounded-2xl bg-white/80 dark:bg-black/40 border border-slate-200 dark:border-white/10 shadow-sm backdrop-blur-md relative overflow-hidden">
          <div className="flex items-center justify-between">
            <span className="text-[11px] font-mono text-slate-500 uppercase tracking-wider">
              Действующие коды
            </span>
            <div className="w-8 h-8 rounded-xl bg-sky-500/10 text-sky-500 flex items-center justify-center">
              <Tag className="w-4 h-4" />
            </div>
          </div>
          <div className="mt-3 text-2xl font-bold font-mono text-slate-900 dark:text-white">
            {activeCount} <span className="text-sm font-normal text-slate-400">/ {promos.length}</span>
          </div>
          <p className="text-[11px] font-mono text-slate-500 mt-1">
            {promos.length - activeCount} деактивировано
          </p>
        </div>
      </div>

      {/* 2. Top Bar: Search + Filter Tabs + Create Button */}
      <div className="flex flex-col sm:flex-row items-stretch sm:items-center justify-between gap-4 p-4 rounded-2xl bg-white/70 dark:bg-white/[0.03] border border-slate-200 dark:border-white/10 shadow-sm">
        <div className="flex flex-1 flex-col sm:flex-row items-stretch sm:items-center gap-3 max-w-xl">
          <div className="relative flex-1">
            <Search className="w-4 h-4 absolute left-3.5 top-1/2 -translate-y-1/2 text-slate-400" />
            <input
              type="text"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder="Поиск по коду или амбассадору..."
              className="w-full pl-10 pr-4 py-2 text-xs font-mono bg-slate-50 dark:bg-black/40 border border-slate-200 dark:border-white/10 rounded-xl text-slate-900 dark:text-white placeholder-slate-400 dark:placeholder-slate-500 focus:outline-none focus:border-amber-500"
            />
          </div>

          {/* Filter Pills */}
          <div className="flex items-center gap-1 bg-slate-100 dark:bg-black/40 p-1 rounded-xl border border-slate-200 dark:border-white/10 text-[11px] font-mono">
            <button
              onClick={() => setFilterActive("all")}
              className={`px-3 py-1 rounded-lg transition-all ${
                filterActive === "all"
                  ? "bg-white dark:bg-white/10 text-slate-900 dark:text-white shadow-sm font-bold"
                  : "text-slate-500 hover:text-slate-900 dark:hover:text-white"
              }`}
            >
              Все ({promos.length})
            </button>
            <button
              onClick={() => setFilterActive("active")}
              className={`px-3 py-1 rounded-lg transition-all ${
                filterActive === "active"
                  ? "bg-white dark:bg-white/10 text-emerald-600 dark:text-emerald-400 shadow-sm font-bold"
                  : "text-slate-500 hover:text-slate-900 dark:hover:text-white"
              }`}
            >
              Активные ({activeCount})
            </button>
            <button
              onClick={() => setFilterActive("inactive")}
              className={`px-3 py-1 rounded-lg transition-all ${
                filterActive === "inactive"
                  ? "bg-white dark:bg-white/10 text-slate-700 dark:text-slate-300 shadow-sm font-bold"
                  : "text-slate-500 hover:text-slate-900 dark:hover:text-white"
              }`}
            >
              Отключенные ({promos.length - activeCount})
            </button>
          </div>
        </div>

        <button
          onClick={openCreate}
          className="px-4 py-2 rounded-xl bg-amber-500 hover:bg-amber-400 text-black text-xs font-mono font-bold flex items-center justify-center gap-2 transition-all shadow-md active:scale-95"
        >
          <Plus className="w-4 h-4" />
          <span>Создать промокод</span>
        </button>
      </div>

      {/* 3. Detailed Promos Table */}
      <div className="overflow-x-auto rounded-2xl border border-slate-200 dark:border-white/10 bg-white dark:bg-black/30 shadow-sm">
        <table className="w-full text-left text-xs font-mono">
          <thead>
            <tr className="border-b border-slate-200 dark:border-white/10 bg-slate-50 dark:bg-white/[0.02] text-slate-600 dark:text-slate-400 font-semibold">
              <th className="py-3 px-4">Код скидки</th>
              <th className="py-3 px-4">Скидка студенту</th>
              <th className="py-3 px-4">Амбассадор / Партнёр</th>
              <th className="py-3 px-4">Оплат / активаций</th>
              <th className="py-3 px-4">Привлечённая выручка</th>
              <th className="py-3 px-4">Комиссия партнёру</th>
              <th className="py-3 px-4">Студенты</th>
              <th className="py-3 px-4">Статус</th>
              <th className="py-3 px-4 text-right">Действия</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-slate-100 dark:divide-white/5 text-slate-800 dark:text-slate-300">
            {filtered.length === 0 ? (
              <tr>
                <td colSpan={9} className="py-8 text-center text-slate-500 font-mono">
                  Промокоды не найдены
                </td>
              </tr>
            ) : (
              filtered.map((promo) => {
                const discountPct = promo.discount_percent || 5;
                const discountRub = Math.round(coursePricing.accelerator_price * (discountPct / 100));
                const discountedPrice = coursePricing.accelerator_price - discountRub;

                const promoRevenue = promo.total_revenue || 0;
                const paidUses = promo.paid_uses || 0;
                const partnerEarnings = paidUses * (promo.reward_amount || 0);
                const studentsCount = promo.activation_count ?? promo.students?.length ?? 0;

                return (
                  <tr key={promo.id} className="hover:bg-slate-50 dark:hover:bg-white/[0.02] transition-colors">
                    {/* Code */}
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

                    {/* Discount */}
                    <td className="py-3.5 px-4 font-bold text-emerald-600 dark:text-emerald-400">
                      <div>-{discountPct}% ({discountRub.toLocaleString("ru-RU")} ₽)</div>
                      <div className="text-[10px] text-slate-400 font-normal">курс: {discountedPrice.toLocaleString("ru-RU")} ₽</div>
                    </td>

                    {/* Owner */}
                    <td className="py-3.5 px-4">
                      {promo.owner_username ? (
                        <span className="text-sky-600 dark:text-sky-400 font-medium">@{promo.owner_username}</span>
                      ) : (
                        <span className="text-slate-400">—</span>
                      )}
                    </td>

                    {/* Paid uses vs activations */}
                    <td className="py-3.5 px-4">
                      <span className="px-2 py-0.5 rounded-full bg-slate-100 dark:bg-white/5 border border-slate-200 dark:border-white/10 font-bold text-slate-800 dark:text-white">
                        {paidUses}
                      </span>
                      <span className="block text-[10px] text-slate-400 mt-1">
                        активаций: {studentsCount}
                      </span>
                    </td>

                    {/* Generated Revenue */}
                    <td className="py-3.5 px-4 font-bold text-emerald-600 dark:text-emerald-400">
                      {promoRevenue.toLocaleString("ru-RU")} ₽
                    </td>

                    {/* Partner Commission */}
                    <td className="py-3.5 px-4">
                      <span className="text-amber-700 dark:text-amber-300 font-semibold">
                        {partnerEarnings.toLocaleString("ru-RU")} ₽
                      </span>
                      <span className="text-[10px] text-slate-400 block">
                        ({promo.reward_amount.toLocaleString("ru-RU")} ₽ / оплата)
                      </span>
                    </td>

                    {/* Students List Button */}
                    <td className="py-3.5 px-4">
                      {studentsCount > 0 ? (
                        <button
                          onClick={() => setStudentsModalPromo(promo)}
                          className="px-2 py-0.5 rounded bg-sky-500/10 hover:bg-sky-500/20 text-sky-600 dark:text-sky-400 border border-sky-500/20 font-bold text-[11px] transition-colors"
                        >
                          {studentsCount} чел. ↗
                        </button>
                      ) : studentsCount > 0 ? (
                        <span className="text-slate-400 text-[11px]">{studentsCount} чел.</span>
                      ) : (
                        <span className="text-slate-400 text-[11px]">—</span>
                      )}
                    </td>

                    {/* Status */}
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

                    {/* Action buttons */}
                    <td className="py-3.5 px-4 text-right">
                      <div className="flex items-center justify-end gap-1.5">
                        <button
                          onClick={() => handleToggle(promo.id)}
                          className={`px-2.5 py-1 rounded text-[11px] font-bold transition-all border ${
                            promo.is_active
                              ? "bg-rose-500/10 text-rose-600 dark:text-rose-300 border-rose-500/20 hover:bg-rose-500/20"
                              : "bg-emerald-500/10 text-emerald-600 dark:text-emerald-300 border-emerald-500/20 hover:bg-emerald-500/20"
                          }`}
                        >
                          {promo.is_active ? "Отключить" : "Включить"}
                        </button>
                        <button
                          onClick={() => openEdit(promo)}
                          className="px-2 py-1 rounded text-[11px] font-bold border border-slate-300 dark:border-white/10 text-slate-700 dark:text-slate-300 hover:bg-slate-100 dark:hover:bg-white/10"
                        >
                          Правка
                        </button>
                        <button
                          onClick={() => setDeleteTarget(promo)}
                          className="px-2 py-1 rounded text-[11px] font-bold border border-rose-500/30 text-rose-600 dark:text-rose-300 hover:bg-rose-500/10"
                        >
                          Удалить
                        </button>
                      </div>
                    </td>
                  </tr>
                );
              })
            )}
          </tbody>
        </table>
      </div>

      {/* 4. Modal: Students list who used promo */}
      {studentsModalPromo && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/60 backdrop-blur-sm animate-in fade-in">
          <div className="bg-white dark:bg-[#12161D] border border-slate-200 dark:border-white/20 rounded-2xl max-w-lg w-full p-6 space-y-4 shadow-2xl text-slate-900 dark:text-white">
            <div className="flex items-center justify-between border-b border-slate-200 dark:border-white/10 pb-3">
              <div className="flex items-center gap-2">
                <Users className="w-4 h-4 text-amber-500" />
                <h3 className="font-editorial text-lg font-bold text-slate-900 dark:text-white">
                  Студенты с промокодом {studentsModalPromo.code}
                </h3>
              </div>
              <button
                onClick={() => setStudentsModalPromo(null)}
                className="text-slate-400 hover:text-slate-700 dark:hover:text-white"
              >
                ✕
              </button>
            </div>

            <div className="text-xs font-mono text-slate-600 dark:text-slate-400 space-y-2">
              <div className="flex justify-between items-center p-3 rounded-xl bg-slate-50 dark:bg-white/5 border border-slate-200 dark:border-white/10">
                <span>Амбассадор:</span>
                <span className="font-bold text-sky-600 dark:text-sky-400">
                  {studentsModalPromo.owner_username ? `@${studentsModalPromo.owner_username}` : "—"}
                </span>
              </div>
              <div className="flex justify-between items-center p-3 rounded-xl bg-slate-50 dark:bg-white/5 border border-slate-200 dark:border-white/10">
                <span>Всего активаций:</span>
                <span className="font-bold text-slate-900 dark:text-white">
                  {studentsModalPromo.activation_count ?? studentsModalPromo.students?.length ?? 0} раз
                </span>
              </div>
              <div className="flex justify-between items-center p-3 rounded-xl bg-slate-50 dark:bg-white/5 border border-slate-200 dark:border-white/10">
                <span>Оплачено с промокодом:</span>
                <span className="font-bold text-slate-900 dark:text-white">
                  {studentsModalPromo.paid_uses ?? 0} раз
                </span>
              </div>
            </div>

            <div className="mt-4">
              <h4 className="text-xs font-mono font-semibold text-slate-500 dark:text-slate-400 mb-2 uppercase tracking-wider">
                Список студентов:
              </h4>
              <div className="max-h-60 overflow-y-auto space-y-1.5 border border-slate-200 dark:border-white/10 rounded-xl p-3 bg-slate-50/50 dark:bg-black/30">
                {studentsModalPromo.students && studentsModalPromo.students.length > 0 ? (
                  studentsModalPromo.students.map((student, idx) => (
                    <div
                      key={idx}
                      className="px-3 py-2 rounded-lg bg-white dark:bg-white/5 border border-slate-200/60 dark:border-white/5 text-xs font-mono flex items-center justify-between"
                    >
                      <span className="font-medium text-slate-900 dark:text-white">{student}</span>
                      <span className="text-[10px] text-emerald-600 dark:text-emerald-400 font-bold">
                        Оплачено со скидкой
                      </span>
                    </div>
                  ))
                ) : (
                  <p className="text-xs font-mono text-slate-400 text-center py-4">
                    Нет подробного списка (активации через внешние шлюзы или ручной ввод)
                  </p>
                )}
              </div>
            </div>

            <div className="flex justify-end pt-2">
              <button
                onClick={() => setStudentsModalPromo(null)}
                className="px-4 py-2 rounded-xl bg-slate-200 dark:bg-white/10 hover:bg-slate-300 dark:hover:bg-white/20 text-xs font-mono font-bold transition-colors"
              >
                Закрыть
              </button>
            </div>
          </div>
        </div>
      )}

      {/* 5. Modal: Create / Edit Promo Code */}
      {showCreateModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/60 backdrop-blur-sm animate-in fade-in">
          <div className="bg-white dark:bg-[#12161D] border border-slate-200 dark:border-white/20 rounded-2xl max-w-md w-full p-6 space-y-4 shadow-2xl text-slate-900 dark:text-white">
            <div className="flex items-center justify-between border-b border-slate-200 dark:border-white/10 pb-3">
              <div className="flex items-center gap-2">
                <Tag className="w-4 h-4 text-amber-500" />
                <h3 className="font-editorial text-lg font-bold text-slate-900 dark:text-white">
                  {editingId ? "Редактирование промокода" : "Новый промокод"}
                </h3>
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
                    Акселератор: {Math.round(coursePricing.accelerator_price * (1 - newDiscount / 100))} ₽
                  </span>
                </div>
                <div>
                  <label className="block text-slate-600 dark:text-slate-400 mb-1">Выплата партнеру (₽):</label>
                  <input
                    type="number"
                    min="0"
                    step="any"
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

      {/* 6. Modal: Delete Confirm */}
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
