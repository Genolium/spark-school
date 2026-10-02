"use client";

import React, { useState, useEffect } from "react";
import { Header } from "@/components/Header";
import { Hero } from "@/components/Hero";
import { SocialProofStrip } from "@/components/SocialProofStrip";
import { HardTruth } from "@/components/HardTruth";
import { KibiRoadmap } from "@/components/KibiRoadmap";
import { PlacementMap } from "@/components/PlacementMap";
import { PlatformFeatures } from "@/components/PlatformFeatures";
import { AuthorSection } from "@/components/AuthorSection";
import { PricingSection } from "@/components/PricingSection";
import { FaqSection } from "@/components/FaqSection";
import { Footer } from "@/components/Footer";
import { PaymentModal, PricingTier } from "@/components/Modals/PaymentModal";
import { ChancesModal } from "@/components/Modals/ChancesModal";
import { ReferralTracker } from "@/components/ReferralTracker";

export default function HomePage() {
  const [isPaymentOpen, setIsPaymentOpen] = useState(false);
  const [isChancesOpen, setIsChancesOpen] = useState(false);
  const [selectedTier, setSelectedTier] = useState<PricingTier>("accelerator");

  const handleOpenPayment = (tier?: PricingTier) => {
    if (tier) {
      setSelectedTier(tier);
    }
    setIsPaymentOpen(true);
  };

  useEffect(() => {
    const scrollToHash = () => {
      if (typeof window !== "undefined" && window.location.hash) {
        const id = window.location.hash.replace("#", "");
        const el = document.getElementById(id);
        if (el) {
          el.scrollIntoView({ behavior: "smooth" });
        }
      }
    };

    const timer = setTimeout(scrollToHash, 150);
    window.addEventListener("hashchange", scrollToHash);

    return () => {
      clearTimeout(timer);
      window.removeEventListener("hashchange", scrollToHash);
    };
  }, []);

  return (
    <main className="min-h-screen bg-[var(--bg-base)] text-[var(--text-main)] relative selection:bg-emerald-500/20 selection:text-emerald-900 dark:selection:bg-white/20 dark:selection:text-white transition-colors duration-300">
      {/* Referral Tracker query string listener (?ref=...) */}
      <ReferralTracker />

      {/* Detached Floating Navigation Pill */}
      <Header
        onOpenPayment={() => handleOpenPayment("accelerator")}
      />

      {/* Screen 1: Spatial Bento-Glass Hero */}
      <Hero
        onOpenPayment={() => handleOpenPayment("accelerator")}
        onOpenChances={() => setIsChancesOpen(true)}
      />

      {/* Social Proof Strip: 5 Capsules (Инфо о программе $20,000) */}
      <SocialProofStrip onOpenChances={() => setIsChancesOpen(true)} />

      {/* Screen 2: Hard Truth (Why 90% Fail) */}
      <HardTruth />

      {/* Screen 3: The Journey (Kibi-Style Curved Flight Path) */}
      <KibiRoadmap />

      {/* Screen 4: Verified Placements Map (24 Past Cohort Campuses) */}
      <PlacementMap />

      {/* Screen 5: Product Features & Bento Architecture */}
      <PlatformFeatures />

      {/* Screen 6: The Author (так называемый Иль Story & J-1 Experience) */}
      <AuthorSection />

      {/* Screen 7: 3-Tier Pricing Matrix & Scarcity */}
      <PricingSection onOpenPayment={handleOpenPayment} />

      {/* Screen 8: FAQ Accordions */}
      <FaqSection />

      {/* Screen 9: Legal Footer & Disclaimer */}
      <Footer />

      {/* Interactive Chances Evaluation Modal */}
      <ChancesModal
        isOpen={isChancesOpen}
        onClose={() => setIsChancesOpen(false)}
      />

      {/* Payment Checkout Modal with Multi-Tier Support */}
      <PaymentModal
        isOpen={isPaymentOpen}
        onClose={() => setIsPaymentOpen(false)}
        initialTier={selectedTier}
      />
    </main>
  );
}
