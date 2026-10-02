import React from "react";
import Link from "next/link";
import type { Metadata } from "next";
import { Header } from "@/components/Header";
import { Footer } from "@/components/Footer";
import { ShieldCheck, ArrowLeft, Lock, Server } from "lucide-react";

export const metadata: Metadata = {
  title: "Политика конфиденциальности",
  description: "Политика обработки и защиты персональных данных участников акселератора так называемый SPARK 2027 в строгом соответствии с Федеральным законом № 152-ФЗ.",
  alternates: {
    canonical: "https://so-called-spark.ru/privacy",
  },
};

export default function PrivacyPage() {
  return (
    <main className="min-h-screen bg-[var(--bg-base)] text-[var(--text-main)] relative selection:bg-emerald-500/20 selection:text-emerald-900 dark:selection:bg-white/20 dark:selection:text-white transition-colors duration-300">
      <Header />

      <div className="max-w-[1000px] mx-auto px-4 sm:px-6 pt-32 pb-20">
        {/* Back Link */}
        <Link
          href="/"
          className="inline-flex items-center gap-2 text-xs font-mono text-slate-500 dark:text-slate-400 hover:text-slate-900 dark:hover:text-white transition-colors mb-8"
        >
          <ArrowLeft className="w-3.5 h-3.5" />
          <span>Вернуться на главную</span>
        </Link>

        {/* Page Title */}
        <div className="mb-12">
          <div className="inline-flex items-center gap-2 px-3.5 py-1.5 rounded-full bento-pill text-xs font-mono text-emerald-700 dark:text-emerald-400 mb-4 border border-emerald-500/20">
            <Lock className="w-3.5 h-3.5" />
            <span>152-ФЗ «О персональных данных»</span>
          </div>
          <h1 className="font-editorial text-3xl sm:text-5xl font-bold text-slate-900 dark:text-white tracking-tight leading-tight">
            Политика конфиденциальности <br />
            <span className="text-slate-500 dark:text-slate-400 text-2xl sm:text-3xl font-bold">
              и обработки персональных данных
            </span>
          </h1>
          <p className="mt-3 text-xs sm:text-sm font-mono text-slate-500 dark:text-slate-400">
            Редакция от 1 сентября 2026 г. • Действует в отношении всех пользователей сайта so-called-spark.ru
          </p>
        </div>

        {/* Key Privacy Highlights Bento */}
        <div className="grid grid-cols-1 sm:grid-cols-3 gap-4 mb-10">
          <div className="bento-card-dark p-5 rounded-2xl border border-emerald-500/15 dark:border-white/10 font-doc">
            <ShieldCheck className="w-5 h-5 text-emerald-600 dark:text-emerald-400 mb-2" />
            <h4 className="font-editorial text-sm font-bold text-slate-900 dark:text-white mb-1">Никакого спама</h4>
            <p className="text-xs text-slate-600 dark:text-slate-400">
              Ваш Telegram и email используются исключительно для предоставления доступа к закрытому Telegram-каналу и фискального чека.
            </p>
          </div>

          <div className="bento-card-dark p-5 rounded-2xl border border-emerald-500/15 dark:border-white/10 font-doc">
            <Lock className="w-5 h-5 text-sky-600 dark:text-sky-400 mb-2" />
            <h4 className="font-editorial text-sm font-bold text-slate-900 dark:text-white mb-1">Конфиденциальность эссе</h4>
            <p className="text-xs text-slate-600 dark:text-slate-400">
              Черновики ваших мотивационных эссе и резюме никогда не публикуются и не передаются третьим лицам.
            </p>
          </div>

          <div className="bento-card-dark p-5 rounded-2xl border border-emerald-500/15 dark:border-white/10 font-doc">
            <Server className="w-5 h-5 text-amber-600 dark:text-amber-400 mb-2" />
            <h4 className="font-editorial text-sm font-bold text-slate-900 dark:text-white mb-1">256-bit SSL</h4>
            <p className="text-xs text-slate-600 dark:text-slate-400">
              Все данные передаются по защищённому протоколу HTTPS с валидацией авторизации через Telegram Login Widget.
            </p>
          </div>
        </div>

        {/* Document Body */}
        <div className="space-y-10 text-sm sm:text-base text-slate-600 dark:text-slate-300 leading-relaxed font-doc">
          
          {/* Section 1 */}
          <section className="bento-card-dark p-6 sm:p-8 rounded-[28px] border border-emerald-500/15 dark:border-white/10">
            <h2 className="text-lg sm:text-xl font-bold text-slate-900 dark:text-white mb-4 flex items-center gap-2">
              <span className="text-emerald-600 dark:text-emerald-400 font-mono text-sm">1.</span> Общие положения и оператор данных
            </h2>
            <div className="space-y-3 text-xs sm:text-sm text-slate-600 dark:text-slate-300">
              <p>
                1.1. Настоящая Политика конфиденциальности определяет порядок обработки и защиты персональных данных пользователей сайта <code className="text-emerald-700 dark:text-emerald-300 font-mono bg-emerald-500/10 dark:bg-white/10 px-1.5 py-0.5 rounded">so-called-spark.ru</code> (далее — «Сайт») и участников проекта «так называемый SPARK».
              </p>
              <p>
                1.2. Оператором персональных данных является физическое лицо, применяющее специальный налоговый режим НПД: <strong className="text-slate-900 dark:text-white">Васюнин Илья Олегович</strong> (ИНН: 780739313219), email: <code className="text-emerald-700 dark:text-emerald-300 font-mono bg-emerald-500/10 dark:bg-white/10 px-1.5 py-0.5 rounded">vas.ilyan@icloud.com</code>.
              </p>
              <p>
                1.3. Использование функционала Сайта, отправка заявок или оплата услуг означает безоговорочное согласие Пользователя с настоящей Политикой и условиями обработки его персональных данных.
              </p>
            </div>
          </section>

          {/* Section 2 */}
          <section className="bento-card-dark p-6 sm:p-8 rounded-[28px] border border-emerald-500/15 dark:border-white/10">
            <h2 className="text-lg sm:text-xl font-bold text-slate-900 dark:text-white mb-4 flex items-center gap-2">
              <span className="text-emerald-600 dark:text-emerald-400 font-mono text-sm">2.</span> Перечень обрабатываемых персональных данных
            </h2>
            <div className="space-y-3 text-xs sm:text-sm text-slate-600 dark:text-slate-300">
              <p>
                Оператор обрабатывает следующие категории данных:
              </p>
              <ul className="list-disc pl-5 space-y-1.5 text-slate-600 dark:text-slate-300">
                <li><strong className="text-slate-900 dark:text-white">Идентификационные данные Telegram:</strong> имя, фамилия, Telegram username (@username), Telegram ID (при авторизации через Telegram Login Widget);</li>
                <li><strong className="text-slate-900 dark:text-white">Контактные данные:</strong> адрес электронной почты (email) для направления чеков об оплате;</li>
                <li><strong className="text-slate-900 dark:text-white">Материалы заявки:</strong> тексты и черновики резюме, мотивационных эссе, видеовизиток, добровольно направляемые студентом для аудита и консультаций;</li>
                <li><strong className="text-slate-900 dark:text-white">Технические данные:</strong> файлы cookies, IP-адрес, тип браузера, операционная система, реферальный идентификатор партнёра (параметр <code className="text-emerald-700 dark:text-emerald-300 font-mono bg-emerald-500/10 dark:bg-white/10 px-1.5 py-0.5 rounded">?ref=...</code>).</li>
              </ul>
            </div>
          </section>

          {/* Section 3 */}
          <section className="bento-card-dark p-6 sm:p-8 rounded-[28px] border border-emerald-500/15 dark:border-white/10">
            <h2 className="text-lg sm:text-xl font-bold text-slate-900 dark:text-white mb-4 flex items-center gap-2">
              <span className="text-emerald-600 dark:text-emerald-400 font-mono text-sm">3.</span> Цели сбора и обработки персональных данных
            </h2>
            <div className="space-y-3 text-xs sm:text-sm text-slate-600 dark:text-slate-300">
              <p>
                3.1. Персональные данные Пользователя обрабатываются исключительно в следующих законных целях:
              </p>
              <ul className="list-disc pl-5 space-y-1.5 text-slate-600 dark:text-slate-300">
                <li>Идентификация Пользователя и открытие доступа к закрытому Telegram-каналу, чату и материалам;</li>
                <li>Проведение персонального менторинга, аудита эссе и Zoom-сессий с автором;</li>
                <li>Формирование и отправка электронных фискальных чеков в соответствии с законодательством РФ о самозанятости;</li>
                <li>Начисление и учёт реферальных вознаграждений партнёрам по программе 15%;</li>
                <li>Оказание технической поддержки через официального Telegram-бота.</li>
              </ul>
            </div>
          </section>

          {/* Section 4 */}
          <section className="bento-card-dark p-6 sm:p-8 rounded-[28px] border border-emerald-500/15 dark:border-white/10">
            <h2 className="text-lg sm:text-xl font-bold text-slate-900 dark:text-white mb-4 flex items-center gap-2">
              <span className="text-emerald-600 dark:text-emerald-400 font-mono text-sm">4.</span> Принципы и безопасность хранения данных
            </h2>
            <div className="space-y-3 text-xs sm:text-sm text-slate-600 dark:text-slate-300">
              <p>
                4.1. Обработка персональных данных осуществляется на основе принципов законности, конфиденциальности и минимизации объёма собираемых данных.
              </p>
              <p>
                4.2. Оператор принимает необходимые организационные и технические меры для защиты персональных данных от неправомерного доступа, уничтожения, изменения или распространения, включая использование SSL-шифрования и проверку подлинности сессий по алгоритму HMAC-SHA256.
              </p>
              <p>
                4.3. Персональные данные хранятся на защищённых серверах на территории Российской Федерации.
              </p>
            </div>
          </section>

          {/* Section 5 */}
          <section className="bento-card-dark p-6 sm:p-8 rounded-[28px] border border-emerald-500/15 dark:border-white/10">
            <h2 className="text-lg sm:text-xl font-bold text-slate-900 dark:text-white mb-4 flex items-center gap-2">
              <span className="text-emerald-600 dark:text-emerald-400 font-mono text-sm">5.</span> Передача третьим лицам
            </h2>
            <div className="space-y-3 text-xs sm:text-sm text-slate-600 dark:text-slate-300">
              <p>
                5.1. Оператор <strong className="text-slate-900 dark:text-white">не передаёт</strong>, не продаёт и не предоставляет персональные данные третьим лицам в маркетинговых или коммерческих целях.
              </p>
              <p>
                5.2. Передача данных допускается исключительно:
              </p>
              <ul className="list-disc pl-5 space-y-1.5 text-slate-600 dark:text-slate-300">
                <li>Федеральной налоговой службе РФ в объёме, строго необходимом для формирования фискального чека («Мой налог»);</li>
                <li>Платёжным сервисам и банкам для проведения защищённого безналичного платежа;</li>
                <li>По требованию уполномоченных органов государственной власти в случаях, прямо предусмотренных действующим законодательством РФ.</li>
              </ul>
            </div>
          </section>

          {/* Section 6 */}
          <section className="bento-card-dark p-6 sm:p-8 rounded-[28px] border border-emerald-500/15 dark:border-white/10">
            <h2 className="text-lg sm:text-xl font-bold text-slate-900 dark:text-white mb-4 flex items-center gap-2">
              <span className="text-emerald-600 dark:text-emerald-400 font-mono text-sm">6.</span> Права пользователя и отзыв согласия
            </h2>
            <div className="space-y-3 text-xs sm:text-sm text-slate-600 dark:text-slate-300">
              <p>
                6.1. Пользователь имеет право на получение сведений об обработке его персональных данных, их уточнение, блокирование или уничтожение.
              </p>
              <p>
                6.2. Согласие на обработку персональных данных может быть отозвано Пользователем в любое время путём направления письменного уведомления с темой «Отзыв согласия на обработку ПД» на электронный адрес: <code className="text-emerald-700 dark:text-emerald-300 font-mono bg-emerald-500/10 dark:bg-white/10 px-1.5 py-0.5 rounded">vas.ilyan@icloud.com</code> либо куратору в Telegram: <code className="text-emerald-700 dark:text-emerald-300 font-mono bg-emerald-500/10 dark:bg-white/10 px-1.5 py-0.5 rounded">@ilyan_vas</code>.
              </p>
              <p>
                6.3. Оператор прекращает обработку данных в течение 10 рабочих дней с момента получения уведомления, за исключением сведений, подлежащих хранению в силу требований налогового законодательства РФ.
              </p>
            </div>
          </section>

          {/* Section 7 */}
          <section className="bento-card-dark p-6 sm:p-8 rounded-[28px] border border-emerald-500/15 dark:border-white/10">
            <h2 className="text-lg sm:text-xl font-bold text-slate-900 dark:text-white mb-4 flex items-center gap-2">
              <span className="text-emerald-600 dark:text-emerald-400 font-mono text-sm">7.</span> Контакты Оператора
            </h2>
            <div className="text-xs sm:text-sm font-mono text-slate-700 dark:text-slate-300 space-y-2 bg-slate-500/5 dark:bg-white/5 p-4 rounded-2xl border border-emerald-500/15 dark:border-white/10">
              <div><strong className="text-slate-900 dark:text-white">Оператор:</strong> Самозанятый Васюнин Илья Олегович</div>
              <div><strong className="text-slate-900 dark:text-white">ИНН:</strong> 780739313219</div>
              <div><strong className="text-slate-900 dark:text-white">Email:</strong> vas.ilyan@icloud.com</div>
              <div><strong className="text-slate-900 dark:text-white">Telegram:</strong> @ilyan_vas</div>
            </div>
          </section>

        </div>

      </div>

      <Footer />
    </main>
  );
}
