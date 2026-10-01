import type { Metadata } from "next";

export const metadata: Metadata = {
  title: "Партнёрская программа (до 2 235 ₽ с заявки) | проект «так называемый SPARK»",
  description: "Зарабатывай 15% с каждой рекомендации и 5% со второго уровня в партнёрской программе проекта «так называемый SPARK». Выплаты на карты РФ и СБП.",
  alternates: {
    canonical: "https://so-called-spark.ru/partner",
  },
  openGraph: {
    title: "Партнёрская сеть проекта «так называемый SPARK»",
    description: "Двухуровневая реферальная программа: 15% прямой бонус + 5% со 2-го уровня. Моментальный вывод через Telegram.",
    url: "https://so-called-spark.ru/partner",
    siteName: "проект «так называемый SPARK»",
    locale: "ru_RU",
    type: "website",
  },
};

export default function PartnerLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return <>{children}</>;
}
