"use client";

import React, { useState, useEffect } from "react";
import dynamic from "next/dynamic";
import { Header } from "@/components/Header";
import { Hero } from "@/components/Hero";
import { SocialProofStrip } from "@/components/SocialProofStrip";
import { HardTruth } from "@/components/HardTruth";
import { KibiRoadmap } from "@/components/KibiRoadmap";
import { PlatformFeatures } from "@/components/PlatformFeatures";
import { AuthorSection } from "@/components/AuthorSection";
import { PricingSection } from "@/components/PricingSection";
import { QualificationSection } from "@/components/QualificationSection";
import { FaqSection } from "@/components/FaqSection";
import { Footer } from "@/components/Footer";
import { RecoveryBanner } from "@/components/RecoveryBanner";
import { DemoLessonSection } from "@/components/DemoLessonSection";
import { StudentReviews } from "@/components/StudentReviews";
import { trackFunnelStep, buildCheckoutStartParam, BOT_URL } from "@/lib/analytics";
import { syncServerExperiments } from "@/lib/experiments";

const PlacementMap = dynamic(
  () => import("@/components/PlacementMap").then((mod) => mod.PlacementMap),
  { ssr: false }
);
const ChancesModal = dynamic(
  () => import("@/components/Modals/ChancesModal").then((mod) => mod.ChancesModal)
);

export default function HomePage() {
  const [isChancesOpen, setIsChancesOpen] = useState(false);

  const handleOpenPayment = (tier: "basic" | "accelerator" | "vip" = "accelerator") => {
    trackFunnelStep("bot_handoff", { tier });
    let refCode = "";
    if (typeof window !== "undefined") {
      try {
        const urlParam = new URLSearchParams(window.location.search).get("ref");
        if (urlParam) {
          refCode = urlParam.trim();
        } else {
          const stored = localStorage.getItem("spark_ref");
          if (stored) refCode = stored.trim();
        }
      } catch {
        // Safe fallback
      }
    }
    const startParam = buildCheckoutStartParam(tier, refCode);
    window.location.href = `${BOT_URL}?start=${startParam}`;
  };

  useEffect(() => {
    // Server is the canonical source of A/B buckets.
    syncServerExperiments();
    trackFunnelStep("landing_view");
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

      {/* Screen 2: Hard Truth (typical first-round mistakes) */}
      <HardTruth />

      {/* Screen 3: The Journey (Kibi-Style Curved Flight Path) */}
      <KibiRoadmap />

      {/* Screen 4: Verified Placements Map (24 Past Cohort Campuses) */}
      <PlacementMap />

      {/* Screen 5: Product Features & Bento Architecture */}
      <PlatformFeatures />

      {/* Screen 6: The Author (так называемый Иль Story & J-1 Experience) */}
      <AuthorSection />

      {/* Screen 6.5: Demo Lesson & Curator Feedback Preview */}
      <DemoLessonSection />

      {/* Screen 6.7: Student testimonials */}
      <StudentReviews />

      {/* Screen 7: 3-Tier Pricing Matrix & Scarcity */}
      <PricingSection onOpenPayment={handleOpenPayment} />

      {/* Screen 8: Qualification Filter (Кому курс НЕ подойдет / Для кого) */}
      <QualificationSection onSelectTier={() => handleOpenPayment("accelerator")} />

      {/* Screen 9: FAQ Accordions */}
      <FaqSection />

      {/* Screen 9: Legal Footer & Disclaimer */}
      <Footer />

      {/* Interactive Chances Evaluation Modal */}
      <ChancesModal
        isOpen={isChancesOpen}
        onClose={() => setIsChancesOpen(false)}
      />

      <RecoveryBanner onOpenDiagnostic={() => setIsChancesOpen(true)} />

      {/* Canonical checkout: direct Telegram deep link with attribution token.
          The unused web PaymentModal is intentionally not mounted. */}
    </main>
  );
}
