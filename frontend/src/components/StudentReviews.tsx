"use client";

import React, { useCallback, useRef, useState } from "react";
import { ChevronLeft, ChevronRight, Quote } from "lucide-react";

const REVIEWS = [
  {
    name: "Mary",
    text: `Курс действительно сделан продуманно и качественно. Смотреть видео и читать посты безумно интересно. Они действительно помогают подробно разобраться в требованиях программы ✨spark ✨, посмотреть на собственный опыт под нужным углом.

Мне кажется, что такой курс будет полезен не только прямо сейчас для подготовки к этой программе, но и в будущем для учёбы и других проектов обмена.`,
  },
  {
    name: "i’m done",
    text: `Впервые беру подобный курс, поэтому я считаю курс Ильи своим первым опытом. На этапе подготовки и сбора документов возникало куча вопросов, ответы на которые довольно-таки трудно найти, но именно курс Ильи помог мне с большинством из них.

Довольно интересен текстовый формат курса, что поначалу оказалось трудным, но как только я внимательно начала вчитываться в материал, так сразу поняла, о чём именно идёт речь. Также безумно нравится практика прямого разговора с Ильёй, ведь это отличный способ получить много важных подсказок, которые непременно помогут при составлении заявки. В текущий момент курс меня полностью устраивает. 🌸`,
  },
  {
    name: "ulia",
    text: `Я очень хочу поблагодарить Иля за такой понятный и содержательный курс 🙏🏻 Как жаль, что его не было в прошлом году...

Мне очень понравилось то, что он всё объясняет простыми словами и помогает с любыми вопросами, интересуется, как дела с заявкой, и т. д. Я прям чувствую его поддержку, и это очень помогает. Сейчас я на этапе написания эссе и могу сказать, что это лучшее эссе, которое я только могла написать. Если вы первый раз подаёте заявку, обязательно пройдите этот курс.`,
  },
  {
    name: "Свят",
    text: `Курс очень полезный, много качественной и интересной информации, всё чётко и подробно. Посты помогают находить старт и отталкиваться от этого.

Чуть-чуть тяжеловато искать информацию, так как слишком большой её поток, но это всё субъективно. Возможно, стоит делить по блокам.`,
  },
  {
    name: "Анюта",
    text: `Так, ну я курс всё ещё прохожу, но уже сейчас могу сказать, что мне очень нравится твоя обратная связь. Она прям очень подробная, всё по делу, все вопросы детально тобой прорабатываются. Это очень круто, так как я, например, не видела некоторых недочётов в резюме, пока ты не указал на них. Прочитав твою обратную связь, я подумала: «блин, а ведь реально это лучше сделать так, а вот это так».`,
  },
];

export function StudentReviews() {
  const [index, setIndex] = useState(0);
  const touchStartX = useRef<number | null>(null);
  const current = REVIEWS[index];

  const go = useCallback((direction: 1 | -1) => {
    setIndex((prev) => (prev + direction + REVIEWS.length) % REVIEWS.length);
  }, []);

  const onKeyDown = (event: React.KeyboardEvent<HTMLDivElement>) => {
    if (event.key === "ArrowRight") {
      event.preventDefault();
      go(1);
    } else if (event.key === "ArrowLeft") {
      event.preventDefault();
      go(-1);
    }
  };

  const onTouchStart = (event: React.TouchEvent) => {
    touchStartX.current = event.touches[0]?.clientX ?? null;
  };

  const onTouchEnd = (event: React.TouchEvent) => {
    const start = touchStartX.current;
    const end = event.changedTouches[0]?.clientX ?? null;
    touchStartX.current = null;
    if (start == null || end == null) return;
    const delta = end - start;
    if (Math.abs(delta) < 45) return;
    go(delta < 0 ? 1 : -1);
  };

  return (
    <section id="reviews" className="w-full max-w-[1520px] mx-auto px-3 sm:px-6 py-16 sm:py-20 scroll-mt-20 sm:scroll-mt-24">
      <div className="flex flex-col md:flex-row md:items-end justify-between gap-6 mb-8">
        <div className="max-w-2xl">
          <div className="inline-flex items-center gap-2 px-3.5 py-1.5 rounded-full bento-pill text-xs font-mono text-emerald-800 dark:text-emerald-300 mb-4">
            <span className="w-2 h-2 rounded-full bg-emerald-500" />
            <span>Отзывы участников</span>
          </div>
          <h2 className="font-editorial text-3xl sm:text-4xl lg:text-5xl font-bold tracking-tight text-slate-900 dark:text-white leading-[1.1]">
            Что говорят те,
            <br />
            кто уже изучает курс
          </h2>
          <p className="mt-3 text-sm text-slate-600 dark:text-slate-300 leading-relaxed max-w-xl">
            Пять честных отзывов о курсе, обратной связи и подготовке к заявке.
          </p>
        </div>

        <div className="flex items-center gap-2 shrink-0">
          <button
            type="button"
            onClick={() => go(-1)}
            aria-label="Предыдущий отзыв"
            className="w-11 h-11 rounded-full bento-pill flex items-center justify-center text-slate-700 dark:text-slate-200 hover:text-emerald-700 dark:hover:text-white transition-colors"
          >
            <ChevronLeft className="w-5 h-5" />
          </button>
          <button
            type="button"
            onClick={() => go(1)}
            aria-label="Следующий отзыв"
            className="w-11 h-11 rounded-full bento-pill flex items-center justify-center text-slate-700 dark:text-slate-200 hover:text-emerald-700 dark:hover:text-white transition-colors"
          >
            <ChevronRight className="w-5 h-5" />
          </button>
        </div>
      </div>

      <div
        role="region"
        aria-roledescription="карусель"
        aria-label="Отзывы участников курса"
        tabIndex={0}
        onKeyDown={onKeyDown}
        onTouchStart={onTouchStart}
        onTouchEnd={onTouchEnd}
        className="rounded-3xl border border-emerald-500/20 dark:border-white/15 bg-white/70 dark:bg-white/[0.04] p-6 sm:p-9 focus:outline-none focus-visible:ring-2 focus-visible:ring-emerald-500/60"
      >
        <figure aria-live="polite" className="min-h-[280px] sm:min-h-[250px] flex flex-col justify-between">
          <div>
            <Quote
              aria-hidden="true"
              className="w-7 h-7 text-emerald-600/40 dark:text-emerald-400/35 mb-4"
              strokeWidth={1.5}
            />
            <blockquote className="font-doc text-base sm:text-lg text-slate-700 dark:text-slate-200 leading-relaxed whitespace-pre-line max-w-3xl">
              {current.text}
            </blockquote>
          </div>
          <figcaption className="mt-6 pt-4 border-t border-emerald-500/15 dark:border-white/10 flex items-center justify-between gap-4">
            <span className="text-sm font-semibold text-slate-900 dark:text-white">{current.name}</span>
            <span className="text-[11px] font-mono text-slate-500 dark:text-slate-400">
              {index + 1} / {REVIEWS.length}
            </span>
          </figcaption>
        </figure>
      </div>

      <div className="flex items-center justify-center gap-2 mt-5" aria-label="Выбор отзыва">
        {REVIEWS.map((review, i) => (
          <button
            key={review.name}
            type="button"
            aria-current={i === index ? "true" : undefined}
            aria-label={`Отзыв ${i + 1}: ${review.name}`}
            onClick={() => setIndex(i)}
            className={`h-2 rounded-full transition-all ${
              i === index ? "w-7 bg-emerald-500" : "w-2 bg-slate-300 dark:bg-white/25 hover:bg-emerald-400"
            }`}
          />
        ))}
      </div>
    </section>
  );
}
