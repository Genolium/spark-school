import React from "react";
import Link from "next/link";
import type { Metadata } from "next";
import { Header } from "@/components/Header";
import { Footer } from "@/components/Footer";
import { FileText, ArrowLeft, Shield, AlertCircle } from "lucide-react";

export const metadata: Metadata = {
  title: "Публичная оферта",
  description: "Официальная публичная оферта на оказание образовательно-консультационных услуг акселератора так называемый SPARK 2027.",
  alternates: {
    canonical: "https://so-called-spark.ru/offer",
  },
};

export default function OfferPage() {
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
          <div className="inline-flex items-center gap-2 px-3.5 py-1.5 rounded-full bento-pill text-xs font-mono text-amber-700 dark:text-amber-300 mb-4 border border-amber-500/20">
            <FileText className="w-3.5 h-3.5" />
            <span>Юридический документ</span>
          </div>
          <h1 className="font-editorial text-3xl sm:text-5xl font-bold text-slate-900 dark:text-white tracking-tight leading-tight">
            Публичная оферта <br />
            <span className="text-slate-500 dark:text-slate-400 text-2xl sm:text-3xl font-bold">
              на оказание образовательно-консультационных услуг
            </span>
          </h1>
          <p className="mt-3 text-xs sm:text-sm font-mono text-slate-500 dark:text-slate-400">
            Редакция от 1 сентября 2026 г. • Действует до отзыва или новой редакции
          </p>
        </div>

        {/* Executive Summary Card */}
        <div className="bento-card-dark p-6 sm:p-8 rounded-[32px] border border-emerald-500/15 dark:border-white/10 mb-10">
          <div className="flex items-start gap-4">
            <div className="w-10 h-10 rounded-2xl bg-amber-500/10 border border-amber-500/20 text-amber-600 dark:text-amber-400 flex items-center justify-center shrink-0">
              <Shield className="w-5 h-5" />
            </div>
            <div>
              <h2 className="font-editorial text-base font-bold text-slate-900 dark:text-white mb-1">
                Краткая суть для участника:
              </h2>
              <p className="text-xs sm:text-sm text-slate-600 dark:text-slate-300 leading-relaxed font-doc">
                Данный договор регулирует покупку доступа к программе проекта «так называемый SPARK» 2027 согласно выбранному тарифу. Оплата является полным и безоговорочным акцептом настоящей оферты. Услуга считается оказанной в полном объёме при добавлении Заказчика в закрытый Telegram-канал/чат и предоставлении 1 (одного) раунда обратной связи по материалам заявки. Всё дальнейшее менторское взаимодействие, последующие проверки и консультации предоставляются как бонус с нашей стороны и являются проявлением нашей доброй воли.
              </p>
            </div>
          </div>
        </div>

        {/* Document Body */}
        <div className="space-y-10 text-sm sm:text-base text-slate-600 dark:text-slate-300 leading-relaxed font-doc">
          
          {/* Section 1 */}
          <section className="bento-card-dark p-6 sm:p-8 rounded-[28px] border border-emerald-500/15 dark:border-white/10">
            <h2 className="text-lg sm:text-xl font-bold text-slate-900 dark:text-white mb-4 flex items-center gap-2">
              <span className="text-amber-600 dark:text-amber-400 font-mono text-sm">1.</span> Общие положения и стороны договора
            </h2>
            <div className="space-y-3 text-xs sm:text-sm text-slate-600 dark:text-slate-300">
              <p>
                1.1. Настоящий документ представляет собой официальное предложение (публичную оферту в соответствии с п. 2 ст. 437 Гражданского кодекса Российской Федерации) физического лица, применяющего специальный налоговый режим «Налог на профессиональный доход» — <strong className="text-slate-900 dark:text-white">Васюнина Ильи Олеговича</strong> (ИНН: 780739313219), именуемого в дальнейшем «<strong className="text-slate-900 dark:text-white">Исполнитель</strong>».
              </p>
              <p>
                1.2. Оферта адресована любому дееспособному физическому лицу (гражданину), именуемому в дальнейшем «<strong className="text-slate-900 dark:text-white">Заказчик</strong>», выразившему готовность воспользоваться услугами Исполнителя на условиях настоящего Договора.
              </p>
              <p>
                1.3. В соответствии с п. 3 ст. 438 ГК РФ оплата услуг Исполнителя является полным и безоговорочным акцептом настоящей публичной оферты. С момента поступления оплаты договор считается заключённым в письменной форме.
              </p>
            </div>
          </section>

          {/* Section 2 */}
          <section className="bento-card-dark p-6 sm:p-8 rounded-[28px] border border-emerald-500/15 dark:border-white/10">
            <h2 className="text-lg sm:text-xl font-bold text-slate-900 dark:text-white mb-4 flex items-center gap-2">
              <span className="text-amber-600 dark:text-amber-400 font-mono text-sm">2.</span> Предмет договора
            </h2>
            <div className="space-y-3 text-xs sm:text-sm text-slate-600 dark:text-slate-300">
              <p>
                2.1. Исполнитель обязуется оказать Заказчику комплекс информационно-консультационных услуг в формате онлайн-акселератора «<strong className="text-slate-900 dark:text-white">так называемый SPARK 2027</strong>», а Заказчик обязуется принять и оплатить данные услуги.
              </p>
              <p>
                2.2. Услуги оказываются в соответствии с выбранным Заказчиком тарифом, актуальные параметры которого публикуются на Сайте Исполнителя:
              </p>
              <ul className="list-disc pl-5 space-y-2 text-slate-600 dark:text-slate-300">
                <li>
                  <strong className="text-slate-900 dark:text-white">Тариф «Акселератор»:</strong> доступ в закрытый Telegram-канал со всеми обучающими материалами, видеоразборами и апдейтами отбора; доступ в закрытый Telegram-чат участников потока; персональный аудит материалов (эссе, резюме, видеовизитка); онлайн-симуляция собеседования (мок-интервью) в Zoom с так называемым Илем; визовый гайд J-1.
                </li>
                <li>
                  <strong className="text-slate-900 dark:text-white">Тариф «VIP»:</strong> всё из тарифа «Акселератор» + расширенные индивидуальные Zoom-симуляции интервью с детальным разбором + приоритетный личный чат с так называемым Илем до вылета в США + персональный контроль всех дедлайнов и вычитка полей заявки + рекомендации по подбору курсов и адаптации на кампусе.
                </li>
              </ul>
              <p>
                2.3. Стороны прямо и безоговорочно согласовали, что базовым существенным объёмом оплачиваемых услуг Исполнителя по настоящему Договору является:
              </p>
              <ul className="list-disc pl-5 space-y-1.5 text-slate-600 dark:text-slate-300">
                <li>предоставление Заказчику доступа (пригласительной ссылки) в закрытый Telegram-канал и/или чат проекта «так называемый SPARK» с размещёнными материалами;</li>
                <li>проведение 1 (одного) раунда персональной обратной связи (в виде письменного аудита драфта одного из материалов заявки либо консультационного ответа Исполнителя в Telegram).</li>
              </ul>
              <p>
                Любое последующее менторское взаимодействие, повторные вычитки текстов, дополнительные симуляции интервью, созвоны, консультации и координация дедлайнов предоставляются исключительно в качестве безвозмездного бонуса со стороны Исполнителя и являются проявлением доброй воли Исполнителя.
              </p>
            </div>
          </section>

          {/* Section 3 */}
          <section className="bento-card-dark p-6 sm:p-8 rounded-[28px] border border-emerald-500/15 dark:border-white/10">
            <h2 className="text-lg sm:text-xl font-bold text-slate-900 dark:text-white mb-4 flex items-center gap-2">
              <span className="text-amber-600 dark:text-amber-400 font-mono text-sm">3.</span> Стоимость услуг и порядок оплаты
            </h2>
            <div className="space-y-3 text-xs sm:text-sm text-slate-600 dark:text-slate-300">
              <p>
                3.1. Стоимость участия фиксируется на момент оформления заявки на Сайте в соответствии с утверждённой тарифной сеткой для выбранного тарифа («Акселератор» либо «VIP»). При применении промокода предоставляется скидка 5% от базовой стоимости выбранного тарифа. НДС не облагается в связи с применением Исполнителем налога на профессиональный доход (НПД) в соответствии с Федеральным законом от 27.11.2018 № 422-ФЗ.
              </p>
              <p>
                3.2. Оплата производится в рублях РФ в безналичном порядке через Систему быстрых платежей (СБП) или банковскими картами платёжных систем, действующих на территории РФ.
              </p>
              <p>
                3.3. После подтверждения поступления оплаты Исполнитель формирует электронный фискальный чек в мобильном приложении ФНС России «Мой налог» и направляет его Заказчику в Telegram или на указанную электронную почту.
              </p>
            </div>
          </section>

          {/* Section 4 */}
          <section className="bento-card-dark p-6 sm:p-8 rounded-[28px] border border-emerald-500/15 dark:border-white/10">
            <h2 className="text-lg sm:text-xl font-bold text-slate-900 dark:text-white mb-4 flex items-center gap-2">
              <span className="text-amber-600 dark:text-amber-400 font-mono text-sm">4.</span> Регламент, факт оказания услуг и бонусная программа
            </h2>
            <div className="space-y-3 text-xs sm:text-sm text-slate-600 dark:text-slate-300">
              <p>
                4.1. Доступ к закрытому Telegram-каналу и чату предоставляется Заказчику в течение 24 часов с момента подтверждения оплаты.
              </p>
              <p>
                4.2. Раунд обратной связи (персональный аудит драфта эссе, резюме или видеовизитки) выполняется Исполнителем в течение 3–5 рабочих дней с момента направления Заказчиком готового драфта в личный Telegram-чат куратора.
              </p>
              <p>
                4.3. Дата и время индивидуального Zoom-интервью согласовываются сторонами в Telegram индивидуально в рамках периода заявочной кампании 2026/2027 года.
              </p>
              <p>
                4.4. <strong className="text-slate-900 dark:text-white">Момент сдачи-приемки услуг:</strong> Услуга считается оказанной Исполнителем в полном объёме, надлежащего качества и безоговорочно принятой Заказчиком без составления дополнительного двустороннего акта приёма-передачи при наступлении совокупности следующих двух событий:
              </p>
              <ul className="list-disc pl-5 space-y-1 text-slate-600 dark:text-slate-300">
                <li>предоставление Заказчику ссылки для входа и добавление Заказчика в закрытый Telegram-канал и/или чат проекта «так называемый SPARK»;</li>
                <li>предоставление Заказчику 1 (одного) раунда обратной связи от Исполнителя (письменный аудит драфта материалов либо консультационный ответ на вопросы по заявке в Telegram).</li>
              </ul>
              <p>
                В случае, если Заказчик был добавлен в закрытый Telegram-канал/чат, но не направил материалы для обратной связи в течение 14 (четырнадцати) календарных дней с момента оплаты, услуги также признаются оказанными в полном объёме надлежащего качества, а обязательства Исполнителя — полностью исполненными.
              </p>
              <p>
                4.5. <strong className="text-slate-900 dark:text-white">Статус дальнейшего взаимодействия:</strong> Всё дальнейшее менторское взаимодействие Сторон после выполнения условий п. 4.4 настоящей Оферты (включая проведение индивидуальных Zoom-сессий, повторную вычитку документов, координацию сроков подачи заявки и визовое консультирование) осуществляется Исполнителем в качестве безвозмездного бонуса со стороны Исполнителя и является исключительно проявлением доброй воли Исполнителя. Невозможность проведения дополнительных сессий по обстоятельствам на стороне Заказчика (включая неявку на согласованные созвоны, непредоставление документов или пропуск дедлайнов) не влияет на факт полного и надлежащего оказания услуг и не может служить основанием для возврата денежных средств.
              </p>
            </div>
          </section>

          {/* Section 5 */}
          <section className="bento-card-dark p-6 sm:p-8 rounded-[28px] border border-emerald-500/15 dark:border-white/10">
            <h2 className="text-lg sm:text-xl font-bold text-slate-900 dark:text-white mb-4 flex items-center gap-2">
              <span className="text-amber-600 dark:text-amber-400 font-mono text-sm">5.</span> Интеллектуальная собственность
            </h2>
            <div className="space-y-3 text-xs sm:text-sm text-slate-600 dark:text-slate-300">
              <p>
                5.1. Все видеоматериалы, аудиозаписи, тексты, презентации, чеклисты и структуры материалов акселератора являются объектами интеллектуальной собственности Исполнителя и защищены нормами части IV ГК РФ.
              </p>
              <p>
                5.2. Заказчику предоставляется ограниченная неисключительная лицензия на личное использование материалов без права их распространения, публичного показа, передачи третьим лицам, перепродажи или публикации в складчинах и Telegram-каналах.
              </p>
              <p>
                5.3. Нарушение авторских прав влечёт ответственность в соответствии со ст. 1301 ГК РФ (взыскание компенсации до 5 000 000 рублей) и блокировку доступа к закрытому каналу и материалам без права возврата средств.
              </p>
            </div>
          </section>

          {/* Section 6 - Disclaimer */}
          <section className="bento-card-dark p-6 sm:p-8 rounded-[28px] border border-amber-500/30 bg-amber-500/[0.04]">
            <div className="flex items-center gap-2 text-amber-700 dark:text-amber-400 font-bold mb-3 text-base">
              <AlertCircle className="w-5 h-5 shrink-0" />
              <span>6. Официальный дисклеймер и отказ от гарантий результата</span>
            </div>
            <div className="space-y-3 text-xs sm:text-sm text-slate-600 dark:text-slate-300">
              <p>
                6.1. Проект <strong className="text-slate-900 dark:text-white">«так называемый SPARK»</strong> является полностью независимым частным образовательным проектом. Проект, сайт и автор не являются официальными представителями, агентами или аффилированными лицами программы SPARK, Американских Советов по международному образованию (American Councils for International Education) или Государственного департамента США.
              </p>
              <p>
                6.2. Исполнитель делится собственным успешным практическим опытом прохождения программы, методиками структурирования мыслей и языковой подготовки.
              </p>
              <p>
                6.3. Исполнитель <strong className="text-slate-900 dark:text-white">не гарантирует и не может гарантировать</strong> факт победы Заказчика в конкурсном отборе, получение финансового гранта или успешное прохождение визового собеседования в посольстве США. Решение о присуждении грантов принимается исключительно независимой отборочной комиссией на основе индивидуальных заслуг кандидата.
              </p>
            </div>
          </section>

          {/* Section 7 */}
          <section className="bento-card-dark p-6 sm:p-8 rounded-[28px] border border-emerald-500/15 dark:border-white/10">
            <h2 className="text-lg sm:text-xl font-bold text-slate-900 dark:text-white mb-4 flex items-center gap-2">
              <span className="text-amber-600 dark:text-amber-400 font-mono text-sm">7.</span> Порядок возврата денежных средств
            </h2>
            <div className="space-y-3 text-xs sm:text-sm text-slate-600 dark:text-slate-300">
              <p>
                7.1. В соответствии со ст. 782 ГК РФ и ст. 32 Закона РФ «О защите прав потребителей» Заказчик вправе в любой момент в одностороннем порядке отказаться от исполнения договора при условии возмещения Исполнителю фактически понесённых расходов (ФПР).
              </p>
              <p>
                7.2. В связи с тем, что услуга считается оказанной Исполнителем в полном объёме при добавлении Заказчика в закрытый Telegram-канал/чат и предоставлении 1 (одного) раунда обратной связи, возврат денежных средств после наступления указанных событий не производится. До момента предоставления 1 раунда обратной связи сумма возврата рассчитывается за вычетом фактически понесённых расходов Исполнителя, включая себестоимость предоставленного доступа к закрытой базе знаний и цифровым материалам канала (составляющую не менее 70% от стоимости тарифа), а также комиссий платёжных систем и банков.
              </p>
              <p>
                7.3. Требование о возврате направляется Заказчиком в свободной форме на электронную почту <code className="text-emerald-700 dark:text-emerald-300 font-mono bg-emerald-500/10 dark:bg-white/10 px-1.5 py-0.5 rounded">vas.ilyan@icloud.com</code> с указанием Telegram username, реквизитов платежа и банковского счёта для возврата. Срок рассмотрения заявления — 10 рабочих дней.
              </p>
            </div>
          </section>

          {/* Section 8 */}
          <section className="bento-card-dark p-6 sm:p-8 rounded-[28px] border border-emerald-500/15 dark:border-white/10">
            <h2 className="text-lg sm:text-xl font-bold text-slate-900 dark:text-white mb-4 flex items-center gap-2">
              <span className="text-amber-600 dark:text-amber-400 font-mono text-sm">8.</span> Реквизиты и контакты Исполнителя
            </h2>
            <div className="text-xs sm:text-sm font-mono text-slate-700 dark:text-slate-300 space-y-2 bg-slate-500/5 dark:bg-white/5 p-4 rounded-2xl border border-emerald-500/15 dark:border-white/10">
              <div><strong className="text-slate-900 dark:text-white">Исполнитель:</strong> Самозанятый Васюнин Илья Олегович</div>
              <div><strong className="text-slate-900 dark:text-white">ИНН:</strong> 780739313219</div>
              <div><strong className="text-slate-900 dark:text-white">Режим налогообложения:</strong> Налог на профессиональный доход (НПД)</div>
              <div><strong className="text-slate-900 dark:text-white">Email для обращений:</strong> vas.ilyan@icloud.com</div>
              <div><strong className="text-slate-900 dark:text-white">Telegram службы заботы:</strong> @ilyan_vas</div>
            </div>
          </section>

        </div>

      </div>

      <Footer />
    </main>
  );
}
