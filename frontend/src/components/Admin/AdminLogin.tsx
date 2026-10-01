"use client";

import React, { useState, useEffect } from "react";
import { Lock, User, Eye, EyeOff, ShieldAlert, ArrowRight, ShieldCheck, Timer } from "lucide-react";
import { api } from "@/lib/api";

interface AdminLoginProps {
  onUnlock: () => void;
}

export function AdminLogin({ onUnlock }: AdminLoginProps) {
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [showPassword, setShowPassword] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [lockoutSeconds, setLockoutSeconds] = useState(0);

  // Countdown timer for rate limiting lockout
  useEffect(() => {
    if (lockoutSeconds <= 0) return;
    const interval = setInterval(() => {
      setLockoutSeconds((prev) => {
        if (prev <= 1) {
          clearInterval(interval);
          setError(null);
          return 0;
        }
        return prev - 1;
      });
    }, 1000);
    return () => clearInterval(interval);
  }, [lockoutSeconds]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (lockoutSeconds > 0) return;

    if (!username.trim() || !password) {
      setError("Пожалуйста, заполните логин и пароль");
      return;
    }

    setLoading(true);
    setError(null);

    const result = await api.adminLogin(username.trim(), password);
    setLoading(false);

    if (result.success) {
      onUnlock();
    } else {
      setError(result.error || "Неверный логин или пароль администратора");
      if (result.retryAfter && result.retryAfter > 0) {
        setLockoutSeconds(result.retryAfter);
      }
    }
  };

  return (
    <div className="min-h-[65vh] flex items-center justify-center p-4">
      <div className="w-full max-w-md bento-card-dark p-8 sm:p-10 rounded-[36px] border border-white/20 text-center shadow-2xl relative overflow-hidden">
        {/* Subtle Ambient Glow */}
        <div className="absolute top-0 left-1/2 -translate-x-1/2 w-48 h-24 bg-amber-500/10 rounded-full blur-2xl pointer-events-none" />

        <div className="w-16 h-16 rounded-2xl bg-amber-500/20 text-amber-400 flex items-center justify-center mx-auto mb-6 border border-amber-500/30">
          <ShieldCheck className="w-8 h-8" />
        </div>

        <h2 className="text-2xl sm:text-3xl font-bold text-white mb-2 tracking-tight">
          Панель администратора
        </h2>
        <p className="text-xs sm:text-sm text-slate-400 mb-6 font-normal">
          Вход в панель управления акселератором SPARK &apos;27
        </p>

        <form onSubmit={handleSubmit} className="space-y-4 text-left">
          {/* Username Input */}
          <div>
            <label className="block text-xs font-mono text-slate-400 mb-1.5 uppercase tracking-wider">
              Логин куратора
            </label>
            <div className="relative">
              <div className="absolute inset-y-0 left-0 pl-3.5 flex items-center pointer-events-none text-slate-400">
                <User className="w-4 h-4" />
              </div>
              <input
                type="text"
                required
                disabled={lockoutSeconds > 0 || loading}
                placeholder="admin"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                autoFocus
                className="w-full pl-10 pr-4 py-3 bg-white/10 border border-white/15 rounded-2xl text-white placeholder-slate-500 text-sm font-medium focus:outline-none focus:border-amber-400 transition-colors disabled:opacity-50"
              />
            </div>
          </div>

          {/* Password Input */}
          <div>
            <label className="block text-xs font-mono text-slate-400 mb-1.5 uppercase tracking-wider">
              Пароль
            </label>
            <div className="relative">
              <div className="absolute inset-y-0 left-0 pl-3.5 flex items-center pointer-events-none text-slate-400">
                <Lock className="w-4 h-4" />
              </div>
              <input
                type={showPassword ? "text" : "password"}
                required
                disabled={lockoutSeconds > 0 || loading}
                placeholder="••••••••••••"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                className="w-full pl-10 pr-11 py-3 bg-white/10 border border-white/15 rounded-2xl text-white placeholder-slate-500 text-sm font-medium focus:outline-none focus:border-amber-400 transition-colors disabled:opacity-50"
              />
              <button
                type="button"
                onClick={() => setShowPassword(!showPassword)}
                className="absolute inset-y-0 right-0 pr-3.5 flex items-center text-slate-400 hover:text-white transition-colors"
              >
                {showPassword ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
              </button>
            </div>
          </div>

          {/* Error / Rate Limit Message */}
          {error && (
            <div className="p-3.5 rounded-xl bg-rose-500/15 border border-rose-500/30 text-xs font-mono text-rose-300 flex items-start gap-2.5">
              {lockoutSeconds > 0 ? (
                <Timer className="w-4 h-4 shrink-0 text-amber-400 mt-0.5 animate-pulse" />
              ) : (
                <ShieldAlert className="w-4 h-4 shrink-0 text-rose-400 mt-0.5" />
              )}
              <div>
                <p>{error}</p>
                {lockoutSeconds > 0 && (
                  <p className="text-amber-400 font-bold mt-1">
                    Повторная попытка доступна через: {lockoutSeconds} сек.
                  </p>
                )}
              </div>
            </div>
          )}

          {/* Submit Button */}
          <button
            type="submit"
            disabled={loading || lockoutSeconds > 0}
            className="btn-primary-mono w-full py-3.5 text-sm font-bold shadow-lg flex items-center justify-center gap-2 mt-2 disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {loading ? (
              <span>Проверка доступа...</span>
            ) : lockoutSeconds > 0 ? (
              <span className="flex items-center gap-1.5 text-slate-400">
                <Timer className="w-4 h-4" />
                Блокировка: {lockoutSeconds} с
              </span>
            ) : (
              <>
                <span>Войти в админ-панель</span>
                <ArrowRight className="w-4 h-4" />
              </>
            )}
          </button>
        </form>

        {/* Test Credentials */}
        <div className="mt-6 pt-4 border-t border-white/10">
          <div className="text-[11px] font-mono text-slate-500 text-center">
            Стандартные данные для входа: логин <strong>admin</strong>, пароль <strong>SparkAdmin2026!</strong>
          </div>
        </div>
      </div>
    </div>
  );
}
