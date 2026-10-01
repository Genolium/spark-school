"use client";

import React, { useState, useEffect } from "react";
import { Header } from "@/components/Header";
import { Footer } from "@/components/Footer";
import { AdminLogin } from "@/components/Admin/AdminLogin";
import { FinancialStats } from "@/components/Admin/FinancialStats";
import { StudentsTable } from "@/components/Admin/StudentsTable";
import { PayoutsTable } from "@/components/Admin/PayoutsTable";
import { PromosTable } from "@/components/Admin/PromosTable";
import { api, AdminStats, UserProfile, PayoutRequest, PromoCode } from "@/lib/api";
import {
  ShieldCheck,
  RefreshCw,
  LogOut,
  Users,
  DollarSign,
  Tag,
} from "lucide-react";

type AdminTab = "students" | "promos" | "payouts";

export default function AdminPage() {
  const [unlocked, setUnlocked] = useState(false);
  const [activeTab, setActiveTab] = useState<AdminTab>("students");

  const [stats, setStats] = useState<AdminStats>({
    gross_volume: 289800,
    active_students: 42,
    total_registered: 656,
    conversion_rate: 6.4,
    pending_payouts_count: 3,
    pending_payouts_sum: 8500,
  });
  const [students, setStudents] = useState<UserProfile[]>([]);
  const [payouts, setPayouts] = useState<PayoutRequest[]>([]);
  const [promos, setPromos] = useState<PromoCode[]>([]);
  const [loading, setLoading] = useState(true);

  // Check if session token exists
  useEffect(() => {
    const savedToken = localStorage.getItem("spark_admin_token");
    const unlockedSession = sessionStorage.getItem("spark_admin_unlocked");
    if (savedToken || unlockedSession === "true") {
      setUnlocked(true);
    }
  }, []);

  const loadAdminData = async () => {
    setLoading(true);
    const [statsData, studentsData, payoutsData, promosData] = await Promise.all([
      api.getAdminStats(),
      api.getAdminStudents(),
      api.getAdminPayouts(),
      api.getAdminPromoCodes(),
    ]);
    setStats(statsData);
    setStudents(studentsData);
    setPayouts(payoutsData);
    setPromos(promosData);
    setLoading(false);
  };

  useEffect(() => {
    if (unlocked) {
      loadAdminData();
    }
  }, [unlocked]);

  const handleUnlock = () => {
    setUnlocked(true);
    loadAdminData();
  };

  const handleLogout = () => {
    api.adminLogout();
    setUnlocked(false);
  };

  const handleToggleAccess = async (userId: number, currentAccess: boolean) => {
    const nextAccess = !currentAccess;
    setStudents((prev) =>
      prev.map((s) => (s.id === userId ? { ...s, has_access: nextAccess } : s))
    );
    await api.toggleUserAccess(userId, nextAccess);
    loadAdminData();
  };

  const handleApprovePayout = async (id: number) => {
    setPayouts((prev) =>
      prev.map((p) => (p.id === id ? { ...p, status: "paid" } : p))
    );
    await api.approvePayout(id);
    loadAdminData();
  };

  const handleRejectPayout = async (id: number) => {
    setPayouts((prev) =>
      prev.map((p) => (p.id === id ? { ...p, status: "rejected" } : p))
    );
    await api.rejectPayout(id);
    loadAdminData();
  };

  return (
    <main className="min-h-screen bg-[var(--bg-base)] text-[var(--text-main)] relative selection:bg-emerald-500/20 selection:text-emerald-900 dark:selection:bg-white/20 dark:selection:text-white transition-colors duration-300">
      <Header />

      <div className="max-w-[1520px] mx-auto px-4 sm:px-6 pt-28 pb-20">
        {!unlocked ? (
          <AdminLogin onUnlock={handleUnlock} />
        ) : (
          <div>
            {/* Header / Admin Navigation Bar */}
            <div className="flex flex-col lg:flex-row lg:items-center justify-between gap-4 p-6 rounded-[32px] bento-card-dark border border-white/20 mb-8">
              <div>
                <div className="flex items-center gap-3">
                  <div className="w-10 h-10 rounded-xl bg-amber-500/20 text-amber-400 flex items-center justify-center font-bold border border-amber-500/30">
                    <ShieldCheck className="w-5 h-5" />
                  </div>
                  <div>
                    <div className="flex items-center gap-2">
                      <h1 className="font-editorial text-2xl sm:text-3xl font-bold text-white">
                        SPARK Admin Panel
                      </h1>
                      <span className="text-[10px] font-mono px-2 py-0.5 rounded-full bg-emerald-500/20 text-emerald-400 border border-emerald-500/30 font-bold">
                        v2.0 Admin
                      </span>
                    </div>
                    <p className="text-xs font-mono text-slate-400 mt-0.5">
                      Управление студентами и выплаты партнёрам • Куратор: так называемый Иль
                    </p>
                  </div>
                </div>
              </div>

              <div className="flex flex-wrap items-center gap-2 sm:gap-3">
                <button
                  onClick={loadAdminData}
                  disabled={loading}
                  className="px-4 py-2.5 rounded-full bg-white/10 hover:bg-white/20 text-xs font-mono text-white flex items-center gap-2 transition-all border border-white/10"
                >
                  <RefreshCw className={`w-3.5 h-3.5 ${loading ? "animate-spin text-amber-400" : ""}`} />
                  <span>Обновить</span>
                </button>

                <button
                  onClick={handleLogout}
                  className="px-4 py-2.5 rounded-full bg-rose-500/10 hover:bg-rose-500/20 text-xs font-mono text-rose-300 flex items-center gap-1.5 transition-all border border-rose-500/20"
                  title="Выйти из админки"
                >
                  <LogOut className="w-3.5 h-3.5" />
                  <span>Выйти</span>
                </button>
              </div>
            </div>

            {/* Navigation Tabs */}
            <div className="flex items-center gap-2 p-1.5 rounded-2xl bg-white/[0.04] border border-white/10 mb-8 max-w-fit overflow-x-auto">
              <button
                onClick={() => setActiveTab("students")}
                className={`px-5 py-2.5 rounded-xl text-xs font-mono font-bold transition-all flex items-center gap-2 whitespace-nowrap ${
                  activeTab === "students"
                    ? "bg-amber-500 text-black shadow-lg"
                    : "text-slate-400 hover:text-white hover:bg-white/5"
                }`}
              >
                <Users className="w-4 h-4" />
                <span>Студенты и доступы</span>
                <span className="px-1.5 py-0.2 rounded bg-black/20 text-[10px]">
                  {students.length}
                </span>
              </button>

              <button
                onClick={() => setActiveTab("promos")}
                className={`px-5 py-2.5 rounded-xl text-xs font-mono font-bold transition-all flex items-center gap-2 whitespace-nowrap ${
                  activeTab === "promos"
                    ? "bg-amber-500 text-black shadow-lg"
                    : "text-slate-400 hover:text-white hover:bg-white/5"
                }`}
              >
                <Tag className="w-4 h-4" />
                <span>Промокоды (-5%)</span>
                <span className="px-1.5 py-0.2 rounded bg-black/20 text-[10px]">
                  {promos.length}
                </span>
              </button>

              {/* Временно закомментировано: вкладка выплат партнёрам
              <button
                onClick={() => setActiveTab("payouts")}
                className={`px-5 py-2.5 rounded-xl text-xs font-mono font-bold transition-all flex items-center gap-2 whitespace-nowrap ${
                  activeTab === "payouts"
                    ? "bg-amber-500 text-black shadow-lg"
                    : "text-slate-400 hover:text-white hover:bg-white/5"
                }`}
              >
                <DollarSign className="w-4 h-4" />
                <span>Выплаты партнёрам</span>
                {stats.pending_payouts_count > 0 && (
                  <span className="px-1.5 py-0.2 rounded bg-rose-500 text-white text-[10px]">
                    {stats.pending_payouts_count}
                  </span>
                )}
              </button>
              */}
            </div>

            {/* TAB CONTENT */}

            {/* Tab 1: Students & Financials */}
            {activeTab === "students" && (
              <div>
                <FinancialStats stats={stats} />
                <StudentsTable
                  students={students}
                  onToggleAccess={handleToggleAccess}
                />
              </div>
            )}

            {/* Tab: Promo Codes (-5%) */}
            {activeTab === "promos" && (
              <div>
                <PromosTable
                  promos={promos}
                  onRefresh={loadAdminData}
                />
              </div>
            )}

            {/* Tab 2: Payouts (временно закомментировано)
            {activeTab === "payouts" && (
              <div>
                <PayoutsTable
                  payouts={payouts}
                  onApprove={handleApprovePayout}
                  onReject={handleRejectPayout}
                />
              </div>
            )}
            */}
          </div>
        )}
      </div>

      <Footer />
    </main>
  );
}
