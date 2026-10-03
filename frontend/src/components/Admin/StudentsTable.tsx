"use client";

import React, { useState } from "react";
import { Search, UserCheck, Lock, Plus, Pencil, Trash2 } from "lucide-react";
import { UserProfile, api } from "@/lib/api";

interface StudentsTableProps {
  students: UserProfile[];
  onToggleAccess: (userId: number, currentAccess: boolean) => void;
  onRefresh: () => void;
}

interface FormState {
  telegram_id: string;
  first_name: string;
  last_name: string;
  username: string;
  role: "student" | "admin";
  has_access: boolean;
}

const emptyForm: FormState = {
  telegram_id: "",
  first_name: "",
  last_name: "",
  username: "",
  role: "student",
  has_access: false,
};

const inputCls =
  "w-full px-3 py-2 text-xs font-mono bg-slate-50 dark:bg-black/40 border border-slate-200 dark:border-white/10 rounded-xl text-slate-900 dark:text-white placeholder-slate-400 focus:outline-none focus:border-amber-500";

export function StudentsTable({ students, onToggleAccess, onRefresh }: StudentsTableProps) {
  const [query, setQuery] = useState("");
  const [modal, setModal] = useState<"create" | "edit" | null>(null);
  const [editingId, setEditingId] = useState<number | null>(null);
  const [form, setForm] = useState<FormState>(emptyForm);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<UserProfile | null>(null);

  const filtered = students.filter((s) => {
    const q = query.toLowerCase();
    return (
      s.username?.toLowerCase().includes(q) ||
      s.first_name?.toLowerCase().includes(q) ||
      s.last_name?.toLowerCase().includes(q) ||
      String(s.telegram_id).includes(q)
    );
  });

  const openCreate = () => {
    setForm(emptyForm);
    setEditingId(null);
    setError(null);
    setModal("create");
  };

  const openEdit = (s: UserProfile) => {
    setForm({
      telegram_id: String(s.telegram_id),
      first_name: s.first_name || "",
      last_name: s.last_name || "",
      username: s.username || "",
      role: s.role,
      has_access: s.has_access,
    });
    setEditingId(s.id);
    setError(null);
    setModal("edit");
  };

  const handleSave = async (e: React.FormEvent) => {
    e.preventDefault();
    setSaving(true);
    setError(null);

    let res: { success: boolean; error?: string };
    if (modal === "create") {
      const tgId = Number(form.telegram_id);
      if (!tgId || !form.first_name.trim()) {
        setSaving(false);
        setError("Укажите Telegram ID и имя");
        return;
      }
      res = await api.createAdminUser({
        telegram_id: tgId,
        first_name: form.first_name.trim(),
        last_name: form.last_name.trim(),
        username: form.username.trim(),
        role: form.role,
        has_access: form.has_access,
      });
    } else {
      res = await api.updateAdminUser(editingId as number, {
        first_name: form.first_name.trim(),
        last_name: form.last_name.trim(),
        username: form.username.trim(),
        role: form.role,
        has_access: form.has_access,
      });
    }

    setSaving(false);
    if (res.success) {
      setModal(null);
      onRefresh();
    } else {
      setError(res.error || "Ошибка сохранения");
    }
  };

  const handleDelete = async () => {
    if (!deleteTarget) return;
    const res = await api.deleteAdminUser(deleteTarget.id);
    if (res.success) {
      setDeleteTarget(null);
      onRefresh();
    } else {
      setError(res.error || "Ошибка удаления");
      setDeleteTarget(null);
    }
  };

  return (
    <div className="p-6 sm:p-8 rounded-[32px] border border-slate-200 dark:border-white/15 bg-white dark:bg-black/30 shadow-sm mb-8">
      {/* Header and Search */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 mb-6">
        <div>
          <h3 className="text-xl font-bold text-slate-900 dark:text-white">
            Управление студентами и ручной эквайринг
          </h3>
          <p className="text-xs font-mono text-slate-500 dark:text-slate-400 mt-1">
            Создавайте, редактируйте и удаляйте студентов, переключайте доступ после оплаты
          </p>
        </div>

        <div className="flex flex-col sm:flex-row gap-3">
          <div className="relative w-full sm:w-72">
            <Search className="w-4 h-4 text-slate-400 absolute left-3.5 top-1/2 -translate-y-1/2" />
            <input
              type="text"
              placeholder="Поиск по @username, имени, ID..."
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              className={`${inputCls} pl-10 pr-4`}
            />
          </div>
          <button
            onClick={openCreate}
            className="px-4 py-2 rounded-xl bg-amber-500 hover:bg-amber-400 text-black text-xs font-mono font-bold flex items-center justify-center gap-2 transition-all shadow-md active:scale-95 whitespace-nowrap"
          >
            <Plus className="w-4 h-4" />
            <span>Добавить студента</span>
          </button>
        </div>
      </div>

      {error && !modal && (
        <div className="mb-4 p-3 bg-rose-500/10 border border-rose-500/30 text-rose-600 dark:text-rose-300 rounded-xl text-xs font-mono">
          {error}
        </div>
      )}

      {/* Table */}
      <div className="overflow-x-auto">
        <table className="w-full text-left text-xs font-mono">
          <thead>
            <tr className="border-b border-slate-200 dark:border-white/10 text-slate-500 dark:text-slate-400 uppercase tracking-wider">
              <th className="pb-3 font-semibold">Студент</th>
              <th className="pb-3 font-semibold">Telegram ID</th>
              <th className="pb-3 font-semibold">Роль</th>
              <th className="pb-3 font-semibold">Статус доступа</th>
              <th className="pb-3 font-semibold text-right">Действия</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-slate-100 dark:divide-white/5">
            {filtered.length === 0 && (
              <tr>
                <td colSpan={5} className="py-8 text-center text-slate-500">
                  Студенты не найдены
                </td>
              </tr>
            )}
            {filtered.map((s) => (
              <tr key={s.id} className="hover:bg-slate-50 dark:hover:bg-white/[0.02] transition-colors">
                <td className="py-4">
                  <div className="flex items-center gap-3">
                    <div className="w-8 h-8 rounded-full bg-slate-100 dark:bg-white/10 text-slate-900 dark:text-white font-bold flex items-center justify-center text-xs shrink-0">
                      {s.first_name ? s.first_name[0] : "U"}
                    </div>
                    <div>
                      <div className="text-slate-900 dark:text-white font-bold text-sm">
                        {s.first_name} {s.last_name}
                      </div>
                      <div className="text-slate-500 dark:text-slate-400">
                        @{s.username || "no_username"}
                      </div>
                    </div>
                  </div>
                </td>

                <td className="py-4 text-slate-600 dark:text-slate-300">{s.telegram_id}</td>

                <td className="py-4">
                  <span
                    className={`px-2 py-0.5 rounded-full text-[10px] font-bold ${
                      s.role === "admin"
                        ? "bg-amber-500/20 text-amber-700 dark:text-amber-300 border border-amber-500/30"
                        : "bg-slate-100 dark:bg-white/10 text-slate-600 dark:text-slate-300"
                    }`}
                  >
                    {s.role === "admin" ? "Администратор" : "Студент"}
                  </span>
                </td>

                <td className="py-4">
                  {s.has_access ? (
                    <span className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-[11px] font-semibold bg-emerald-500/15 text-emerald-600 dark:text-emerald-400 border border-emerald-500/25">
                      <UserCheck className="w-3.5 h-3.5" />
                      Активен
                    </span>
                  ) : (
                    <span className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-[11px] font-semibold bg-rose-500/15 text-rose-600 dark:text-rose-300 border border-rose-500/25">
                      <Lock className="w-3.5 h-3.5" />
                      Закрыт
                    </span>
                  )}
                </td>

                <td className="py-4">
                  <div className="flex items-center justify-end gap-2">
                    <button
                      onClick={() => onToggleAccess(s.id, s.has_access)}
                      className={`px-3 py-2 rounded-xl text-[11px] font-bold transition-all border ${
                        s.has_access
                          ? "bg-rose-500/10 text-rose-600 dark:text-rose-300 hover:bg-rose-500/20 border-rose-500/30"
                          : "bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 hover:bg-emerald-500/20 border-emerald-500/30"
                      }`}
                    >
                      {s.has_access ? "Отозвать доступ" : "Активировать доступ"}
                    </button>
                    <button
                      onClick={() => openEdit(s)}
                      title="Редактировать"
                      className="p-2 rounded-xl border border-slate-200 dark:border-white/10 text-slate-600 dark:text-slate-300 hover:bg-slate-100 dark:hover:bg-white/10 transition-all"
                    >
                      <Pencil className="w-3.5 h-3.5" />
                    </button>
                    <button
                      onClick={() => setDeleteTarget(s)}
                      title="Удалить"
                      className="p-2 rounded-xl border border-rose-500/30 text-rose-600 dark:text-rose-300 hover:bg-rose-500/10 transition-all"
                    >
                      <Trash2 className="w-3.5 h-3.5" />
                    </button>
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {/* Create / Edit modal */}
      {modal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/60 backdrop-blur-sm">
          <form
            onSubmit={handleSave}
            className="bg-white dark:bg-[#12161D] border border-slate-200 dark:border-white/20 rounded-2xl max-w-md w-full p-6 space-y-4 shadow-2xl text-slate-900 dark:text-white"
          >
            <div className="flex items-center justify-between border-b border-slate-200 dark:border-white/10 pb-3">
              <h3 className="text-lg font-bold">
                {modal === "create" ? "Новый студент" : "Редактирование студента"}
              </h3>
              <button
                type="button"
                onClick={() => setModal(null)}
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

            <div className="space-y-3">
              <label className="block text-[11px] font-mono text-slate-500">
                Telegram ID
                <input
                  type="number"
                  value={form.telegram_id}
                  onChange={(e) => setForm({ ...form, telegram_id: e.target.value })}
                  disabled={modal === "edit"}
                  className={`${inputCls} mt-1 disabled:opacity-50`}
                  placeholder="123456789"
                />
              </label>
              <div className="grid grid-cols-2 gap-3">
                <label className="block text-[11px] font-mono text-slate-500">
                  Имя
                  <input
                    value={form.first_name}
                    onChange={(e) => setForm({ ...form, first_name: e.target.value })}
                    className={`${inputCls} mt-1`}
                  />
                </label>
                <label className="block text-[11px] font-mono text-slate-500">
                  Фамилия
                  <input
                    value={form.last_name}
                    onChange={(e) => setForm({ ...form, last_name: e.target.value })}
                    className={`${inputCls} mt-1`}
                  />
                </label>
              </div>
              <label className="block text-[11px] font-mono text-slate-500">
                Username
                <input
                  value={form.username}
                  onChange={(e) => setForm({ ...form, username: e.target.value })}
                  className={`${inputCls} mt-1`}
                  placeholder="@username"
                />
              </label>
              <label className="block text-[11px] font-mono text-slate-500">
                Роль
                <select
                  value={form.role}
                  onChange={(e) => setForm({ ...form, role: e.target.value as "student" | "admin" })}
                  className={`${inputCls} mt-1`}
                >
                  <option value="student">Студент</option>
                  <option value="admin">Администратор</option>
                </select>
              </label>
              <label className="flex items-center gap-2 text-xs font-mono text-slate-700 dark:text-slate-300">
                <input
                  type="checkbox"
                  checked={form.has_access}
                  onChange={(e) => setForm({ ...form, has_access: e.target.checked })}
                />
                Доступ к закрытому каналу
              </label>
            </div>

            <div className="flex gap-3 pt-2">
              <button
                type="button"
                onClick={() => setModal(null)}
                className="flex-1 px-4 py-2 rounded-xl border border-slate-200 dark:border-white/10 text-xs font-mono"
              >
                Отмена
              </button>
              <button
                type="submit"
                disabled={saving}
                className="flex-1 px-4 py-2 rounded-xl bg-amber-500 hover:bg-amber-400 text-black text-xs font-mono font-bold disabled:opacity-60"
              >
                {saving ? "Сохранение..." : "Сохранить"}
              </button>
            </div>
          </form>
        </div>
      )}

      {/* Delete confirmation */}
      {deleteTarget && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/60 backdrop-blur-sm">
          <div className="bg-white dark:bg-[#12161D] border border-slate-200 dark:border-white/20 rounded-2xl max-w-sm w-full p-6 space-y-4 shadow-2xl text-slate-900 dark:text-white">
            <h3 className="text-lg font-bold">Удалить студента?</h3>
            <p className="text-xs font-mono text-slate-500 dark:text-slate-400">
              {deleteTarget.first_name} {deleteTarget.last_name} (@{deleteTarget.username || "no_username"}) будет удалён без возможности восстановления.
            </p>
            <div className="flex gap-3">
              <button
                onClick={() => setDeleteTarget(null)}
                className="flex-1 px-4 py-2 rounded-xl border border-slate-200 dark:border-white/10 text-xs font-mono"
              >
                Отмена
              </button>
              <button
                onClick={handleDelete}
                className="flex-1 px-4 py-2 rounded-xl bg-rose-500 hover:bg-rose-400 text-white text-xs font-mono font-bold"
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
