"use client";

import React from "react";
import { BookOpen, PlayCircle } from "lucide-react";

const LESSON_VIDEOS = [
  {
    id: process.env.NEXT_PUBLIC_KINESCOPE_LESSON1_ID || "0S9sBwRj9ebLNJPugiWnJ3",
    title: "Урок 1. Вводный",
  },
  {
    id: process.env.NEXT_PUBLIC_KINESCOPE_LESSON2_ID || "nbkdHzXwkkaXNzNM44nYMf",
    title: "Урок 2. Поиск центральной идеи",
  },
];

// Accept either a bare Kinescope video ID or a pasted embed URL.
function kinescopeId(value: string) {
  const trimmed = value.trim().replace(/\/+$/, "");
  if (!trimmed.includes("/")) return trimmed;
  return trimmed.split("/").filter(Boolean).pop() || "";
}

function KinescopeVideo({ videoId: rawVideoId, title }: { videoId: string; title: string }) {
  const videoId = kinescopeId(rawVideoId);
  if (!videoId) return null;

  return (
    <div className="rounded-2xl overflow-hidden border border-slate-200 dark:border-white/15 bg-black shadow-lg">
      <div className="relative aspect-video">
        <iframe
          src={`https://kinescope.io/embed/${videoId}`}
          title={title}
          loading="lazy"
          allow="autoplay; fullscreen; picture-in-picture; encrypted-media"
          allowFullScreen
          className="absolute inset-0 h-full w-full"
        />
      </div>
    </div>
  );
}

export function DemoLessonSection() {
  const videos = LESSON_VIDEOS.filter((lesson) => kinescopeId(lesson.id));
  if (videos.length === 0) return null;

  return (
    <section id="demo-lesson" className="w-full max-w-[1520px] mx-auto px-4 sm:px-6 py-16 scroll-mt-20">
      <div className="bento-card-dark p-6 sm:p-10 lg:p-12 rounded-[32px] border border-slate-200 dark:border-white/15 bg-white/95 dark:bg-[#12161f]/95 shadow-xl relative overflow-hidden backdrop-blur-xl">
        <div className="absolute top-0 right-1/4 w-96 h-48 bg-emerald-500/10 rounded-full blur-3xl pointer-events-none" />

        <div className="max-w-3xl mb-8">
          <div className="inline-flex items-center gap-2 px-3.5 py-1.5 rounded-full bento-pill text-xs font-mono text-emerald-800 dark:text-emerald-300 mb-3">
            <BookOpen className="w-3.5 h-3.5 text-emerald-600 dark:text-emerald-400" />
            <span>Открытые уроки курса</span>
          </div>
          <h2 className="font-editorial text-2xl sm:text-4xl font-bold tracking-tight text-slate-900 dark:text-white">
            Посмотрите уроки изнутри
          </h2>
          <p className="mt-2 text-xs sm:text-sm text-slate-600 dark:text-slate-300 font-doc leading-relaxed">
            Два первых урока целиком — без вырезок и пересказа.
          </p>
        </div>

        <div className="space-y-6">
          {videos.map((lesson) => (
            <div key={lesson.id} className="space-y-2.5">
              <div className="flex items-center gap-2 text-xs font-mono font-bold uppercase tracking-wider text-slate-500 dark:text-slate-400">
                <PlayCircle className="w-4 h-4 text-emerald-500" />
                <span>{lesson.title}</span>
              </div>
              <KinescopeVideo videoId={lesson.id} title={lesson.title} />
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}
