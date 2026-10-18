"use client";

import React, { useState } from "react";
import {
  Search,
  UserCheck,
  Lock,
  Plus,
  Pencil,
  Trash2,
  Tag,
  CreditCard,
  RotateCcw,
  Filter,
  ArrowUpDown,
  ArrowUp,
  ArrowDown,
  CheckCircle2,
  AlertCircle,
  Calendar,
  Receipt,
  ExternalLink,
  Send,
  FileCheck,
} from "lucide-react";
import { UserProfile, api } from "@/lib/api";

interface StudentsTableProps {
  students: UserProfile[];
  onToggleAccess: (userId: number, currentAccess: boolean, extra?: { tier?: "accelerator" | "vip"; paid_amount?: number; promo_code?: string }) => void;
  onRefresh: () => void;
}

interface FormState {
  telegram_id: string;
  first_name: string;
  last_name: string;
  username: string;
  role: "student" | "admin";
  has_access: boolean;
  tier: "accelerator" | "vip";
  paid_amount: string;
  promo_code: string;
  receipt_issued: boolean;
  receipt_url?: string;
  receipt_number?: string;
}

const emptyForm: FormState = {
  telegram_id: "",
  first_name: "",
  last_name: "",
  username: "",
  role: "student",
  has_access: false,
  tier: "accelerator",
  paid_amount: "6900",
  promo_code: "",
  receipt_issued: false,
  receipt_url: "",
  receipt_number: "",
};

function formatPurchaseDate(accessGrantedAt?: string, createdAt?: string): string {
  const dateStr = accessGrantedAt || createdAt;
  if (!dateStr) return "—";
  try {
    const d = new Date(dateStr);
    if (isNaN(d.getTime())) return "—";
    const day = String(d.getDate()).padStart(2, "0");
    const months = ["янв", "фев", "мар", "апр", "май", "июн", "июл", "авг", "сен", "окт", "ноя", "дек"];
    const month = months[d.getMonth()];
    const hours = String(d.getHours()).padStart(2, "0");
    const minutes = String(d.getMinutes()).padStart(2, "0");
    return `${day} ${month}, ${hours}:${minutes}`;
  } catch {
    return "—";
  }
}

const inputCls =
  "w-full px-3 py-2 text-xs font-mono bg-slate-50 dark:bg-black/40 border border-slate-200 dark:border-white/10 rounded-xl text-slate-900 dark:text-white placeholder-slate-400 focus:outline-none focus:border-amber-500";

const selectCls =
  "px-3 py-2 text-xs font-mono bg-slate-50 dark:bg-black/40 border border-slate-200 dark:border-white/10 rounded-xl text-slate-900 dark:text-white focus:outline-none focus:border-amber-500 cursor-pointer";

export function StudentsTable({ students, onToggleAccess, onRefresh }: StudentsTableProps) {
  const [query, setQuery] = useState("");
  const [tierFilter, setTierFilter] = useState<string>("all");
  const [accessFilter, setAccessFilter] = useState<string>("all");
  const [receiptFilter, setReceiptFilter] = useState<string>("all");
  const [receiptIssuedFilter, setReceiptIssuedFilter] = useState<string>("all");

  // Sorting state: purchase_date, paid_amount, name, username, tier, receipt_issued, created_at
  const [sortBy, setSortBy] = useState<string>("purchase_date");
  const [sortDir, setSortDir] = useState<"asc" | "desc">("desc");

  const [modal, setModal] = useState<"create" | "edit" | null>(null);
  const [editingId, setEditingId] = useState<number | null>(null);
  const [form, setForm] = useState<FormState>(emptyForm);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<UserProfile | null>(null);

  // Quick activate modal
  const [activateTarget, setActivateTarget] = useState<UserProfile | null>(null);
  const [activateTier, setActivateTier] = useState<"accelerator" | "vip">("accelerator");
  const [activatePromo, setActivatePromo] = useState("");
  const [activateAmount, setActivateAmount] = useState("6900");

  const [coursePricing, setCoursePricing] = useState({ accelerator_price: 6900, vip_price: 14900 });

  // Issue Receipt modal state
  const [isIssueModalOpen, setIsIssueModalOpen] = useState(false);
  const [issueReceiptTarget, setIssueReceiptTarget] = useState<UserProfile | null>(null);
  const [receiptStudentId, setReceiptStudentId] = useState<number | null>(null);
  const [receiptUrl, setReceiptUrl] = useState("");
  const [receiptNumber, setReceiptNumber] = useState("");
  const [receiptAmount, setReceiptAmount] = useState("");
  const [receiptServiceName, setReceiptServiceName] = useState("");
  const [receiptSendTelegram, setReceiptSendTelegram] = useState(true);
  const [issuingReceipt, setIssuingReceipt] = useState(false);
  const [receiptSuccessMsg, setReceiptSuccessMsg] = useState<string | null>(null);
  const [studentSearchTerm, setStudentSearchTerm] = useState("");

  const unissuedCount = (students || []).filter((s) => s.has_access && !s.receipt_issued).length;

  React.useEffect(() => {
    api.getCoursePricing().then((p) => {
      if (p && p.accelerator_price > 0) {
        setCoursePricing({ accelerator_price: p.accelerator_price, vip_price: p.vip_price });
        setActivateAmount(String(p.accelerator_price));
      }
    }).catch(() => {});
  }, []);

  const isFiltered =
    query.trim() !== "" ||
    tierFilter !== "all" ||
    accessFilter !== "all" ||
    receiptFilter !== "all" ||
    receiptIssuedFilter !== "all" ||
    sortBy !== "purchase_date" ||
    sortDir !== "desc";

  const [serverStudents, setServerStudents] = useState<UserProfile[] | null>(null);
  const [isSearching, setIsSearching] = useState(false);

  React.useEffect(() => {
    if (!isFiltered) {
      setServerStudents(null);
      return;
    }

    const timer = setTimeout(async () => {
      setIsSearching(true);
      try {
        const fetched = await api.getAdminStudents({
          q: query.trim(),
          tier: tierFilter,
          access: accessFilter,
          receipt_status: receiptFilter,
          receipt_issued: receiptIssuedFilter,
          sort_by: sortBy,
          sort_dir: sortDir,
        });
        setServerStudents(fetched);
      } catch {
        // Fallback to client-side filter
      } finally {
        setIsSearching(false);
      }
    }, 250);

    return () => clearTimeout(timer);
  }, [query, tierFilter, accessFilter, receiptFilter, receiptIssuedFilter, sortBy, sortDir, isFiltered]);

  const resetFilters = () => {
    setQuery("");
    setTierFilter("all");
    setAccessFilter("all");
    setReceiptFilter("all");
    setReceiptIssuedFilter("all");
    setSortBy("purchase_date");
    setSortDir("desc");
    setServerStudents(null);
  };

  const handleSort = (field: string) => {
    if (sortBy === field) {
      setSortDir((prev) => (prev === "asc" ? "desc" : "asc"));
    } else {
      setSortBy(field);
      setSortDir(field === "name" || field === "username" ? "asc" : "desc");
    }
  };

  const handleToggleReceiptIssued = async (student: UserProfile, e?: React.MouseEvent) => {
    if (e) e.stopPropagation();
    const nextVal = !student.receipt_issued;
    if (serverStudents) {
      setServerStudents(
        serverStudents.map((s) =>
          s.id === student.id
            ? { ...s, receipt_issued: nextVal, receipt_issued_at: nextVal ? new Date().toISOString() : undefined }
            : s
        )
      );
    }
    const res = await api.toggleReceiptIssued(student.id, nextVal);
    if (!res.success) {
      setError(res.error || "Не удалось обновить статус чека");
    }
    onRefresh();
  };

  const filtered = (students || []).filter((s) => {
    const q = query.toLowerCase();
    const matchesQuery =
      !q ||
      s.username?.toLowerCase().includes(q) ||
      s.first_name?.toLowerCase().includes(q) ||
      s.last_name?.toLowerCase().includes(q) ||
      String(s.telegram_id).includes(q) ||
      s.promo_code?.toLowerCase().includes(q);

    const matchesTier =
      tierFilter === "all" ||
      (s.tier && s.tier.toLowerCase() === tierFilter.toLowerCase());

    const matchesAccess =
      accessFilter === "all" ||
      (accessFilter === "active" ? s.has_access : !s.has_access);

    const matchesReceipt =
      receiptFilter === "all" ||
      (s.receipt_status && s.receipt_status.toLowerCase() === receiptFilter.toLowerCase()) ||
      (receiptFilter === "none" && (!s.receipt_status || s.receipt_status.toLowerCase() === "none"));

    const matchesReceiptIssued =
      receiptIssuedFilter === "all" ||
      (receiptIssuedFilter === "issued" ? !!s.receipt_issued : !s.receipt_issued);

    return matchesQuery && matchesTier && matchesAccess && matchesReceipt && matchesReceiptIssued;
  });

  // Client-side sort fallback
  filtered.sort((a, b) => {
    let cmp = 0;
    switch (sortBy) {
      case "purchase_date": {
        const dA = a.access_granted_at || a.created_at || "";
        const dB = b.access_granted_at || b.created_at || "";
        cmp = dA.localeCompare(dB);
        break;
      }
      case "paid_amount": {
        cmp = (a.paid_amount || 0) - (b.paid_amount || 0);
        break;
      }
      case "name": {
        const nA = `${a.first_name} ${a.last_name || ""}`.trim().toLowerCase();
        const nB = `${b.first_name} ${b.last_name || ""}`.trim().toLowerCase();
        cmp = nA.localeCompare(nB, "ru");
        break;
      }
      case "username": {
        const uA = (a.username || "").toLowerCase();
        const uB = (b.username || "").toLowerCase();
        cmp = uA.localeCompare(uB);
        break;
      }
      case "tier": {
        cmp = (a.tier || "").localeCompare(b.tier || "");
        break;
      }
      case "receipt_issued": {
        const rA = a.receipt_issued ? 1 : 0;
        const rB = b.receipt_issued ? 1 : 0;
        cmp = rA - rB;
        break;
      }
      case "created_at": {
        cmp = (a.created_at || "").localeCompare(b.created_at || "");
        break;
      }
      default: {
        const dA = a.access_granted_at || a.created_at || "";
        const dB = b.access_granted_at || b.created_at || "";
        cmp = dA.localeCompare(dB);
        break;
      }
    }
    return sortDir === "asc" ? cmp : -cmp;
  });

  const displayList = serverStudents !== null ? serverStudents : filtered;

  const calculateDefaultAmount = (tier: "accelerator" | "vip", promo: string): string => {
    const hasDiscount = promo.trim().length > 0;
    const base = tier === "vip" ? coursePricing.vip_price : coursePricing.accelerator_price;
    const finalPrice = hasDiscount ? Math.round(base * 0.95) : base;
    return String(finalPrice);
  };

  const handleTierChange = (newTier: "accelerator" | "vip") => {
    const newAmount = calculateDefaultAmount(newTier, form.promo_code);
    setForm({ ...form, tier: newTier, paid_amount: newAmount });
  };

  const handlePromoChange = (newPromo: string) => {
    const newAmount = calculateDefaultAmount(form.tier, newPromo);
    setForm({ ...form, promo_code: newPromo, paid_amount: newAmount });
  };

  const openCreate = () => {
    setForm(emptyForm);
    setEditingId(null);
    setError(null);
    setModal("create");
  };

  const openEdit = (s: UserProfile) => {
    const currentTier: "accelerator" | "vip" = s.tier === "vip" ? "vip" : "accelerator";
    const currentPromo = s.promo_code || "";
    const currentAmount = s.paid_amount ? String(s.paid_amount) : calculateDefaultAmount(currentTier, currentPromo);

    setForm({
      telegram_id: String(s.telegram_id),
      first_name: s.first_name || "",
      last_name: s.last_name || "",
      username: s.username || "",
      role: s.role,
      has_access: s.has_access,
      tier: currentTier,
      paid_amount: currentAmount,
      promo_code: currentPromo,
      receipt_issued: !!s.receipt_issued,
      receipt_url: s.receipt_url || "",
      receipt_number: s.receipt_number || "",
    });
    setEditingId(s.id);
    setError(null);
    setModal("edit");
  };

  const openIssueReceipt = (student?: UserProfile) => {
    setError(null);
    setReceiptSuccessMsg(null);
    setStudentSearchTerm("");
    if (student) {
      setIssueReceiptTarget(student);
      setReceiptStudentId(student.id);
      setReceiptUrl(student.receipt_url || "");
      setReceiptNumber(student.receipt_number || "");
      const defaultAmount = student.paid_amount && student.paid_amount > 0
        ? String(student.paid_amount)
        : student.tier === "vip"
        ? String(coursePricing.vip_price)
        : String(coursePricing.accelerator_price);
      setReceiptAmount(defaultAmount);
      const tierLabel = student.tier === "vip" ? "VIP" : student.tier === "basic" ? "Базовый" : "Акселератор";
      setReceiptServiceName(`Информационно-консультационные услуги по подготовке к поступлению в ШАД (Тариф «${tierLabel}»)`);
      setReceiptSendTelegram(true);
    } else {
      setIssueReceiptTarget(null);
      setReceiptStudentId(null);
      setReceiptUrl("");
      setReceiptNumber("");
      setReceiptAmount(String(coursePricing.accelerator_price));
      setReceiptServiceName("Информационно-консультационные услуги по подготовке к поступлению в ШАД (Тариф «Акселератор»)");
      setReceiptSendTelegram(true);
    }
    setIsIssueModalOpen(true);
  };

  const handleSelectStudentForReceipt = (student: UserProfile) => {
    setIssueReceiptTarget(student);
    setReceiptStudentId(student.id);
    setReceiptUrl(student.receipt_url || "");
    setReceiptNumber(student.receipt_number || "");
    const defaultAmount = student.paid_amount && student.paid_amount > 0
      ? String(student.paid_amount)
      : student.tier === "vip"
      ? String(coursePricing.vip_price)
      : String(coursePricing.accelerator_price);
    setReceiptAmount(defaultAmount);
    const tierLabel = student.tier === "vip" ? "VIP" : student.tier === "basic" ? "Базовый" : "Акселератор";
    setReceiptServiceName(`Информационно-консультационные услуги по подготовке к поступлению в ШАД (Тариф «${tierLabel}»)`);
  };

  const handleConfirmIssueReceipt = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!receiptStudentId) {
      setError("Пожалуйста, выберите ученика для выписки чека");
      return;
    }
    setIssuingReceipt(true);
    setError(null);
    setReceiptSuccessMsg(null);

    const amountNum = parseFloat(receiptAmount) || undefined;
    const cleanUrl = receiptUrl.trim();
    const cleanNumber = receiptNumber.trim();
    const cleanService = receiptServiceName.trim();

    const res = await api.issueReceipt(receiptStudentId, {
      receipt_url: cleanUrl,
      receipt_number: cleanNumber,
      amount: amountNum,
      service_name: cleanService,
      send_telegram: receiptSendTelegram,
    });

    setIssuingReceipt(false);
    if (!res.success) {
      setError(res.error || "Не удалось выписать чек");
      return;
    }

    setReceiptSuccessMsg(res.message || "Чек успешно сформирован и отправлен ученику в Telegram!");

    if (serverStudents) {
      setServerStudents((prev) =>
        prev
          ? prev.map((s) =>
              s.id === receiptStudentId
                ? {
                    ...s,
                    receipt_issued: true,
                    receipt_issued_at: new Date().toISOString(),
                    receipt_url: cleanUrl,
                    receipt_number: cleanNumber,
                    paid_amount: amountNum || s.paid_amount,
                  }
                : s
            )
          : null
      );
    }

    setTimeout(() => {
      setIsIssueModalOpen(false);
      onRefresh();
    }, 1500);
  };

  const handleSave = async (e: React.FormEvent) => {
    e.preventDefault();
    setSaving(true);
    setError(null);

    const paidNum = parseFloat(form.paid_amount) || 0;

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
        tier: form.tier,
        paid_amount: paidNum,
        promo_code: form.promo_code.trim().toUpperCase(),
        receipt_issued: form.receipt_issued,
      });
    } else {
      res = await api.updateAdminUser(editingId as number, {
        first_name: form.first_name.trim(),
        last_name: form.last_name.trim(),
        username: form.username.trim(),
        role: form.role,
        has_access: form.has_access,
        tier: form.tier,
        paid_amount: paidNum,
        promo_code: form.promo_code.trim().toUpperCase(),
        receipt_issued: form.receipt_issued,
        receipt_url: form.receipt_url?.trim(),
        receipt_number: form.receipt_number?.trim(),
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

  const handleOpenActivate = (student: UserProfile) => {
    const studentTier: "accelerator" | "vip" = student.tier === "vip" ? "vip" : "accelerator";
    const studentPromo = student.promo_code || "";
    setActivateTarget(student);
    setActivateTier(studentTier);
    setActivatePromo(studentPromo);
    setActivateAmount(student.paid_amount ? String(student.paid_amount) : calculateDefaultAmount(studentTier, studentPromo));
  };

  const handleConfirmActivate = () => {
    if (!activateTarget) return;
    const paidNum = parseFloat(activateAmount) || 0;
    onToggleAccess(activateTarget.id, false, {
      tier: activateTier,
      promo_code: activatePromo.trim().toUpperCase(),
      paid_amount: paidNum,
    });
    setActivateTarget(null);
  };

  const renderSortHeader = (label: string, field: string, alignRight: boolean = false) => {
    const isActive = sortBy === field;
    return (
      <th
        onClick={() => handleSort(field)}
        className={`pb-3 font-semibold cursor-pointer select-none group transition-colors hover:text-slate-900 dark:hover:text-white ${
          alignRight ? "text-right" : ""
        }`}
        title={`Сортировать по полю «${label}»`}
      >
        <div className={`inline-flex items-center gap-1.5 ${alignRight ? "justify-end" : ""}`}>
          <span>{label}</span>
          {isActive ? (
            sortDir === "asc" ? (
              <ArrowUp className="w-3.5 h-3.5 text-amber-500 shrink-0" />
            ) : (
              <ArrowDown className="w-3.5 h-3.5 text-amber-500 shrink-0" />
            )
          ) : (
            <ArrowUpDown className="w-3.5 h-3.5 opacity-30 group-hover:opacity-75 transition-opacity shrink-0" />
          )}
        </div>
      </th>
    );
  };

  return (
    <div className="p-6 sm:p-8 rounded-[32px] border border-slate-200 dark:border-white/15 bg-white dark:bg-black/30 shadow-sm mb-8">
      {/* Header and Search & Filters */}
      <div className="flex flex-col gap-4 mb-6">
        <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
          <div>
            <h3 className="text-xl font-bold text-slate-900 dark:text-white">
              Управление студентами и ручной эквайринг
            </h3>
            <p className="text-xs font-mono text-slate-500 dark:text-slate-400 mt-1">
              Сортировки по дате покупки и суммам • Чеки самозанятого «Мой налог» • Поиск и мгновенная активация
            </p>
          </div>

          <div className="flex items-center gap-2.5 self-start sm:self-auto flex-wrap">
            <button
              type="button"
              onClick={() => openIssueReceipt()}
              className="px-4 py-2 rounded-xl bg-emerald-500/15 hover:bg-emerald-500/25 border border-emerald-500/30 text-emerald-700 dark:text-emerald-400 text-xs font-mono font-bold flex items-center justify-center gap-2 transition-all shadow-sm active:scale-95 whitespace-nowrap"
              title="Выбрать ученика, указать ссылку на чек из «Мой налог» и отправить в Telegram-бота"
            >
              <Receipt className="w-4 h-4 text-emerald-500" />
              <span>Выписать чек</span>
              {unissuedCount > 0 && (
                <span className="px-1.5 py-0.2 rounded-full text-[10px] bg-amber-500 text-black font-bold" title={`Не выписано чеков: ${unissuedCount}`}>
                  {unissuedCount}
                </span>
              )}
            </button>

            <button
              onClick={openCreate}
              className="px-4 py-2 rounded-xl bg-amber-500 hover:bg-amber-400 text-black text-xs font-mono font-bold flex items-center justify-center gap-2 transition-all shadow-md active:scale-95 whitespace-nowrap"
            >
              <Plus className="w-4 h-4" />
              <span>Добавить студента</span>
            </button>
          </div>
        </div>

        {/* Search & Filters Bar */}
        <div className="flex flex-wrap items-center gap-3 pt-1">
          <div className="relative flex-1 min-w-[220px]">
            <Search className="w-4 h-4 text-slate-400 absolute left-3.5 top-1/2 -translate-y-1/2" />
            <input
              type="text"
              placeholder="Поиск по @username, имени, TG ID, промокоду..."
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              className={`${inputCls} pl-10 pr-4`}
            />
          </div>

          {/* Sort Dropdown */}
          <select
            value={`${sortBy}:${sortDir}`}
            onChange={(e) => {
              const [field, dir] = e.target.value.split(":");
              setSortBy(field);
              setSortDir(dir as "asc" | "desc");
            }}
            className={selectCls}
            title="Сортировка списка"
          >
            <option value="purchase_date:desc">📅 Дата покупки (новые)</option>
            <option value="purchase_date:asc">📅 Дата покупки (старые)</option>
            <option value="paid_amount:desc">💰 Оплата (убывание)</option>
            <option value="paid_amount:asc">💰 Оплата (возрастание)</option>
            <option value="name:asc">🔤 Имя (А → Я)</option>
            <option value="name:desc">🔤 Имя (Я → А)</option>
            <option value="username:asc">@ Username (А → Я)</option>
            <option value="tier:asc">🏷️ Тариф (Акселератор → VIP)</option>
            <option value="tier:desc">🏷️ Тариф (VIP → Акселератор)</option>
            <option value="receipt_issued:desc">🧾 Чек СЗ (выписан)</option>
            <option value="receipt_issued:asc">🧾 Чек СЗ (не выписан)</option>
            <option value="created_at:desc">🕒 Регистрация (новые)</option>
            <option value="created_at:asc">🕒 Регистрация (старые)</option>
          </select>

          {/* Tier Filter */}
          <select
            value={tierFilter}
            onChange={(e) => setTierFilter(e.target.value)}
            className={selectCls}
            title="Фильтр по тарифу"
          >
            <option value="all">Все тарифы</option>
            <option value="accelerator">Акселератор ({coursePricing.accelerator_price.toLocaleString("ru-RU")} ₽)</option>
            <option value="vip">★ VIP ({coursePricing.vip_price.toLocaleString("ru-RU")} ₽)</option>
          </select>

          {/* Access Filter */}
          <select
            value={accessFilter}
            onChange={(e) => setAccessFilter(e.target.value)}
            className={selectCls}
            title="Фильтр по доступу"
          >
            <option value="all">Все доступы</option>
            <option value="active">Доступ активен</option>
            <option value="pending">Доступ закрыт</option>
          </select>

          {/* Payment Receipt Status Filter */}
          <select
            value={receiptFilter}
            onChange={(e) => setReceiptFilter(e.target.value)}
            className={selectCls}
            title="Фильтр по статусу чека перевода"
          >
            <option value="all">Чеки перевода: Все</option>
            <option value="pending">⏳ На проверке</option>
            <option value="approved">✅ Одобрен</option>
            <option value="rejected">❌ Отклонён</option>
            <option value="none">— Без чека</option>
          </select>

          {/* Self-Employed Receipt Filter */}
          <select
            value={receiptIssuedFilter}
            onChange={(e) => setReceiptIssuedFilter(e.target.value)}
            className={selectCls}
            title="Фильтр по чеку самозанятого (Мой налог)"
          >
            <option value="all">Чек СЗ: Все</option>
            <option value="issued">✅ Чек выписан</option>
            <option value="not_issued">⚠️ Чек не выписан</option>
          </select>

          {/* Reset Filters */}
          {isFiltered && (
            <button
              onClick={resetFilters}
              className="px-3 py-2 rounded-xl bg-slate-100 hover:bg-slate-200 dark:bg-white/10 dark:hover:bg-white/15 text-slate-700 dark:text-slate-300 text-xs font-mono flex items-center gap-1.5 transition-all"
              title="Сбросить все фильтры и сортировку"
            >
              <RotateCcw className="w-3.5 h-3.5" />
              <span>Сбросить</span>
            </button>
          )}
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
              {renderSortHeader("Студент", "name")}
              <th className="pb-3 font-semibold">Telegram ID</th>
              {renderSortHeader("Тариф", "tier")}
              {renderSortHeader("Оплачено", "paid_amount")}
              {renderSortHeader("Дата покупки", "purchase_date")}
              <th className="pb-3 font-semibold">Чек перевода</th>
              {renderSortHeader("Чек самозанятого", "receipt_issued")}
              <th className="pb-3 font-semibold">Промокод</th>
              <th className="pb-3 font-semibold">Статус</th>
              <th className="pb-3 font-semibold text-right">Действия</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-slate-100 dark:divide-white/5">
            {displayList.length === 0 && (
              <tr>
                <td colSpan={10} className="py-8 text-center text-slate-500">
                  {isFiltered ? "По заданным фильтрам студенты не найдены" : "Студенты не найдены"}
                </td>
              </tr>
            )}
            {displayList.map((s) => {
              const isVIP = s.tier === "vip";
              const isBasic = s.tier === "basic";
              const defaultTierPrice = isVIP
                ? coursePricing.vip_price
                : isBasic
                ? 3990
                : coursePricing.accelerator_price;
              const displayAmount = s.paid_amount
                ? s.paid_amount.toLocaleString("ru-RU") + " ₽"
                : s.has_access
                ? (defaultTierPrice.toLocaleString("ru-RU") + " ₽")
                : "0 ₽";

              return (
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

                  {/* Tier Badge & Upsells */}
                  <td className="py-4">
                    <div className="flex flex-col gap-1 items-start">
                      <span
                        className={`px-2.5 py-1 rounded-full text-[10px] font-bold uppercase tracking-wider inline-flex items-center gap-1 ${
                          isVIP
                            ? "bg-amber-500/15 text-amber-600 dark:text-amber-400 border border-amber-500/30"
                            : isBasic
                            ? "bg-slate-500/15 text-slate-700 dark:text-slate-300 border border-slate-500/30"
                            : "bg-emerald-500/15 text-emerald-700 dark:text-emerald-400 border border-emerald-500/30"
                        }`}
                      >
                        {isVIP ? "★ VIP" : isBasic ? "Базовый" : "Акселератор"}
                      </span>
                      {s.has_audit_upsell && (
                        <span className="px-2 py-0.5 rounded-md text-[9px] font-bold uppercase tracking-tight bg-indigo-500/15 text-indigo-700 dark:text-indigo-400 border border-indigo-500/30">
                          +Аудит эссе (1 900 ₽)
                        </span>
                      )}
                    </div>
                  </td>

                  {/* Paid Amount */}
                  <td className="py-4 font-bold text-slate-900 dark:text-white">
                    {displayAmount}
                  </td>

                  {/* Purchase Date */}
                  <td className="py-4 text-slate-600 dark:text-slate-300 text-[11px] whitespace-nowrap">
                    {formatPurchaseDate(s.access_granted_at, s.created_at)}
                  </td>

                  {/* Transfer Receipt Status Badge */}
                  <td className="py-4">
                    {s.receipt_status === "approved" ? (
                      <span className="inline-flex items-center gap-1 px-2.5 py-1 rounded-full text-[10px] font-semibold bg-emerald-500/15 text-emerald-700 dark:text-emerald-400 border border-emerald-500/25">
                        ✅ Одобрен
                      </span>
                    ) : s.receipt_status === "pending" ? (
                      <span className="inline-flex items-center gap-1 px-2.5 py-1 rounded-full text-[10px] font-semibold bg-amber-500/15 text-amber-600 dark:text-amber-400 border border-amber-500/25 animate-pulse">
                        ⏳ Проверка
                      </span>
                    ) : s.receipt_status === "rejected" ? (
                      <span className="inline-flex items-center gap-1 px-2.5 py-1 rounded-full text-[10px] font-semibold bg-rose-500/15 text-rose-600 dark:text-rose-400 border border-rose-500/25">
                        ❌ Отклонён
                      </span>
                    ) : (
                      <span className="text-slate-400 text-xs">—</span>
                    )}
                  </td>

                  {/* Self-Employed Fiscal Receipt (Мой налог) */}
                  <td className="py-4">
                    <div className="flex flex-col gap-1 items-start">
                      {s.receipt_issued ? (
                        <div className="flex items-center gap-1.5 flex-wrap">
                          <button
                            type="button"
                            onClick={(e) => handleToggleReceiptIssued(s, e)}
                            title={
                              s.receipt_issued_at
                                ? `Чек выписан (${formatPurchaseDate(s.receipt_issued_at)}). Нажмите для отмены`
                                : "Чек выписан в «Мой налог». Нажмите для отмены"
                            }
                            className="inline-flex items-center gap-1 px-2.5 py-1 rounded-full text-[10px] font-bold bg-emerald-500/15 text-emerald-700 dark:text-emerald-400 border border-emerald-500/30 hover:bg-emerald-500/25 active:scale-95 transition-all shadow-sm"
                          >
                            <CheckCircle2 className="w-3.5 h-3.5 text-emerald-500" />
                            <span>Выписан</span>
                          </button>
                          {s.receipt_url && (
                            <a
                              href={s.receipt_url}
                              target="_blank"
                              rel="noreferrer"
                              title="Открыть официальный чек в новой вкладке"
                              className="inline-flex items-center gap-1 px-2 py-0.5 rounded-md text-[10px] font-mono bg-slate-100 dark:bg-white/10 hover:bg-slate-200 dark:hover:bg-white/20 text-slate-700 dark:text-slate-300 border border-slate-200 dark:border-white/10 transition-colors"
                            >
                              <ExternalLink className="w-3 h-3 text-emerald-500" />
                              <span>Чек</span>
                            </a>
                          )}
                          <button
                            type="button"
                            onClick={(e) => {
                              e.stopPropagation();
                              openIssueReceipt(s);
                            }}
                            title="Редактировать реквизиты или переотправить чек через Telegram-бота"
                            className="p-1 rounded-md text-slate-400 hover:text-amber-500 hover:bg-slate-100 dark:hover:bg-white/10 transition-colors"
                          >
                            <Pencil className="w-3 h-3" />
                          </button>
                        </div>
                      ) : (
                        <div className="flex items-center gap-1.5">
                          <button
                            type="button"
                            onClick={(e) => {
                              e.stopPropagation();
                              openIssueReceipt(s);
                            }}
                            title="Выписать официальный чек и отправить в Telegram-бота"
                            className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-[10px] font-bold bg-amber-500/15 text-amber-700 dark:text-amber-400 border border-amber-500/30 hover:bg-amber-500/25 active:scale-95 transition-all shadow-sm group"
                          >
                            <Receipt className="w-3.5 h-3.5 text-amber-500 group-hover:scale-110 transition-transform" />
                            <span>Выписать чек</span>
                          </button>
                        </div>
                      )}
                    </div>
                  </td>

                  {/* Promo Code */}
                  <td className="py-4">
                    {s.promo_code ? (
                      <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded bg-amber-500/10 border border-amber-500/20 text-amber-700 dark:text-amber-400 font-bold text-[11px]">
                        <Tag className="w-3 h-3" />
                        <span>{s.promo_code}</span>
                      </span>
                    ) : (
                      <span className="text-slate-400">—</span>
                    )}
                  </td>

                  {/* Status Badge */}
                  <td className="py-4">
                    {s.has_access ? (
                      <span className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-[11px] font-semibold bg-emerald-500/15 text-emerald-700 dark:text-emerald-400 border border-emerald-500/25">
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

                  {/* Actions */}
                  <td className="py-4">
                    <div className="flex items-center justify-end gap-2">
                      {s.has_access ? (
                        <button
                          onClick={() => onToggleAccess(s.id, true)}
                          className="px-3 py-1.5 rounded-xl text-[11px] font-bold transition-all border bg-rose-500/10 text-rose-600 dark:text-rose-300 hover:bg-rose-500/20 border-rose-500/30"
                        >
                          Отозвать
                        </button>
                      ) : (
                        <button
                          onClick={() => handleOpenActivate(s)}
                          className="px-3 py-1.5 rounded-xl text-[11px] font-bold transition-all border bg-emerald-500/10 text-emerald-700 dark:text-emerald-400 hover:bg-emerald-500/20 border-emerald-500/30"
                        >
                          Активировать
                        </button>
                      )}
                      <button
                        onClick={() => openEdit(s)}
                        title="Редактировать"
                        className="p-1.5 rounded-xl border border-slate-200 dark:border-white/10 text-slate-600 dark:text-slate-300 hover:bg-slate-100 dark:hover:bg-white/10 transition-all"
                      >
                        <Pencil className="w-3.5 h-3.5" />
                      </button>
                      <button
                        onClick={() => setDeleteTarget(s)}
                        title="Удалить"
                        className="p-1.5 rounded-xl border border-rose-500/30 text-rose-600 dark:text-rose-300 hover:bg-rose-500/10 transition-all"
                      >
                        <Trash2 className="w-3.5 h-3.5" />
                      </button>
                    </div>
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>

      {/* Quick Activate Modal */}
      {activateTarget && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/60 backdrop-blur-sm">
          <div className="bg-white dark:bg-[#12161D] border border-slate-200 dark:border-white/20 rounded-2xl max-w-sm w-full p-6 space-y-4 shadow-2xl text-slate-900 dark:text-white">
            <h3 className="text-lg font-bold">Активировать доступ студенту</h3>
            <p className="text-xs font-mono text-slate-500 dark:text-slate-400">
              {activateTarget.first_name} {activateTarget.last_name} (@{activateTarget.username || activateTarget.telegram_id})
            </p>

            <div className="space-y-3">
              <label className="block text-[11px] font-mono text-slate-500">
                Тариф
                <select
                  value={activateTier}
                  onChange={(e) => {
                    const t = e.target.value as "accelerator" | "vip";
                    setActivateTier(t);
                    setActivateAmount(calculateDefaultAmount(t, activatePromo));
                  }}
                  className={`${inputCls} mt-1 font-semibold`}
                >
                  <option value="accelerator">Акселератор ({coursePricing.accelerator_price.toLocaleString("ru-RU")} ₽)</option>
                  <option value="vip">★ VIP-сопровождение ({coursePricing.vip_price.toLocaleString("ru-RU")} ₽)</option>
                </select>
              </label>

              <label className="block text-[11px] font-mono text-slate-500">
                Промокод (если применялся)
                <input
                  value={activatePromo}
                  onChange={(e) => {
                    const p = e.target.value.toUpperCase();
                    setActivatePromo(p);
                    setActivateAmount(calculateDefaultAmount(activateTier, p));
                  }}
                  className={`${inputCls} mt-1`}
                  placeholder="SPARK26"
                />
              </label>

              <label className="block text-[11px] font-mono text-slate-500">
                Фактическая сумма оплаты (₽)
                <input
                  type="number"
                  value={activateAmount}
                  onChange={(e) => setActivateAmount(e.target.value)}
                  className={`${inputCls} mt-1 font-bold`}
                  placeholder={coursePricing.accelerator_price.toString()}
                />
              </label>
            </div>

            <div className="flex gap-3 pt-2">
              <button
                type="button"
                onClick={() => setActivateTarget(null)}
                className="flex-1 px-4 py-2 rounded-xl border border-slate-200 dark:border-white/10 text-xs font-mono"
              >
                Отмена
              </button>
              <button
                type="button"
                onClick={handleConfirmActivate}
                className="flex-1 px-4 py-2 rounded-xl bg-emerald-600 hover:bg-emerald-500 text-white text-xs font-mono font-bold shadow-md"
              >
                Выдать доступ
              </button>
            </div>
          </div>
        </div>
      )}

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

              <div className="grid grid-cols-2 gap-3">
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
              </div>

              {/* Tier Selection */}
              <div className="grid grid-cols-2 gap-3">
                <label className="block text-[11px] font-mono text-slate-500">
                  Тариф
                  <select
                    value={form.tier}
                    onChange={(e) => handleTierChange(e.target.value as "accelerator" | "vip")}
                    className={`${inputCls} mt-1 font-semibold`}
                  >
                    <option value="accelerator">Акселератор ({coursePricing.accelerator_price.toLocaleString("ru-RU")} ₽)</option>
                    <option value="vip">★ VIP ({coursePricing.vip_price.toLocaleString("ru-RU")} ₽)</option>
                  </select>
                </label>

                <label className="block text-[11px] font-mono text-slate-500">
                  Промокод
                  <input
                    value={form.promo_code}
                    onChange={(e) => handlePromoChange(e.target.value.toUpperCase())}
                    className={`${inputCls} mt-1`}
                    placeholder="SPARK26"
                  />
                </label>
              </div>

              {/* Paid Amount Input */}
              <label className="block text-[11px] font-mono text-slate-500">
                Сумма оплаты (₽)
                <input
                  type="number"
                  value={form.paid_amount}
                  onChange={(e) => setForm({ ...form, paid_amount: e.target.value })}
                  className={`${inputCls} mt-1 font-bold`}
                  placeholder={coursePricing.accelerator_price.toString()}
                />
              </label>

              <label className="flex items-center gap-2 text-xs font-mono text-slate-700 dark:text-slate-300 pt-1">
                <input
                  type="checkbox"
                  checked={form.has_access}
                  onChange={(e) => setForm({ ...form, has_access: e.target.checked })}
                />
                Доступ к закрытому каналу и материалам
              </label>

              <label className="flex items-center gap-2 text-xs font-mono text-slate-700 dark:text-slate-300">
                <input
                  type="checkbox"
                  checked={form.receipt_issued}
                  onChange={(e) => setForm({ ...form, receipt_issued: e.target.checked })}
                />
                Чек самозанятого выписан в «Мой налог» (ФНС)
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

      {/* Issue Receipt Modal */}
      {isIssueModalOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/60 backdrop-blur-sm">
          <form
            onSubmit={handleConfirmIssueReceipt}
            className="bg-white dark:bg-[#12161D] border border-slate-200 dark:border-white/20 rounded-3xl max-w-lg w-full p-6 sm:p-7 space-y-4 shadow-2xl text-slate-900 dark:text-white max-h-[90vh] overflow-y-auto"
          >
            {/* Modal Header */}
            <div className="flex items-center justify-between border-b border-slate-200 dark:border-white/10 pb-4">
              <div className="flex items-center gap-3">
                <div className="p-2.5 rounded-2xl bg-emerald-500/15 border border-emerald-500/30 text-emerald-500">
                  <Receipt className="w-5 h-5" />
                </div>
                <div>
                  <h3 className="text-base font-bold text-slate-900 dark:text-white">
                    Выписка электронного чека
                  </h3>
                  <p className="text-[11px] font-mono text-slate-500 dark:text-slate-400">
                    Регистрация чека самозанятого и доставка ученику в Telegram
                  </p>
                </div>
              </div>
              <button
                type="button"
                onClick={() => setIsIssueModalOpen(false)}
                className="p-1 rounded-lg text-slate-400 hover:text-slate-700 dark:hover:text-white hover:bg-slate-100 dark:hover:bg-white/10 transition-colors"
              >
                ✕
              </button>
            </div>

            {error && (
              <div className="p-3 bg-rose-500/10 border border-rose-500/30 text-rose-600 dark:text-rose-300 rounded-xl text-xs font-mono flex items-center gap-2">
                <AlertCircle className="w-4 h-4 shrink-0 text-rose-500" />
                <span>{error}</span>
              </div>
            )}

            {receiptSuccessMsg && (
              <div className="p-3 bg-emerald-500/10 border border-emerald-500/30 text-emerald-600 dark:text-emerald-300 rounded-xl text-xs font-mono flex items-center gap-2">
                <CheckCircle2 className="w-4 h-4 shrink-0 text-emerald-500" />
                <span>{receiptSuccessMsg}</span>
              </div>
            )}

            {/* Student Selection (if not pre-selected) */}
            {!issueReceiptTarget ? (
              <div className="space-y-2">
                <label className="block text-[11px] font-mono font-semibold text-slate-600 dark:text-slate-300">
                  1. Выберите ученика для оформления чека:
                </label>
                <div className="relative">
                  <Search className="w-4 h-4 text-slate-400 absolute left-3 top-1/2 -translate-y-1/2" />
                  <input
                    type="text"
                    placeholder="Поиск по имени, @username или Telegram ID..."
                    value={studentSearchTerm}
                    onChange={(e) => setStudentSearchTerm(e.target.value)}
                    className={`${inputCls} pl-9`}
                    autoFocus
                  />
                </div>
                <div className="max-h-52 overflow-y-auto space-y-1.5 border border-slate-200 dark:border-white/10 rounded-2xl p-2 bg-slate-50/50 dark:bg-black/20">
                  {(students || [])
                    .filter((s) => {
                      if (!studentSearchTerm.trim()) return true;
                      const term = studentSearchTerm.toLowerCase();
                      return (
                        s.first_name?.toLowerCase().includes(term) ||
                        s.last_name?.toLowerCase().includes(term) ||
                        s.username?.toLowerCase().includes(term) ||
                        String(s.telegram_id).includes(term)
                      );
                    })
                    .slice(0, 20)
                    .map((s) => (
                      <button
                        key={s.id}
                        type="button"
                        onClick={() => handleSelectStudentForReceipt(s)}
                        className="w-full text-left p-2.5 rounded-xl hover:bg-slate-200/60 dark:hover:bg-white/10 transition-colors flex items-center justify-between gap-3 text-xs font-mono border border-transparent hover:border-slate-300 dark:hover:border-white/15 group"
                      >
                        <div className="flex items-center gap-2.5 min-w-0">
                          <div className="w-7 h-7 rounded-full bg-amber-500/20 text-amber-600 dark:text-amber-400 flex items-center justify-center font-bold text-xs shrink-0 group-hover:scale-105 transition-transform">
                            {s.first_name ? s.first_name[0] : "?"}
                          </div>
                          <div className="truncate">
                            <span className="font-bold text-slate-900 dark:text-white">
                              {s.first_name} {s.last_name}
                            </span>
                            <span className="text-slate-400 ml-1.5">
                              {s.username ? `@${s.username}` : `TG: ${s.telegram_id}`}
                            </span>
                          </div>
                        </div>
                        <div className="flex items-center gap-2 shrink-0">
                          <span className={`text-[10px] px-2 py-0.5 rounded-full font-bold ${
                            s.receipt_issued
                              ? "bg-emerald-500/15 text-emerald-600 dark:text-emerald-400"
                              : "bg-amber-500/15 text-amber-700 dark:text-amber-400"
                          }`}>
                            {s.receipt_issued ? "Чек выписан" : "Без чека"}
                          </span>
                          <span className="font-bold text-slate-800 dark:text-slate-200">
                            {s.paid_amount ? `${s.paid_amount.toLocaleString("ru-RU")} ₽` : "—"}
                          </span>
                        </div>
                      </button>
                    ))}
                </div>
              </div>
            ) : (
              /* Selected Student Card */
              <div className="p-4 rounded-2xl border border-slate-200 dark:border-white/10 bg-slate-50 dark:bg-black/40 flex items-center justify-between gap-3 shadow-inner">
                <div className="flex items-center gap-3 min-w-0">
                  <div className="w-10 h-10 rounded-2xl bg-amber-500/20 text-amber-600 dark:text-amber-400 flex items-center justify-center font-bold text-sm shrink-0 border border-amber-500/30">
                    {issueReceiptTarget.first_name ? issueReceiptTarget.first_name[0] : "?"}
                  </div>
                  <div className="truncate">
                    <div className="font-bold text-xs text-slate-900 dark:text-white flex items-center gap-2">
                      <span>{issueReceiptTarget.first_name} {issueReceiptTarget.last_name}</span>
                      {issueReceiptTarget.receipt_issued && (
                        <span className="inline-flex items-center gap-0.5 px-2 py-0.2 rounded-full text-[9px] font-bold bg-emerald-500/15 text-emerald-600 dark:text-emerald-400">
                          <CheckCircle2 className="w-2.5 h-2.5" /> Выписан
                        </span>
                      )}
                    </div>
                    <div className="text-[11px] font-mono text-slate-400 flex items-center gap-2 mt-0.5">
                      <span>{issueReceiptTarget.username ? `@${issueReceiptTarget.username}` : `ID: ${issueReceiptTarget.telegram_id}`}</span>
                      <span>•</span>
                      <span className="text-amber-600 dark:text-amber-400 font-semibold">
                        {issueReceiptTarget.tier === "vip" ? "Тариф VIP" : issueReceiptTarget.tier === "basic" ? "Тариф Базовый" : "Тариф Акселератор"}
                      </span>
                    </div>
                  </div>
                </div>

                <button
                  type="button"
                  onClick={() => {
                    setIssueReceiptTarget(null);
                    setReceiptStudentId(null);
                  }}
                  className="px-3 py-1.5 text-[11px] font-mono rounded-xl border border-slate-200 dark:border-white/10 hover:bg-slate-200/60 dark:hover:bg-white/10 text-slate-600 dark:text-slate-300 transition-colors shrink-0"
                >
                  Сменить
                </button>
              </div>
            )}

            {/* Receipt Details Form */}
            {issueReceiptTarget && (
              <div className="space-y-3.5 pt-1">
                <label className="block text-[11px] font-mono text-slate-500 dark:text-slate-400">
                  Ссылка на чек в ФНС / «Мой налог» / ОФД:
                  <div className="relative mt-1">
                    <input
                      type="url"
                      value={receiptUrl}
                      onChange={(e) => setReceiptUrl(e.target.value)}
                      placeholder="https://lknpd.nalog.ru/api/v1/receipt/.../print"
                      className={`${inputCls} pr-8`}
                    />
                    {receiptUrl && (
                      <a
                        href={receiptUrl}
                        target="_blank"
                        rel="noreferrer"
                        className="absolute right-2.5 top-1/2 -translate-y-1/2 text-slate-400 hover:text-emerald-500 transition-colors"
                        title="Проверить ссылку"
                      >
                        <ExternalLink className="w-3.5 h-3.5" />
                      </a>
                    )}
                  </div>
                  <span className="text-[10px] text-slate-400 mt-1 block">
                    Скопируйте ссылку в приложении «Мой налог» (раздел «Чеки» ➔ открыть чек ➔ «Поделиться / Ссылка»)
                  </span>
                </label>

                <div className="grid grid-cols-2 gap-3">
                  <label className="block text-[11px] font-mono text-slate-500 dark:text-slate-400">
                    Номер чека (необязательно):
                    <input
                      type="text"
                      value={receiptNumber}
                      onChange={(e) => setReceiptNumber(e.target.value)}
                      placeholder="Например: 2026-10-18/001"
                      className={`${inputCls} mt-1`}
                    />
                  </label>

                  <label className="block text-[11px] font-mono text-slate-500 dark:text-slate-400">
                    Сумма в чеке (₽):
                    <input
                      type="number"
                      value={receiptAmount}
                      onChange={(e) => setReceiptAmount(e.target.value)}
                      placeholder="6900"
                      className={`${inputCls} mt-1 font-bold text-amber-600 dark:text-amber-400`}
                    />
                  </label>
                </div>

                <label className="block text-[11px] font-mono text-slate-500 dark:text-slate-400">
                  Наименование услуги в чеке:
                  <input
                    type="text"
                    value={receiptServiceName}
                    onChange={(e) => setReceiptServiceName(e.target.value)}
                    placeholder="Информационно-консультационные услуги по подготовке к поступлению в ШАД"
                    className={`${inputCls} mt-1`}
                  />
                </label>

                <label className="flex items-start gap-2.5 text-xs font-mono text-slate-700 dark:text-slate-300 pt-1 cursor-pointer select-none">
                  <input
                    type="checkbox"
                    checked={receiptSendTelegram}
                    onChange={(e) => setReceiptSendTelegram(e.target.checked)}
                    className="mt-0.5 rounded accent-emerald-500"
                  />
                  <div>
                    <span className="font-bold">Отправить чек ученику через Telegram-бота</span>
                    <p className="text-[10px] text-slate-400">
                      Бот моментально доставит ученику сообщение с реквизитами платежа и кнопкой прямого перехода к чеку ФНС.
                    </p>
                  </div>
                </label>

                {/* Live Telegram Preview */}
                {receiptSendTelegram && (
                  <div className="p-4 rounded-2xl border border-slate-200 dark:border-white/10 bg-slate-100/70 dark:bg-black/50 space-y-2 text-xs">
                    <div className="flex items-center justify-between text-[10px] font-mono text-slate-400 border-b border-slate-200 dark:border-white/10 pb-2">
                      <span className="flex items-center gap-1 font-semibold text-sky-500">
                        <Send className="w-3 h-3" /> Превью сообщения в Telegram-боте:
                      </span>
                      <span>@spark_prep_bot</span>
                    </div>

                    <div className="text-[11px] leading-relaxed text-slate-800 dark:text-slate-200 font-mono space-y-1">
                      <p className="font-bold">🧾 Электронный чек об оплате</p>
                      <p className="text-slate-500 dark:text-slate-400 text-[10px]">
                        Здравствуйте! Ваш платёж за обучение в проекте «так называемый SPARK» успешно зарегистрирован и фискализирован в соответствии с законодательством РФ (ФЗ № 422-ФЗ / 54-ФЗ).
                      </p>
                      <div className="pt-1 text-[10px] space-y-0.5">
                        <p>• <b>Услуга:</b> {receiptServiceName || "—"}</p>
                        <p>• <b>Тариф:</b> {issueReceiptTarget.tier === "vip" ? "«VIP»" : issueReceiptTarget.tier === "basic" ? "«Базовый»" : "«Акселератор»"}</p>
                        <p>• <b>Сумма:</b> {receiptAmount || "0"} ₽</p>
                        {receiptNumber && <p>• <b>Номер чека:</b> {receiptNumber}</p>}
                        <p>• <b>Дата формирования:</b> {new Date().toLocaleDateString("ru-RU")} МСК</p>
                      </div>
                    </div>

                    {/* Inline Button Preview */}
                    <div className="pt-2 flex flex-col gap-1.5">
                      {receiptUrl ? (
                        <div className="w-full py-2 px-3 rounded-xl bg-sky-500 hover:bg-sky-400 text-white font-mono text-xs font-bold text-center flex items-center justify-center gap-1.5 shadow-sm">
                          <Receipt className="w-3.5 h-3.5" />
                          <span>🧾 Открыть официальный чек</span>
                        </div>
                      ) : (
                        <div className="w-full py-1.5 px-3 rounded-xl border border-dashed border-slate-300 dark:border-white/20 text-slate-400 font-mono text-[10px] text-center">
                          (Кнопка чека появится при заполнении ссылки на чек)
                        </div>
                      )}
                    </div>
                  </div>
                )}
              </div>
            )}

            {/* Modal Actions */}
            <div className="flex gap-3 pt-3 border-t border-slate-200 dark:border-white/10">
              <button
                type="button"
                onClick={() => setIsIssueModalOpen(false)}
                className="flex-1 px-4 py-2.5 rounded-xl border border-slate-200 dark:border-white/10 text-xs font-mono hover:bg-slate-100 dark:hover:bg-white/5 transition-colors"
              >
                Отмена
              </button>
              <button
                type="submit"
                disabled={issuingReceipt || !issueReceiptTarget}
                className="flex-1 px-4 py-2.5 rounded-xl bg-emerald-600 hover:bg-emerald-500 text-white text-xs font-mono font-bold disabled:opacity-50 transition-all shadow-md flex items-center justify-center gap-2"
              >
                <FileCheck className="w-4 h-4" />
                <span>{issuingReceipt ? "Формирование..." : "Выписать и отправить чек"}</span>
              </button>
            </div>
          </form>
        </div>
      )}
    </div>
  );
}
