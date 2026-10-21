"use client";

import React, { useCallback, useEffect, useState } from "react";
import { Check, Clipboard, MessageCircle, SkipForward } from "lucide-react";
import { api, OnboardingQueue, UserProfile } from "@/lib/api";

const WELCOME_TEMPLATE = `Привет! Я Иль, твой ментор по подготовке к SPARK. Я буду вести курс на этом аккаунте.

Можно тебя попросить написать ответы на эти вопросы сюда? Я хочу лучше тебя узнать — так мне будет гораздо проще давать точечные комментарии по твоим черновикам, а не общие советы.

Напиши, пожалуйста, пару слов о себе:
🔵 База: из какого ты города, в каком вузе учишься, какой курс и специальность?
🔵 Чем занимаешься вне пар: проекты, студенческие клубы, волонтёрство, спорт, музыка или другие увлечения? Даже если кажется, что это мелочи — пиши всё, мы потом вытащим оттуда самое интересное.
🔵 Как оцениваешь свой уровень английского? Есть ли опыт общения или пока языковой барьер?
🔵 Что в процессе подачи пугает или вызывает больше всего вопросов прямо сейчас?

Буду ждать твой ответ 🤩

Если уже начал(а) проходить курс — расскажи, как успехи. Если будут вопросы по заявке или захочешь прислать что-то на проверку, можешь писать прямо сюда. Буду рад помочь 🤩`;

const STATUS_LABELS: Record<string, string> = {
  pending_intro: "Нужно приветствие",
  awaiting_reply: "Ждём ответ",
  completed: "Ответил",
  skipped: "Пропущен",
};

export function OnboardingBoard() {
  const [data, setData] = useState<OnboardingQueue | null>(null);
  const [error, setError] = useState(false);
  const [copied, setCopied] = useState<number | null>(null);

  const load = useCallback(async () => {
    const result = await api.getOnboarding();
    setData(result);
    setError(!result);
  }, []);

  useEffect(() => {
    load();
  }, [load]);

  const copy = async (user: UserProfile) => {
    const text = `Привет, ${user.first_name || "друг"}! ${WELCOME_TEMPLATE.slice("Привет! ".length)}`;
    try {
      await navigator.clipboard.writeText(text);
      setCopied(user.id);
      setTimeout(() => setCopied(null), 2000);
    } catch {
      setCopied(null);
    }
  };

  const move = async (user: UserProfile, status: string) => {
    if (await api.updateOnboarding(user.id, status)) load();
  };

  return (
    <div className="rounded-3xl border border-slate-200 dark:border-white/10 bg-white dark:bg-white/5 p-6 mb-6">
      <div className="flex flex-wrap items-center justify-between gap-3 mb-5">
        <div>
          <h3 className="font-semibold text-lg">Онбординг новых студентов</h3>
          <p className="text-xs text-slate-500">
            Доступ выдан → приветствие отправлено → student ответил. Только новые оплаты.
          </p>
        </div>
        <button onClick={load} className="rounded-lg bg-emerald-600 text-white px-3 py-1 text-sm">
          Обновить
        </button>
      </div>

      {data && (
        <div className="grid grid-cols-3 gap-3 mb-5">
          <div className="rounded-2xl bg-amber-500/10 p-3">
            <p className="text-xs text-slate-500">Нужно приветствие</p>
            <p className="text-xl font-semibold">{data.counts.pending_intro}</p>
          </div>
          <div className="rounded-2xl bg-sky-500/10 p-3">
            <p className="text-xs text-slate-500">Ждём ответ</p>
            <p className="text-xl font-semibold">{data.counts.awaiting_reply}</p>
          </div>
          <div className="rounded-2xl bg-emerald-500/10 p-3">
            <p className="text-xs text-slate-500">Ответили</p>
            <p className="text-xl font-semibold">{data.counts.completed}</p>
          </div>
        </div>
      )}

      {error && <p className="text-sm text-red-500">Не удалось загрузить очередь онбординга.</p>}
      {!error && data?.users.length === 0 && <p className="text-sm text-slate-500">Новых студентов нет.</p>}

      {data && data.users.length > 0 && (
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="text-left text-slate-500 border-b">
                <th className="py-2">Студент</th>
                <th className="py-2">Тариф</th>
                <th className="py-2">Доступ</th>
                <th className="py-2">Статус</th>
                <th className="py-2">Действия</th>
              </tr>
            </thead>
            <tbody>
              {data.users.map((user) => (
                <tr key={user.id} className="border-b last:border-0">
                  <td className="py-3">
                    <div className="font-medium">{user.first_name || "Без имени"}</div>
                    <div className="text-xs text-slate-500">@{user.username || "—"} · ID {user.telegram_id}</div>
                  </td>
                  <td className="py-3 uppercase text-xs">{user.tier}</td>
                  <td className="py-3 text-xs">{user.access_granted_at ? new Date(user.access_granted_at).toLocaleString("ru-RU") : "—"}</td>
                  <td className="py-3">
                    <span className="rounded-full bg-slate-500/10 px-2 py-0.5 text-xs">
                      {STATUS_LABELS[user.onboarding_status || ""] || user.onboarding_status || "—"}
                    </span>
                  </td>
                  <td className="py-3">
                    <div className="flex flex-wrap gap-2">
                      <button onClick={() => copy(user)} className="inline-flex items-center gap-1 rounded-lg border px-2 py-1 text-xs">
                        {copied === user.id ? <Check className="w-3 h-3" /> : <Clipboard className="w-3 h-3" />}
                        {copied === user.id ? "Скопировано" : "Приветствие"}
                      </button>
                      {user.onboarding_status === "pending_intro" && (
                        <button onClick={() => move(user, "awaiting_reply")} className="inline-flex items-center gap-1 rounded-lg bg-sky-600 px-2 py-1 text-xs text-white">
                          <MessageCircle className="w-3 h-3" /> Отправлено
                        </button>
                      )}
                      {user.onboarding_status === "awaiting_reply" && (
                        <button onClick={() => move(user, "completed")} className="inline-flex items-center gap-1 rounded-lg bg-emerald-600 px-2 py-1 text-xs text-white">
                          <Check className="w-3 h-3" /> Ответил
                        </button>
                      )}
                      <button onClick={() => move(user, "skipped")} className="inline-flex items-center gap-1 rounded-lg border px-2 py-1 text-xs text-slate-500">
                        <SkipForward className="w-3 h-3" /> Пропустить
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
