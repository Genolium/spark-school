import type { Metadata } from "next";

export const metadata: Metadata = {
  title: "Персональный план подготовки — SPARK",
  robots: { index: false, follow: false },
};

export default function PlanLayout({ children }: { children: React.ReactNode }) {
  return children;
}
