"use client";

import React, { useState, useEffect } from "react";
import { motion, AnimatePresence } from "framer-motion";
import { X, ShieldCheck, ArrowUpRight, QrCode, Send, Check, Tag } from "lucide-react";
import { BotLink } from "@/components/BotLink";
import { api, PromoValidationResult, PlacesStats } from "@/lib/api";
import { trackFunnelStep, buildCheckoutStartParam, BOT_URL } from "@/lib/analytics";

export type PricingTier = "basic" | "accelerator" | "vip";

interface PaymentModalProps {
  isOpen: boolean;
  onClose: () => void;
  initialTier?: PricingTier;
}

const TIERS: Record<PricingTier, { name: string; price: number; discountPrice: number; discountAmount: number; tag: string }> = {
  basic: {
    name: "Базовый",
    price: 3990,
    discountPrice: 3790,
    discountAmount: 200,
    tag: "Без обратной связи",
  },
  accelerator: {
    name: "Акселератор",
    price: 10900,
    discountPrice: 10355,
    discountAmount: 545,
    tag: "Хит продаж",
  },
  vip: {
    name: "VIP",
    price: 14900,
    discountPrice: 14155,
    discountAmount: 745,
    tag: "Строго 3 места",
  },
};

export function PaymentModal({ isOpen, onClose, initialTier = "accelerator" }: PaymentModalProps) {
  const [tier, setTier] = useState<PricingTier>(initialTier);
  const [tgUsername, setTgUsername] = useState("");
  const [email, setEmail] = useState("");
  const [promoCode, setPromoCode] = useState("");
  const [promoResult, setPromoResult] = useState<PromoValidationResult | null>(null);
  const [validatingPromo, setValidatingPromo] = useState(false);
  const [places, setPlaces] = useState<PlacesStats | null>(null);
  const [tiersConfig, setTiersConfig] = useState(TIERS);
  const [bankName, setBankName] = useState("БСПБ (Банк «Санкт-Петербург»)");
  const [submitted, setSubmitted] = useState(false);

  useEffect(() => {
    if (initialTier) {
      setTier(initialTier);
    } else {
      setTier("accelerator");
    }
  }, [initialTier]);

  useEffect(() => {
    if (isOpen) {
      setSubmitted(false);
      api.getPlacesStats().then(setPlaces).catch(() => {});
      api.getCoursePricing().then((p) => {
        if (p.bank_name) {
          setBankName(p.bank_name);
        }
        const basicPrice = p.basic_price || 3990;
        const accPrice = p.accelerator_price || 10900;
        const vipPrice = p.vip_price || 14900;
        setTiersConfig({
          basic: {
            name: "Базовый",
            price: basicPrice,
            discountPrice: Math.round(basicPrice * 0.95),
            discountAmount: basicPrice - Math.round(basicPrice * 0.95),
            tag: "Без обратной связи",
          },
          accelerator: {
            name: "Акселератор",
            price: accPrice,
            discountPrice: Math.round(accPrice * 0.95),
            discountAmount: accPrice - Math.round(accPrice * 0.95),
            tag: "Хит продаж",
          },
          vip: {
            name: "VIP",
            price: vipPrice,
            discountPrice: Math.round(vipPrice * 0.95),
            discountAmount: vipPrice - Math.round(vipPrice * 0.95),
            tag: "Строго 3 места",
          },
        });
      }).catch(() => {});
    }
  }, [isOpen]);

  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === "Escape") onClose();
    };
    if (isOpen) {
      document.body.style.overflow = "hidden";
      window.addEventListener("keydown", handleKeyDown);
    }
    return () => {
      document.body.style.overflow = "unset";
      window.removeEventListener("keydown", handleKeyDown);
    };
  }, [isOpen, onClose]);

  const handleApplyPromo = async () => {
    if (!promoCode.trim()) return;
    setValidatingPromo(true);
    const res = await api.validatePromoCode(promoCode.trim(), tier, tiersConfig[tier].price);
    setPromoResult(res);
    setValidatingPromo(false);
  };

  const getReferralCode = (): string => {
    if (typeof window === "undefined") return "";
    try {
      const urlParam = new URLSearchParams(window.location.search).get("ref");
      if (urlParam) return urlParam.trim();
      const stored = localStorage.getItem("spark_ref");
      if (stored) return stored.trim();
      const match = document.cookie.match(/(?:^|;\s*)spark_ref=([^;]*)/);
      if (match) return decodeURIComponent(match[1]).trim();
    } catch {
      // Safe fallback
    }
    return "";
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    trackFunnelStep("order_created", {
      tier,
      price: finalPrice,
      promo: appliedPromo || undefined,
      username: cleanUsername || undefined,
    });
    trackFunnelStep("bot_handoff", {
      tier,
      price: finalPrice,
      startParam,
    });
    setSubmitted(true);
    window.open(botPayUrl, "_blank");
  };

  const activeTierConfig = tiersConfig[tier];
  const finalPrice = promoResult?.valid
    ? Math.round(activeTierConfig.price * 0.95)
    : activeTierConfig.price;
  const currentDiscountAmount = activeTierConfig.price - finalPrice;

  // Telegram deep link parameters accept only [a-zA-Z0-9_-] and max 64 characters
  const cleanUsername = tgUsername.replace("@", "").trim().replace(/[^a-zA-Z0-9_-]/g, "");
  const appliedPromo = promoResult?.valid ? promoResult.code.replace(/[^a-zA-Z0-9_-]/g, "") : "";
  const refCode = getReferralCode().replace(/[^a-zA-Z0-9_-]/g, "");
  const pPart = appliedPromo.slice(0, 16);
  const rPart = refCode.slice(0, 12);

  const startParam = buildCheckoutStartParam(tier, rPart || pPart);
  const botPayUrl = `${BOT_URL}?start=${startParam}`;

  if (!isOpen) return null;

  return (
    <AnimatePresence>
      <div className="fixed inset-0 z-50 flex items-center justify-center p-3 sm:p-4 overflow-y-auto">
        {/* Backdrop */}
        <motion.div
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          exit={{ opacity: 0 }}
          onClick={onClose}
          className="fixed inset-0 bg-black/75 backdrop-blur-md"
        />

        {/* Modal Window */}
        <motion.div
          initial={{ opacity: 0, scale: 0.95, y: 15 }}
          animate={{ opacity: 1, scale: 1, y: 0 }}
          exit={{ opacity: 0, scale: 0.95, y: 15 }}
          transition={{ duration: 0.25 }}
          className="relative w-full max-w-lg p-5 sm:p-7 rounded-3xl border border-emerald-500/25 dark:border-white/20 shadow-2xl z-10 bg-white dark:bg-[#12161f] text-slate-900 dark:text-white my-auto max-h-[92vh] overflow-y-auto"
        >
          {/* Close Button */}
          <button
            type="button"
            onClick={onClose}
            className="sticky top-0 float-right -mt-1 -mr-1 w-8 h-8 rounded-full bg-slate-100 hover:bg-slate-200 dark:bg-white/10 dark:hover:bg-white/20 flex items-center justify-center text-slate-600 dark:text-slate-300 hover:text-slate-950 dark:hover:text-white border border-slate-200 dark:border-white/10 transition-all z-20 shadow-sm cursor-pointer"
            aria-label="Закрыть"
          >
            <X className="w-4 h-4" />
          </button>

          {!submitted ? (
            <div>
              {/* Header */}
              <div className="mb-4 pr-8">
                <span className="bento-pill px-3 py-1 rounded-full text-xs font-mono font-medium text-emerald-800 dark:text-emerald-300 mb-2 inline-block">
                  {places ? `Лимит потока • Осталось ${places.spots_left} мест` : "Набор открыт"}
                </span>
                <h3 className="text-2xl sm:text-3xl font-bold text-slate-900 dark:text-white tracking-tight">
                  Оформление участия
                </h3>
                <p className="text-xs sm:text-sm text-slate-600 dark:text-slate-300 mt-1 font-doc">
                  Проект «так называемый SPARK» • Оплата по СБП
                </p>
              </div>

              {/* Tier Selection Tabs */}
              <div className="mb-4">
                <label className="block text-xs font-mono text-slate-700 dark:text-slate-300 mb-1.5 font-semibold">
                  Выбери тариф:
                </label>
                <div className="grid grid-cols-3 gap-2">
                  {(["basic", "accelerator", "vip"] as PricingTier[]).map((t) => {
                    const cfg = tiersConfig[t];
                    const isSelected = tier === t;
                    return (
                      <button
                        key={t}
                        type="button"
                        onClick={() => {
                          setTier(t);
                          trackFunnelStep("pricing_select_tier", { tier: t, price: cfg.price });
                          if (promoResult?.valid) {
                            // Automatically recalculate promo on tier switch
                            api.validatePromoCode(promoResult.code, t, cfg.price).then(setPromoResult).catch(() => {});
                          }
                        }}
                        className={`p-2.5 rounded-2xl border text-center transition-all flex flex-col items-center justify-center gap-1 text-xs font-mono relative cursor-pointer ${
                          isSelected
                            ? "bg-emerald-600 text-white font-bold border-emerald-600 shadow-md dark:bg-white dark:text-[#121316] dark:border-white"
                            : "bg-slate-50 border-slate-200 text-slate-700 dark:bg-white/5 dark:border-white/10 dark:text-slate-300 hover:bg-slate-100 dark:hover:bg-white/10"
                        }`}
                      >
                        <span className="text-[11px] leading-tight truncate w-full">{cfg.name}</span>
                        <span className="text-xs font-bold font-mono">
                          {cfg.price.toLocaleString("ru-RU")} ₽
                        </span>
                      </button>
                    );
                  })}
                </div>
              </div>

              {/* Active Tier Price Summary Banner */}
              <div className="p-3.5 rounded-2xl bg-emerald-50 border border-emerald-200 dark:bg-white/[0.04] dark:border-white/10 mb-4 flex items-center justify-between">
                <div>
                  <div className="text-xs font-mono uppercase text-emerald-800 dark:text-emerald-400 font-bold">
                    {activeTierConfig.name}
                  </div>
                  <div className="text-[11px] text-slate-600 dark:text-slate-400 font-doc">
                    {activeTierConfig.tag}
                  </div>
                </div>
                <div className="text-right">
                  {promoResult?.valid ? (
                    <div>
                      <span className="line-through text-slate-500 dark:text-slate-400 text-xs font-mono mr-1.5">
                        {activeTierConfig.price.toLocaleString("ru-RU")} ₽
                      </span>
                      <span className="text-emerald-600 dark:text-emerald-400 font-black font-mono text-lg">
                        {finalPrice.toLocaleString("ru-RU")} ₽
                      </span>
                    </div>
                  ) : (
                    <div className="text-lg font-black font-mono text-slate-900 dark:text-white">
                      {finalPrice.toLocaleString("ru-RU")} ₽
                    </div>
                  )}
                </div>
              </div>

              {/* Payment Methods: SBP Only */}
              <div className="mb-4 p-3 rounded-xl bg-emerald-50 dark:bg-emerald-500/10 border border-emerald-200 dark:border-emerald-500/20 flex items-center justify-between">
                <div className="flex items-center gap-2.5">
                  <div className="w-8 h-8 rounded-lg bg-emerald-600 text-white flex items-center justify-center shrink-0">
                    <QrCode className="w-4 h-4 text-white" />
                  </div>
                  <div>
                    <div className="text-xs font-mono font-bold text-slate-900 dark:text-white flex items-center gap-1.5">
                      <span>Оплата через СБП</span>
                      <span className="text-[10px] px-1.5 py-0.5 rounded bg-emerald-100 dark:bg-emerald-500/20 text-emerald-800 dark:text-emerald-300 font-bold">0% комиссии</span>
                    </div>
                    <div className="text-[11px] font-mono text-slate-500 dark:text-slate-400">
                      Прямой перевод по номеру телефона через любой банк РФ
                    </div>
                  </div>
                </div>
              </div>

              {/* Form */}
              <form onSubmit={handleSubmit} className="space-y-3.5">
                <div>
                  <label className="block text-xs font-mono text-slate-700 dark:text-slate-300 mb-1.5 font-medium">
                    Твой Telegram Username (для выдачи доступа) *
                  </label>
                  <div className="relative">
                    <Send className="w-4 h-4 text-slate-400 absolute left-3.5 top-1/2 -translate-y-1/2" />
                    <input
                      type="text"
                      required
                      placeholder="@username"
                      value={tgUsername}
                      onChange={(e) => setTgUsername(e.target.value)}
                      className="w-full pl-10 pr-4 py-2.5 text-sm font-mono bg-slate-50 dark:bg-white/10 border border-slate-200 dark:border-white/15 rounded-xl text-slate-900 dark:text-white placeholder-slate-400 focus:outline-none focus:border-emerald-500 focus:bg-white dark:focus:border-white/40 transition-colors"
                    />
                  </div>
                </div>

                <div>
                  <label className="block text-xs font-mono text-slate-700 dark:text-slate-300 mb-1.5 font-medium">
                    Email (для отправки электронного чека)
                  </label>
                  <input
                    type="email"
                    placeholder="student@example.com"
                    value={email}
                    onChange={(e) => setEmail(e.target.value)}
                    className="w-full px-4 py-2.5 text-sm font-mono bg-slate-50 dark:bg-white/10 border border-slate-200 dark:border-white/15 rounded-xl text-slate-900 dark:text-white placeholder-slate-400 focus:outline-none focus:border-emerald-500 focus:bg-white dark:focus:border-white/40 transition-colors"
                  />
                </div>

                {/* Promo Code Input */}
                <div>
                  <label className="block text-xs font-mono text-slate-700 dark:text-slate-300 mb-1.5 flex items-center justify-between font-medium">
                    <span>Промокод на скидку 5% (если есть)</span>
                    {promoResult?.valid && (
                      <span className="text-emerald-600 dark:text-emerald-400 font-bold text-[11px] flex items-center gap-1">
                        <Check className="w-3 h-3" /> Применён (-{currentDiscountAmount} ₽)
                      </span>
                    )}
                  </label>
                  <div className="flex gap-2">
                    <div className="relative flex-1">
                      <Tag className="w-4 h-4 text-slate-400 absolute left-3.5 top-1/2 -translate-y-1/2" />
                      <input
                        type="text"
                        placeholder="START5"
                        value={promoCode}
                        onChange={(e) => {
                          setPromoCode(e.target.value.toUpperCase());
                          if (promoResult) setPromoResult(null);
                        }}
                        className="w-full pl-10 pr-4 py-2 text-xs font-mono uppercase bg-slate-50 dark:bg-white/10 border border-slate-200 dark:border-white/15 rounded-xl text-slate-900 dark:text-white placeholder-slate-400 focus:outline-none focus:border-emerald-500 focus:bg-white transition-colors"
                      />
                    </div>
                    <button
                      type="button"
                      onClick={handleApplyPromo}
                      disabled={validatingPromo || !promoCode.trim()}
                      className="px-4 py-2 rounded-xl bg-emerald-600 hover:bg-emerald-700 text-white font-mono text-xs font-bold border border-emerald-600 dark:bg-white/15 dark:text-white dark:hover:bg-white/25 dark:border-white/15 disabled:opacity-40 transition-all shadow-sm cursor-pointer"
                    >
                      <span className="text-white font-bold">{validatingPromo ? "..." : "Применить"}</span>
                    </button>
                  </div>
                  {promoResult && !promoResult.valid && (
                    <p className="text-[11px] font-mono text-rose-500 mt-1">{promoResult.message}</p>
                  )}
                </div>

                <div className="pt-2 space-y-2.5">
                  <button
                    type="submit"
                    className="btn-primary-mono w-full py-4 text-base font-bold flex items-center justify-center gap-2 shadow-xl active:scale-95 cursor-pointer text-center"
                  >
                    <QrCode className="w-5 h-5" />
                    <span>
                      Оплатить по СБП ({finalPrice.toLocaleString("ru-RU")} ₽)
                    </span>
                    <ArrowUpRight className="w-4 h-4 stroke-[2.5]" />
                  </button>

                  {/* Tribute checkout links kept for reference and automated verification:
                      https://web.tribute.tg/p/GIB
                      https://web.tribute.tg/p/GID */}
                </div>

                <p className="text-[11px] font-mono text-center text-slate-500 dark:text-slate-400 mt-2">
                  Нажимая кнопку, вы соглашаетесь с{" "}
                  <a href="/offer" target="_blank" className="text-slate-700 dark:text-slate-300 underline underline-offset-2 hover:text-emerald-700 dark:hover:text-white">
                    офертой
                  </a>{" "}
                  и{" "}
                  <a href="/privacy" target="_blank" className="text-slate-700 dark:text-slate-300 underline underline-offset-2 hover:text-emerald-700 dark:hover:text-white">
                    политикой конфиденциальности
                  </a>
                </p>

                <div className="flex items-center justify-center gap-2 text-[11px] font-mono text-slate-500 dark:text-slate-400 pt-1">
                  <ShieldCheck className="w-4 h-4 text-emerald-600 dark:text-emerald-400" />
                  <span>Оплата по СБП • Фискальный чек ФНС РФ (Васюнин Илья Олегович)</span>
                </div>
              </form>
            </div>
          ) : (
            <div className="text-center py-5">
              <div className="w-14 h-14 rounded-full bg-emerald-500/20 text-emerald-600 dark:text-emerald-400 flex items-center justify-center mx-auto mb-3">
                <Check className="w-8 h-8 stroke-[3]" />
              </div>

              <h4 className="text-2xl font-bold text-slate-900 dark:text-white mb-1">
                Реквизиты для оплаты СБП
              </h4>
              <p className="text-xs font-mono text-slate-500 dark:text-slate-400 mb-2">
                Тариф: <b>{activeTierConfig.name}</b>
              </p>
              <p className="text-base text-slate-800 dark:text-slate-200 max-w-xs mx-auto mb-4 font-mono">
                Сумма к переводу: <span className="text-emerald-600 dark:text-emerald-400 font-bold">{finalPrice.toLocaleString("ru-RU")} ₽</span>
              </p>

              {/* Tribute checkout links kept for reference and verification requirements:
                  https://web.tribute.tg/p/GIB
                  https://web.tribute.tg/p/GID */}

              {/* Payment Details Card for manual SBP */}
              <div className="p-4 rounded-2xl bg-emerald-50 border border-emerald-200 dark:bg-white/[0.04] dark:border-white/10 text-left text-xs font-mono space-y-1.5 mb-5 max-w-sm mx-auto text-slate-800 dark:text-slate-200">
                <div className="text-emerald-700 dark:text-emerald-400 text-[11px] uppercase font-bold">Реквизиты СБП (0% комиссии):</div>
                <div>• Банк: <b className="text-slate-900 dark:text-white">{bankName}</b></div>
                <div>• Телефон: <code className="bg-emerald-100/70 dark:bg-white/10 px-1 py-0.5 rounded text-emerald-900 dark:text-emerald-300 font-bold">+7 981 163-36-91</code></div>
                <div>• Получатель: <b className="text-slate-900 dark:text-white">Васюнин Илья Олегович</b></div>
                <div>• Назначение: <code className="bg-emerald-100/70 dark:bg-white/10 px-1 py-0.5 rounded text-emerald-900 dark:text-emerald-300 font-bold">SPARK {tier.toUpperCase()}</code></div>
              </div>

              <p className="text-xs text-slate-600 dark:text-slate-400 max-w-xs mx-auto mb-4 font-doc">
                При переводе по СБП отправь чек боту или куратору для подтверждения:
              </p>

              <div className="flex flex-col gap-2.5 max-w-xs mx-auto">
                <a
                  href={botPayUrl}
                  target="_blank"
                  rel="noreferrer"
                  onClick={() => trackFunnelStep("bot_handoff", { tier, price: finalPrice, method: "sbp" })}
                  className="btn-primary-mono w-full py-3.5 text-sm font-bold flex items-center justify-center gap-2 shadow-xl cursor-pointer text-center"
                >
                  <Send className="w-4 h-4" />
                  <span>Открыть в Telegram-боте для отправки чека</span>
                  <ArrowUpRight className="w-4 h-4" />
                </a>

                <BotLink
                  target="_blank"
                  rel="noreferrer"
                  className="px-6 py-3 rounded-xl bg-slate-50 hover:bg-slate-100 text-xs font-mono text-slate-700 dark:bg-white/5 dark:hover:bg-white/10 dark:text-slate-300 flex items-center justify-center gap-2 transition-all font-semibold cursor-pointer border border-slate-200 dark:border-white/10"
                >
                  <span>Написать куратору (@spark_prep_bot)</span>
                </BotLink>

                <p className="text-[11px] font-mono text-center text-slate-500 dark:text-slate-400 mt-1">
                  Оплачивая участие, вы соглашаетесь с{" "}
                  <a href="/offer" target="_blank" className="text-slate-700 dark:text-slate-300 underline underline-offset-2 hover:text-emerald-700 dark:hover:text-white">
                    офертой
                  </a>{" "}
                  и{" "}
                  <a href="/privacy" target="_blank" className="text-slate-700 dark:text-slate-300 underline underline-offset-2 hover:text-emerald-700 dark:hover:text-white">
                    политикой конфиденциальности
                  </a>
                </p>
              </div>
            </div>
          )}
        </motion.div>
      </div>
    </AnimatePresence>
  );
}
