import type { Metadata, Viewport } from "next";
import Script from "next/script";
import localFont from "next/font/local";
import { IBM_Plex_Mono, Spectral, Golos_Text } from "next/font/google";
import "./globals.css";

// 1. Golos Text (ParaType / Google Fonts) — сверхжирный (Black 900 / ExtraBold 800) акцидентный гротеск для мощных заголовков
const golosHeading = Golos_Text({
  subsets: ["latin", "cyrillic"],
  weight: ["600", "700", "800", "900"],
  variable: "--font-heading",
  display: "swap",
});

// 2. PT Root UI (ParaType) — главный интерфейсный гротеск (переменный шрифт Variable Font)
const ptRootUI = localFont({
  src: "../../public/fonts/pt-root-ui-vf/PT-Root-UI_VF.ttf",
  variable: "--font-sans",
  display: "swap",
});

// 3. PT Astra Sans (ParaType) — официальный документный гротеск для регламентов, договоров и таблиц
const ptAstraSans = localFont({
  src: [
    {
      path: "../../public/fonts/pt-astra-sans/PT-Astra-Sans_Regular.ttf",
      weight: "400",
      style: "normal",
    },
    {
      path: "../../public/fonts/pt-astra-sans/PT-Astra-Sans_Bold.ttf",
      weight: "700",
      style: "normal",
    },
  ],
  variable: "--font-doc",
  display: "swap",
});

// 4. IBM Plex Mono (IBM) — моноширинный супершрифт для тегов, цифр, баллов и кодов (из статьи)
const ibmPlexMono = IBM_Plex_Mono({
  subsets: ["latin", "cyrillic"],
  weight: ["400", "500", "600", "700"],
  variable: "--font-mono",
  display: "swap",
});

// 5. Spectral (Production Type / type.today) — премиальная экранная антиква для эдиториал-акцентов (из статьи)
const spectral = Spectral({
  subsets: ["latin", "cyrillic"],
  weight: ["400", "500", "600", "700"],
  style: ["normal", "italic"],
  variable: "--font-serif",
  display: "swap",
});

export const viewport: Viewport = {
  width: "device-width",
  initialScale: 1,
  maximumScale: 5,
  themeColor: "#121316",
  colorScheme: "dark",
};

export const metadata: Metadata = {
  metadataBase: new URL("https://so-called-spark.ru"),
  title: {
    default: "так называемый SPARK — Грант $20,000 на учёбу в США | Подготовка к отбору 2027",
    template: "так называемый SPARK — %s",
  },
  description:
    "Практический проект от финалиста программы SPARK 2026 так называемого Иля. Разборы победных заявок, международное резюме американского формата, симуляция 45-минутного интервью в Zoom и визовый гайд J-1. Выиграй грант $20,000 на бесплатную учёбу в американском университете.",
  keywords: [
    "так называемый SPARK",
    "SPARK 2027",
    "грант SPARK",
    "гранты США для студентов",
    "обучение в Америке бесплатно",
    "подготовка к SPARK",
    "академический обмен США",
    "University of Wyoming",
    "резюме для гранта США",
    "эссе на грант США",
    "Zoom интервью США",
    "мок интервью на английском",
    "виза J-1 подготовка",
    "форма DS-160",
    "стипендии на учебу в США",
    "так называемый Иль",
  ],
  authors: [{ name: "так называемый Иль", url: "https://t.me/ilyan_vas" }],
  creator: "так называемый Иль (Финалист SPARK 2026)",
  publisher: "проект «так называемый SPARK»",
  robots: {
    index: true,
    follow: true,
    googleBot: {
      index: true,
      follow: true,
      "max-video-preview": -1,
      "max-image-preview": "large",
      "max-snippet": -1,
    },
  },
  alternates: {
    canonical: "https://so-called-spark.ru",
  },
  icons: {
    icon: [
      { url: "/favicon.ico", sizes: "any" },
      { url: "/icon.svg", type: "image/svg+xml" },
      { url: "/favicon-32x32.png", sizes: "32x32", type: "image/png" },
      { url: "/favicon-16x16.png", sizes: "16x16", type: "image/png" },
    ],
    apple: [
      { url: "/apple-touch-icon.png", sizes: "180x180", type: "image/png" },
    ],
  },
  openGraph: {
    type: "website",
    locale: "ru_RU",
    url: "https://so-called-spark.ru",
    siteName: "так называемый SPARK",
    title: "так называемый SPARK — Выиграй грант $20,000 на учёбу в США",
    description: "Практический проект от финалиста программы SPARK 2026 так называемого Иля: резюме по стандартам Гарварда, личный аудит документов и симуляция 45-мин интервью в Zoom.",
    images: [
      {
        url: "/og-image.png",
        width: 1200,
        height: 630,
        alt: "так называемый SPARK — Выиграй грант $20,000 на учёбу в США",
        type: "image/png",
      },
    ],
  },
  twitter: {
    card: "summary_large_image",
    title: "так называемый SPARK — Выиграй грант $20,000 на учёбу в США",
    description: "Практический проект от финалиста программы SPARK 2026 так называемого Иля: резюме по стандартам Гарварда, личный аудит документов и симуляция 45-мин интервью в Zoom.",
    images: ["/og-image.png"],
    creator: "@ilyan_vas",
  },
  verification: {
    yandex: "345a2f8dfaaff053",
  },
  category: "education",
};

const jsonLd = {
  "@context": "https://schema.org",
  "@graph": [
    {
      "@type": "WebSite",
      "@id": "https://so-called-spark.ru/#website",
      "url": "https://so-called-spark.ru",
      "name": "так называемый SPARK",
      "description": "Практический проект подготовки к гранту $20,000 на учёбу в США летом 2027 года.",
      "inLanguage": "ru"
    },
    {
      "@type": "BreadcrumbList",
      "@id": "https://so-called-spark.ru/#breadcrumbs",
      "itemListElement": [
        {
          "@type": "ListItem",
          "position": 1,
          "name": "Главная",
          "item": "https://so-called-spark.ru"
        },
        {
          "@type": "ListItem",
          "position": 2,
          "name": "Тарифы",
          "item": "https://so-called-spark.ru/#pricing"
        },
        {
          "@type": "ListItem",
          "position": 3,
          "name": "Публичная оферта",
          "item": "https://so-called-spark.ru/offer"
        }
      ]
    },
    {
      "@type": "Course",
      "@id": "https://so-called-spark.ru/#course",
      "name": "проект «так называемый SPARK» 2027",
      "description": "Практический проект от финалиста программы SPARK 2026 так называемого Иля. Разборы победных заявок, шаблоны документов, симуляция 45-минутного интервью в Zoom.",
      "courseMode": "online",
      "provider": {
        "@type": "Organization",
        "name": "так называемый SPARK",
        "url": "https://so-called-spark.ru"
      },
      "offers": [
        {
          "@type": "Offer",
          "name": "Акселератор",
          "price": "6900",
          "priceCurrency": "RUB",
          "availability": "https://schema.org/InStock",
          "url": "https://so-called-spark.ru/#pricing",
          "validFrom": "2026-09-01"
        },
        {
          "@type": "Offer",
          "name": "VIP",
          "price": "14900",
          "priceCurrency": "RUB",
          "availability": "https://schema.org/InStock",
          "url": "https://so-called-spark.ru/#pricing",
          "validFrom": "2026-09-01"
        }
      ],
      "instructor": {
        "@type": "Person",
        "name": "так называемый Иль",
        "jobTitle": "Финалист программы SPARK 2026",
        "sameAs": "https://t.me/ilyan_vas"
      },
      "inLanguage": "ru",
      "educationalCredentialAwarded": "Готовый пакет документов для гранта $20,000"
    },
    {
      "@type": "EducationalOrganization",
      "@id": "https://so-called-spark.ru/#organization",
      "name": "так называемый SPARK",
      "url": "https://so-called-spark.ru",
      "logo": "https://so-called-spark.ru/icon.svg",
      "founder": {
        "@type": "Person",
        "name": "так называемый Иль"
      }
    },
    {
      "@type": "FAQPage",
      "@id": "https://so-called-spark.ru/#faq",
      "mainEntity": [
        {
          "@type": "Question",
          "name": "Какой уровень английского языка нужен для победы?",
          "acceptedAnswer": {
            "@type": "Answer",
            "text": "Для уверенной победы в отборе рекомендуется уровень B1-B2."
          }
        },
        {
          "@type": "Question",
          "name": "Что покрывает грант SPARK $20,000?",
          "acceptedAnswer": {
            "@type": "Answer",
            "text": "Грант Госдепартамента США покрывает обучение в американском университете, проживание на кампусе, трехразовое питание, перелет в обе стороны, страховку и стипендию."
          }
        },
        {
          "@type": "Question",
          "name": "Кто проводит проверку эссе и мок-интервью?",
          "acceptedAnswer": {
            "@type": "Answer",
            "text": "Личный аудит документов и 45-минутную симуляцию интервью в Zoom проводит лично так называемый Иль — финалист SPARK 2026."
          }
        }
      ]
    }
  ]
};

import { AuthProvider } from "@/context/AuthContext";
import { ThemeProvider } from "@/context/ThemeContext";
import { ReferralTracker } from "@/components/ReferralTracker";
import { YandexMetrikaTracker } from "@/components/YandexMetrikaTracker";

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html
      lang="ru"
      suppressHydrationWarning
      className={`scroll-smooth ${golosHeading.variable} ${ptRootUI.variable} ${ptAstraSans.variable} ${ibmPlexMono.variable} ${spectral.variable}`}
    >
      <head>
        <meta name="yandex-verification" content="345a2f8dfaaff053" />
        <script
          dangerouslySetInnerHTML={{
            __html: `
              (function() {
                try {
                  var saved = localStorage.getItem('spark_theme');
                  var prefersDark = window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches;
                  
                  if (saved === 'dark' || (!saved && prefersDark)) {
                    document.documentElement.classList.add('dark');
                  } else {
                    document.documentElement.classList.remove('dark');
                  }
                } catch (e) {}
              })();
            `,
          }}
        />
        <script
          type="application/ld+json"
          dangerouslySetInnerHTML={{ __html: JSON.stringify(jsonLd) }}
        />
      </head>
      <body className="bg-[var(--bg-base)] text-[var(--text-main)] antialiased font-sans selection:bg-emerald-500/20 selection:text-emerald-900 dark:selection:bg-white/20 dark:selection:text-white transition-colors duration-300">
        {/* Yandex.Metrika counter */}
        <Script
          id="yandex-metrika"
          strategy="lazyOnload"
          dangerouslySetInnerHTML={{
            __html: `
              (function(m,e,t,r,i,k,a){
                  m[i]=m[i]||function(){(m[i].a=m[i].a||[]).push(arguments)};
                  m[i].l=1*new Date();
                  for (var j = 0; j < document.scripts.length; j++) {if (document.scripts[j].src === r) { return; }}
                  k=e.createElement(t),a=e.getElementsByTagName(t)[0],k.async=1,k.src=r,a.parentNode.insertBefore(k,a)
              })(window, document,'script','https://mc.yandex.ru/metrika/tag.js?id=113185946', 'ym');

              ym(113185946, 'init', {ssr:true, webvisor:false, clickmap:true, ecommerce:"dataLayer", referrer: document.referrer, url: location.href, accurateTrackBounce:true, trackLinks:true});
            `,
          }}
        />
        <noscript>
          <div>
            <img
              src="https://mc.yandex.ru/watch/113185946"
              style={{ position: "absolute", left: "-9999px" }}
              alt=""
              width="1"
              height="1"
            />
          </div>
        </noscript>
        {/* /Yandex.Metrika counter */}
        <ThemeProvider>
          <AuthProvider>
            <ReferralTracker />
            <YandexMetrikaTracker />
            {children}
          </AuthProvider>
        </ThemeProvider>
      </body>
    </html>
  );
}
