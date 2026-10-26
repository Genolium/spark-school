import React from "react";
import Link from "next/link";
import type { Metadata } from "next";
import { Header } from "@/components/Header";
import { Footer } from "@/components/Footer";
import { ArrowLeft, ArrowUpRight, Compass, GraduationCap, ShieldCheck } from "lucide-react";

export const metadata: Metadata = {
  title: "Принципы нашей работы",
  description:
    "Принципы проекта «так называемый SPARK»: самостоятельность участников, образовательный подход, честные рекомендации и независимость от организаторов программы SPARK.",
  alternates: {
    canonical: "https://so-called-spark.ru/principles",
  },
};

const PILLARS = [
  {
    icon: Compass,
    title: "Самостоятельность",
    text: "Только реальный опыт и собственные решения участника.",
  },
  {
    icon: GraduationCap,
    title: "Образование",
    text: "Методики, примеры и навыки, применимые за пределами одной программы.",
  },
  {
    icon: ShieldCheck,
    title: "Прозрачность",
    text: "Независимый проект с уважением к официальным правилам отбора.",
  },
];

const SKILLS = [
  "принципы структурирования академического и профессионального опыта;",
  "особенности англоязычного резюме;",
  "основы сторителлинга и письменной коммуникации;",
  "навыки самопрезентации и публичных выступлений;",
  "способы самостоятельного анализа собственных достижений.",
];

export default function PrinciplesPage() {
  return (
    <main className="min-h-screen bg-[var(--bg-base)] text-[var(--text-main)] relative selection:bg-emerald-500/20 selection:text-emerald-900 dark:selection:bg-white/20 dark:selection:text-white transition-colors duration-300">
      <Header />

      <div className="max-w-[1000px] mx-auto px-4 sm:px-6 pt-32 pb-20">
        <Link
          href="/"
          className="inline-flex items-center gap-2 text-xs font-mono text-slate-500 dark:text-slate-400 hover:text-slate-900 dark:hover:text-white transition-colors mb-8"
        >
          <ArrowLeft className="w-3.5 h-3.5" />
          <span>Вернуться на главную</span>
        </Link>

        <div className="mb-10">
          <div className="inline-flex items-center gap-2 px-3.5 py-1.5 rounded-full bento-pill text-xs font-mono text-emerald-700 dark:text-emerald-400 mb-4 border border-emerald-500/20">
            <Compass className="w-3.5 h-3.5" />
            <span>О проекте</span>
          </div>
          <h1 className="font-editorial text-3xl sm:text-5xl font-bold text-slate-900 dark:text-white tracking-tight leading-tight">
            Принципы нашей работы
          </h1>
          <p className="mt-5 font-editorial text-xl sm:text-2xl font-bold text-emerald-700 dark:text-emerald-400">
            Твой опыт. Твои идеи. Твоя история.
          </p>
          <p className="mt-3 max-w-2xl text-sm sm:text-base text-slate-600 dark:text-slate-300 font-doc leading-relaxed">
            Мы делимся знаниями и инструментами, которые помогают развивать навыки самостоятельной подготовки и
            самопрезентации.
          </p>
        </div>

        <div className="grid grid-cols-1 sm:grid-cols-3 gap-4 mb-10">
          {PILLARS.map((pillar) => (
            <div
              key={pillar.title}
              className="bento-card-dark p-5 rounded-2xl border border-emerald-500/15 dark:border-white/10 font-doc"
            >
              <pillar.icon className="w-5 h-5 text-emerald-600 dark:text-emerald-400 mb-2" />
              <p className="font-editorial text-sm font-bold text-slate-900 dark:text-white mb-1">{pillar.title}</p>
              <p className="text-xs text-slate-600 dark:text-slate-300 leading-relaxed">{pillar.text}</p>
            </div>
          ))}
        </div>

        <div className="space-y-8 text-sm sm:text-base text-slate-600 dark:text-slate-300 leading-relaxed font-doc">
          <section className="bento-card-dark p-6 sm:p-8 rounded-[28px] border border-emerald-500/15 dark:border-white/10">
            <p>
              <strong className="text-slate-900 dark:text-white">«так называемый SPARK»</strong> — независимый
              образовательный проект, основанный участником программы SPARK 2026.
            </p>
            <p className="mt-3">
              Мы делимся личным опытом участия в международной образовательной программе, знаниями об академической
              коммуникации, подготовке документов и навыками самопрезентации.
            </p>
          </section>

          <section className="bento-card-dark p-6 sm:p-8 rounded-[28px] border border-emerald-500/15 dark:border-white/10">
            <h2 className="text-lg sm:text-xl font-bold text-slate-900 dark:text-white mb-4">
              1. Самостоятельность участников
            </h2>
            <p>
              Мы убеждены, что качественная заявка должна отражать личность, реальный опыт и собственные идеи
              кандидата.
            </p>
            <p className="mt-3">
              Поэтому мы не подменяем участника при подготовке материалов. Не придумываем достижения, не создаём
              вымышленные истории и не гарантируем прохождение отбора.
            </p>
            <p className="mt-3">
              Все материалы заявки, включая резюме, эссе и видеовизитку, должны достоверно отражать опыт и решения
              самого кандидата.
            </p>
          </section>

          <section className="bento-card-dark p-6 sm:p-8 rounded-[28px] border border-emerald-500/15 dark:border-white/10">
            <h2 className="text-lg sm:text-xl font-bold text-slate-900 dark:text-white mb-4">
              2. Образовательный подход
            </h2>
            <p>
              Наши материалы посвящены развитию навыков, которые полезны не только при участии в SPARK, но и в
              дальнейшем образовании и профессиональной деятельности.
            </p>
            <p className="mt-3">В частности, мы рассматриваем:</p>
            <ul className="mt-3 space-y-2">
              {SKILLS.map((skill) => (
                <li key={skill} className="flex items-start gap-2.5">
                  <span className="mt-2 w-1.5 h-1.5 rounded-full bg-emerald-500 shrink-0" />
                  <span>{skill}</span>
                </li>
              ))}
            </ul>
          </section>

          <section className="bento-card-dark p-6 sm:p-8 rounded-[28px] border border-emerald-500/15 dark:border-white/10">
            <h2 className="text-lg sm:text-xl font-bold text-slate-900 dark:text-white mb-4">
              3. Рекомендации и ответственность
            </h2>
            <p>
              Любые образовательные примеры и рекомендации не заменяют официальных требований программы.
            </p>
            <p className="mt-3">
              Каждый участник самостоятельно принимает решения о содержании своих документов, несёт ответственность за
              достоверность предоставленной информации и обязан соблюдать актуальные правила отбора.
            </p>
          </section>

          <section className="bento-card-dark p-6 sm:p-8 rounded-[28px] border border-emerald-500/15 dark:border-white/10">
            <h2 className="text-lg sm:text-xl font-bold text-slate-900 dark:text-white mb-4">
              4. Соответствие требованиям SPARK
            </h2>
            <p>
              Мы не являемся официальными представителями программы SPARK и не связаны с её организаторами.
            </p>
          </section>
        </div>

        <div className="mt-10 flex flex-col sm:flex-row items-center justify-center gap-3">
          <Link
            href="/#pricing"
            className="btn-primary-mono w-full sm:w-auto px-6 py-3.5 text-xs sm:text-sm font-bold flex items-center justify-center gap-2"
          >
            <span>Посмотреть тарифы</span>
            <ArrowUpRight className="w-4 h-4" />
          </Link>
          <Link
            href="/offer"
            className="w-full sm:w-auto px-6 py-3.5 rounded-2xl bento-pill border border-emerald-500/25 text-xs sm:text-sm font-mono font-bold text-slate-700 dark:text-slate-200 flex items-center justify-center gap-2 hover:bg-emerald-500/10 transition-all"
          >
            <span>Публичная оферта</span>
          </Link>
        </div>
      </div>

      <Footer />
    </main>
  );
}
