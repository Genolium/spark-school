"use client";

import React, { useState, useMemo, useEffect } from "react";
import { motion, AnimatePresence } from "framer-motion";
import { MapPin, Search, GraduationCap, Building2, ShieldCheck, Check } from "lucide-react";
import mapData from "@/data/us-map-data.json";

interface Placement {
  id: number;
  name: string;
  state: string;
  city: string;
  lat: number;
  lng: number;
  x: number;
  y: number;
  type: "University" | "Community College";
  region: "Mountain" | "West" | "Midwest" | "South" | "East";
  regionLabel: string;
  highlight: string;
  isAuthorCampus?: boolean;
}

const ROTATION_CAMPUSES = [
  { id: 9, name: "Green River College" },
  { id: 13, name: "Lake Tahoe Community College" },
  { id: 14, name: "California State University – San Marcos" },
  { id: 1, name: "University of Wyoming" },
  { id: 16, name: "University of Missouri (Mizzou)" },
  { id: 20, name: "University of Tampa" },
  { id: 23, name: "Lehigh University" },
];

export function PlacementMap() {
  const placements = mapData.placements as Placement[];
  const [activeRegion, setActiveRegion] = useState<string>("All");

  const rotationPlacements = useMemo(() => {
    return ROTATION_CAMPUSES.map((target) =>
      placements.find(
        (p) => p.id === target.id || p.name.toLowerCase().includes(target.name.toLowerCase())
      )
    ).filter((p): p is Placement => Boolean(p));
  }, [placements]);

  const [isAutoRotating, setIsAutoRotating] = useState(true);
  const [selectedPlacement, setSelectedPlacement] = useState<Placement>(() => {
    return rotationPlacements[0] || placements.find((p) => p.isAuthorCampus) || placements[0];
  });
  const [hoveredPlacement, setHoveredPlacement] = useState<Placement | null>(null);
  const [searchQuery, setSearchQuery] = useState("");

  const stopAutoRotation = () => {
    setIsAutoRotating(false);
  };

  useEffect(() => {
    if (!isAutoRotating || rotationPlacements.length === 0) return;

    const timer = setInterval(() => {
      if (typeof document !== "undefined" && document.hidden) return;

      setSelectedPlacement((current) => {
        const currentIndex = rotationPlacements.findIndex((p) => p.id === current.id);
        const nextIndex = (currentIndex + 1) % rotationPlacements.length;
        return rotationPlacements[nextIndex];
      });
    }, 5000);

    return () => clearInterval(timer);
  }, [isAutoRotating, rotationPlacements]);

  const filteredPlacements = useMemo(() => {
    return placements.filter((p) => {
      const matchesRegion = activeRegion === "All" || p.region === activeRegion;
      const matchesSearch =
        p.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
        p.state.toLowerCase().includes(searchQuery.toLowerCase()) ||
        p.city.toLowerCase().includes(searchQuery.toLowerCase());
      return matchesRegion && matchesSearch;
    });
  }, [placements, activeRegion, searchQuery]);

  const regionTabs = useMemo(() => {
    const total = placements.length;
    const countFor = (region: string) => placements.filter((p) => p.region === region).length;
    return [
      { key: "All", label: `Все кампусы (${total})` },
      { key: "Mountain", label: `Mountain (${countFor("Mountain")})` },
      { key: "West", label: `West Coast (${countFor("West")})` },
      { key: "Midwest", label: `Midwest (${countFor("Midwest")})` },
      { key: "South", label: `South (${countFor("South")})` },
      { key: "East", label: `East Coast (${countFor("East")})` },
    ];
  }, [placements]);

  return (
    <section id="map" className="relative w-full max-w-[1520px] mx-auto px-3 sm:px-6 py-16 sm:py-24 scroll-mt-20 sm:scroll-mt-24">
      {/* Section Header */}
      <div className="flex flex-col lg:flex-row lg:items-end justify-between mb-8 sm:mb-12 gap-6">
        <div className="max-w-3xl">
          <h2 className="font-editorial text-3xl sm:text-5xl lg:text-6xl font-bold tracking-tight text-slate-900 dark:text-white leading-[1.1]">
            География распределения <br />
            <span className="text-emerald-600 dark:text-emerald-400 font-serif italic">плейсментов прошлых лет</span>
          </h2>
          <p className="mt-4 text-sm sm:text-base text-slate-600 dark:text-slate-300 leading-relaxed font-normal">
            Интерактивная карта университетов и комьюнити-колледжей по всей территории США, куда распределяли участников программы в прошлые годы. Кликай на пины для просмотра деталей.
          </p>
        </div>

        {/* Mandatory Transparency Disclaimer */}
        <div className="bento-card-dark p-4 rounded-xl max-w-md border border-emerald-500/20 dark:border-white/10 text-xs font-doc text-slate-600 dark:text-slate-400 leading-relaxed">
          <div className="flex items-center gap-1.5 text-amber-700 dark:text-amber-300 font-medium mb-1 font-mono">
            <ShieldCheck className="w-4 h-4" />
            <span>Статус: Кампусы прошлых потоков</span>
          </div>
          Статистика распределений участников предыдущих потоков программы SPARK. Итоговое распределение нового потока определяется исключительно Госдепартаментом США.
        </div>
      </div>

      {/* Filter Bar & Search */}
      <div className="flex flex-col md:flex-row items-stretch md:items-center justify-between gap-4 mb-6">
        {/* Region Pills */}
        <div className="flex items-center gap-2 overflow-x-auto pb-2 md:pb-0 scrollbar-none">
          {regionTabs.map((tab) => (
            <button
              key={tab.key}
              onClick={() => {
                stopAutoRotation();
                setActiveRegion(tab.key);
              }}
              className={`px-4 py-2 rounded-full text-xs font-mono whitespace-nowrap transition-all ${
                activeRegion === tab.key
                  ? "bg-emerald-600 text-white font-bold shadow-md dark:bg-white dark:text-[#121316]"
                  : "bento-pill text-slate-700 dark:text-slate-300 hover:text-emerald-800 dark:hover:text-white"
              }`}
            >
              {tab.label}
            </button>
          ))}
        </div>

        {/* Search Input */}
        <div className="relative min-w-[260px]">
          <Search className="w-4 h-4 text-slate-400 absolute left-3.5 top-1/2 -translate-y-1/2" />
          <input
            type="text"
            placeholder="Поиск по названию, штату, городу..."
            value={searchQuery}
            onFocus={stopAutoRotation}
            onChange={(e) => {
              stopAutoRotation();
              setSearchQuery(e.target.value);
            }}
            className="w-full pl-10 pr-4 py-2 text-xs font-mono bg-white dark:bg-white/10 border border-emerald-500/20 dark:border-white/15 rounded-full text-slate-900 dark:text-white placeholder-slate-400 focus:outline-none focus:border-emerald-500 dark:focus:border-white/40 focus:ring-1 focus:ring-emerald-500 dark:focus:ring-white/40 transition-all"
          />
        </div>
      </div>

      {/* ========================================================================= */}
      {/* MASTER MAP & INSPECTOR BENTO GRID                                         */}
      {/* ========================================================================= */}
      <div className="grid grid-cols-1 lg:grid-cols-12 gap-6 items-start">
        
        {/* Left (or Top on mobile): Styled Interactive Vector USA Map (Span 8) */}
        <div className="lg:col-span-8 bento-card-dark p-4 sm:p-7 rounded-2xl sm:rounded-3xl border border-emerald-500/20 dark:border-white/20 relative overflow-hidden flex flex-col justify-between shadow-[0_12px_35px_rgba(16,185,129,0.06)] dark:shadow-none">
          
          {/* Top Map Status Bar */}
          <div className="flex items-center justify-between gap-3 mb-3 sm:mb-4 z-10">
            <div className="flex items-center gap-2.5 text-xs font-mono text-slate-600 dark:text-slate-300">
              <span className="hidden sm:inline-flex items-center gap-2">
                <span className="w-2.5 h-2.5 rounded-full bg-emerald-500 dark:bg-emerald-400" />
                <span>Отображено на карте: <strong>{filteredPlacements.length}</strong> из {placements.length}</span>
              </span>
              <span className="text-slate-400 dark:text-white/20 hidden sm:inline">•</span>
              {isAutoRotating ? (
                <button
                  type="button"
                  onClick={() => setIsAutoRotating(false)}
                  className="inline-flex items-center gap-1.5 text-emerald-700 dark:text-emerald-400 font-medium hover:opacity-80 transition-opacity cursor-pointer bento-pill sm:bg-transparent sm:border-0 px-2.5 py-1 sm:p-0 rounded-full"
                  title="Нажмите, чтобы приостановить автообзор"
                >
                  <span className="w-2 h-2 rounded-full bg-emerald-500 animate-pulse" />
                  <span>Автообзор (5с)</span>
                </button>
              ) : (
                <button
                  type="button"
                  onClick={() => {
                    setActiveRegion("All");
                    setIsAutoRotating(true);
                  }}
                  className="inline-flex items-center gap-1.5 text-slate-500 hover:text-emerald-700 dark:text-slate-400 dark:hover:text-emerald-300 font-medium transition-colors cursor-pointer bento-pill sm:bg-transparent sm:border-0 px-2.5 py-1 sm:p-0 rounded-full"
                  title="Нажмите, чтобы включить автообзор"
                >
                  <span className="w-2 h-2 rounded-full bg-slate-400" />
                  <span>Включить автообзор</span>
                </button>
              )}
            </div>

            {/* Wyoming Legend Pill (Desktop only) */}
            <div className="hidden sm:flex items-center gap-2 text-[11px] font-mono text-slate-700 dark:text-slate-300 bento-pill px-3.5 py-1.5 rounded-full shadow-sm">
              <span className="text-sky-600 dark:text-sky-400 font-bold text-xs">★</span>
              <span className="font-semibold">Кампус автора (Wyoming)</span>
            </div>
          </div>

          {/* SVG Map Canvas with Atmospheric Styling */}
          <div
            onClick={stopAutoRotation}
            className="relative w-full aspect-[975/610] select-none rounded-3xl overflow-hidden bg-gradient-to-b from-[#F0FDF4] via-[#E6F4EA] to-[#DCFCE7] dark:from-[#14171f] dark:via-[#101217] dark:to-[#0c0d11] border border-emerald-500/20 dark:border-white/10 shadow-inner"
          >
            <svg
              viewBox={mapData.viewBox}
              className="w-full h-full filter drop-shadow-[0_20px_40px_rgba(0,0,0,0.15)] dark:drop-shadow-[0_20px_40px_rgba(0,0,0,0.7)]"
            >
              <defs>
                {/* Subtle Dark Landmass Fill Gradient */}
                <radialGradient id="landmassGrad" cx="50%" cy="45%" r="65%">
                  <stop offset="0%" stopColor="#252936" />
                  <stop offset="60%" stopColor="#1a1c24" />
                  <stop offset="100%" stopColor="#12141a" />
                </radialGradient>

                {/* Light Landmass Fill Gradient */}
                <radialGradient id="landmassGradLight" cx="50%" cy="45%" r="65%">
                  <stop offset="0%" stopColor="#FFFFFF" />
                  <stop offset="65%" stopColor="#F7FCF9" />
                  <stop offset="100%" stopColor="#E2F2E8" />
                </radialGradient>

                {/* Wyoming Glowing Ambient Aura */}
                <radialGradient id="amberGlow" cx="50%" cy="50%" r="50%">
                  <stop offset="0%" stopColor="#F59E0B" stopOpacity="0.85" />
                  <stop offset="50%" stopColor="#F59E0B" stopOpacity="0.25" />
                  <stop offset="100%" stopColor="#F59E0B" stopOpacity="0" />
                </radialGradient>

                {/* Selected Pin Halo */}
                <radialGradient id="selectedGlow" cx="50%" cy="50%" r="50%">
                  <stop offset="0%" stopColor="#10B981" stopOpacity="0.8" />
                  <stop offset="60%" stopColor="#10B981" stopOpacity="0.2" />
                  <stop offset="100%" stopColor="#10B981" stopOpacity="0" />
                </radialGradient>

                {/* Ocean Vignette */}
                <radialGradient id="oceanAmbient" cx="40%" cy="40%" r="60%">
                  <stop offset="0%" stopColor="rgba(16, 185, 129, 0.04)" />
                  <stop offset="100%" stopColor="rgba(0, 0, 0, 0.25)" />
                </radialGradient>
              </defs>

              {/* Ocean Ambient Overlay */}
              <rect width="975" height="610" fill="url(#oceanAmbient)" className="dark:block hidden" />

              {/* Coordinate Grid Lines (Subtle Latitude / Longitude) */}
              <g stroke="currentColor" strokeDasharray="3 4" strokeWidth="0.8" className="text-emerald-700/15 dark:text-white/5">
                <line x1="40" y1="120" x2="940" y2="120" />
                <line x1="40" y1="240" x2="940" y2="240" />
                <line x1="40" y1="360" x2="940" y2="360" />
                <line x1="40" y1="480" x2="940" y2="480" />
                <line x1="180" y1="30" x2="180" y2="580" />
                <line x1="360" y1="30" x2="360" y2="580" />
                <line x1="540" y1="30" x2="540" y2="580" />
                <line x1="720" y1="30" x2="720" y2="580" />
              </g>

              {/* Geographic Regional Watermark Typography */}
              <g fill="currentColor" className="text-emerald-900/20 dark:text-white/10" fontFamily="monospace" fontWeight="bold" fontSize="10" letterSpacing="0.2em">
                <text x="70" y="160">PACIFIC NORTHWEST</text>
                <text x="210" y="215">ROCKY MOUNTAINS</text>
                <text x="500" y="225">MIDWEST &amp; GREAT LAKES</text>
                <text x="460" y="470">SOUTH &amp; GULF COAST</text>
                <text x="770" y="320">EAST COAST</text>
              </g>

              {/* US States Boundaries */}
              <g>
                {mapData.statePaths.map((state) => (
                  <path
                    key={state.id}
                    d={state.d}
                    strokeWidth="0.85"
                    className="transition-colors fill-[url(#landmassGradLight)] dark:fill-[url(#landmassGrad)] stroke-emerald-600/30 dark:stroke-white/15 hover:fill-emerald-100 dark:hover:fill-[#252a38]"
                  />
                ))}
              </g>

              {/* US Outer Border Outline with Highlight */}
              <path
                d={mapData.nationPath}
                fill="none"
                strokeWidth="1.8"
                className="stroke-emerald-600/60 dark:stroke-white/35"
              />

              {/* ========================================================================= */}
              {/* INTERACTIVE PLACEMENT PINS                                                */}
              {/* ========================================================================= */}
              {placements.map((p) => {
                const isSelected = selectedPlacement.id === p.id;
                const isHovered = hoveredPlacement?.id === p.id;
                const isMatchingFilter = filteredPlacements.some((fp) => fp.id === p.id);

                return (
                  <g
                    key={p.id}
                    className="cursor-pointer transition-transform duration-200"
                    style={{
                      opacity: isMatchingFilter ? 1 : 0.25,
                      transformOrigin: `${p.x}px ${p.y}px`,
                    }}
                    onClick={() => {
                      stopAutoRotation();
                      setSelectedPlacement(p);
                    }}
                    onMouseEnter={() => setHoveredPlacement(p)}
                    onMouseLeave={() => setHoveredPlacement(null)}
                  >
                    {/* Generous Invisible Touch Target for Mobile Finger Tapping */}
                    <circle cx={p.x} cy={p.y} r={30} fill="transparent" />

                    {/* Outer Tactile Halo Ring */}
                    <circle
                      cx={p.x}
                      cy={p.y}
                      r={isSelected ? 22 : isHovered ? 19 : 15}
                      fill={
                        isSelected
                          ? "rgba(56, 189, 248, 0.35)"
                          : isHovered
                          ? "rgba(56, 189, 248, 0.25)"
                          : "rgba(56, 189, 248, 0.15)"
                      }
                      className="transition-all"
                    />

                    {/* Main High-Contrast Circle */}
                    <circle
                      cx={p.x}
                      cy={p.y}
                      r={isSelected ? 14 : isHovered ? 12 : 9.5}
                      fill={
                        isSelected
                          ? "#FFFFFF"
                          : isHovered
                          ? "#38BDF8"
                          : "#F8FAFC"
                      }
                      stroke="#0F172A"
                      strokeWidth={isSelected ? 2.5 : 1.8}
                      className="transition-all filter drop-shadow-md"
                    />

                    {/* Center Core: Star for Wyoming (Author's Campus), Dot for other campuses */}
                    {p.isAuthorCampus ? (
                      <path
                        d="M 0,-4.8 L 1.4,-1.5 L 4.8,-1.5 L 2.0,0.5 L 3.0,3.8 L 0,1.8 L -3.0,3.8 L -2.0,0.5 L -4.8,-1.5 L -1.4,-1.5 Z"
                        transform={`translate(${p.x}, ${p.y}) scale(${isSelected ? 1.25 : isHovered ? 1.1 : 0.95})`}
                        fill={isSelected ? "#0284C7" : isHovered ? "#FFFFFF" : "#0284C7"}
                        className="transition-all"
                      />
                    ) : (
                      <circle
                        cx={p.x}
                        cy={p.y}
                        r={isSelected ? 6 : isHovered ? 5 : 4}
                        fill={
                          isSelected
                            ? "#0284C7"
                            : isHovered
                            ? "#FFFFFF"
                            : "#0284C7"
                        }
                      />
                    )}
                  </g>
                );
              })}
            </svg>

            {/* Hover Tooltip Overlay with Frosted Glass */}
            {hoveredPlacement && hoveredPlacement.id !== selectedPlacement.id && (
              <div
                className="absolute pointer-events-none z-30 transition-all -translate-x-1/2 -translate-y-full"
                style={{
                  left: `${(hoveredPlacement.x / 975) * 100}%`,
                  top: `${(hoveredPlacement.y / 610) * 100 - 3}%`,
                }}
              >
                <div className="bento-card-glass px-4 py-2 rounded-2xl shadow-2xl text-center border border-white/35 whitespace-nowrap mb-2.5 backdrop-blur-xl">
                  <div className="text-xs font-bold text-white leading-tight">
                    {hoveredPlacement.name}
                  </div>
                  <div className="text-[10px] font-mono text-slate-300 mt-0.5">
                    {hoveredPlacement.city}, {hoveredPlacement.state} • {hoveredPlacement.regionLabel}
                  </div>
                </div>
              </div>
            )}
          </div>

          {/* Quick Clickable Horizontal Strip of Campuses for Mobile/Touch */}
          <div className="mt-4 pt-3 border-t border-emerald-500/15 dark:border-white/10 flex items-center gap-2 overflow-x-auto pb-1 scrollbar-none">
            <span className="text-[11px] font-mono text-slate-500 dark:text-slate-400 shrink-0">Быстрый выбор:</span>
            {filteredPlacements.map((cp) => (
              <button
                key={cp.id}
                onClick={() => {
                  stopAutoRotation();
                  setSelectedPlacement(cp);
                }}
                className={`px-3.5 py-1.5 rounded-full text-[11px] font-mono whitespace-nowrap transition-all ${
                  selectedPlacement.id === cp.id
                    ? "bg-emerald-600 text-white font-bold shadow-sm dark:bg-white dark:text-[#121316]"
                    : "bg-emerald-500/[0.06] text-slate-700 hover:bg-emerald-500/[0.12] dark:bg-white/5 dark:text-slate-300 dark:hover:bg-white/10"
                }`}
              >
                {cp.isAuthorCampus ? `★ ${cp.name}` : cp.name}
              </button>
            ))}
          </div>
        </div>

        {/* Right: Selected Campus Inspector Card (Span 4) */}
        <div className="lg:col-span-4 sticky top-28">
          <AnimatePresence mode="wait">
            <motion.div
              key={selectedPlacement.id}
              onClick={stopAutoRotation}
              initial={{ opacity: 0, y: 10 }}
              animate={{ opacity: 1, y: 0 }}
              exit={{ opacity: 0, y: -10 }}
              transition={{ duration: 0.25 }}
              className="p-7 sm:p-9 rounded-2xl sm:rounded-3xl bento-card-dark border border-emerald-500/20 dark:border-white/20 relative overflow-hidden shadow-[0_12px_35px_rgba(16,185,129,0.06)] dark:shadow-none"
            >
              {/* Status Header */}
              <div className="flex items-center justify-between mb-4">
                <span className="px-3 py-1 rounded-full text-xs font-mono font-medium bento-pill text-emerald-800 dark:text-slate-200">
                  {selectedPlacement.type}
                </span>

                <div className="text-xs font-mono text-slate-500 dark:text-slate-400">
                  {selectedPlacement.lat.toFixed(4)}°, {selectedPlacement.lng.toFixed(4)}°
                </div>
              </div>

              {/* University Name */}
              <h3 className="font-editorial text-2xl sm:text-3xl font-bold text-slate-900 dark:text-white mb-2 leading-tight">
                {selectedPlacement.name}
              </h3>

              {/* Location Badge */}
              <div className="flex items-center gap-2 text-sm font-mono text-slate-600 dark:text-slate-300 mb-6">
                <MapPin className="w-4 h-4 text-emerald-600 dark:text-emerald-400" />
                <span>
                  {selectedPlacement.city}, {selectedPlacement.state}
                </span>
                <span className="text-slate-400 dark:text-white/30">•</span>
                <span className="text-slate-500 dark:text-slate-400">{selectedPlacement.regionLabel}</span>
              </div>

              {/* Campus Highlights */}
              <div className="mb-6 font-doc">
                <div className="text-[11px] font-mono uppercase tracking-wider text-slate-500 dark:text-slate-400 font-bold mb-2">
                  Особенности кампусной среды:
                </div>
                <p className="text-xs sm:text-sm text-slate-800 dark:text-slate-200 leading-relaxed">
                  {selectedPlacement.highlight}
                </p>
              </div>

              {/* Author Campus Special Section */}
              {selectedPlacement.isAuthorCampus ? (
                <div className="p-4 rounded-xl bg-emerald-500/[0.06] border border-emerald-500/20 text-xs text-slate-800 dark:text-slate-200 mb-6 leading-relaxed font-doc">
                  <div className="font-bold font-mono mb-1 flex items-center gap-1.5 text-emerald-700 dark:text-emerald-400">
                    <span className="text-sky-600 dark:text-sky-400">★</span>
                    Личный опыт так называемого Иля (SPARK &apos;26)
                  </div>
                  «Вайоминг дал мне доступ к передовым исследовательским лабораториям, окружение из профессоров мирового уровня и потрясающие походы в Скалистые горы. В программе делюсь всеми деталями адаптации к жизни в американском колледже».
                </div>
              ) : (
                <div className="p-4 rounded-xl bg-emerald-500/[0.04] border border-emerald-500/15 dark:bg-white/5 dark:border-white/10 text-xs text-slate-700 dark:text-slate-300 mb-6 leading-relaxed font-doc">
                  <span className="text-slate-900 dark:text-white font-semibold font-mono">Статус распределения:</span> Кампус фигурировал в официальных пулах участников программы прошлых лет.
                </div>
              )}

              {/* Geo Specs Mini Bar */}
              <div className="pt-4 border-t border-emerald-500/15 dark:border-white/10 grid grid-cols-2 gap-3 text-xs font-mono">
                <div>
                  <span className="text-slate-500 dark:text-slate-400 block text-[10px] uppercase">Регион США</span>
                  <span className="text-slate-900 dark:text-white font-semibold">{selectedPlacement.regionLabel}</span>
                </div>
                <div>
                  <span className="text-slate-500 dark:text-slate-400 block text-[10px] uppercase">Тип финансирования</span>
                  <span className="text-emerald-700 dark:text-emerald-400 font-semibold">Full $20k Grant</span>
                </div>
              </div>
            </motion.div>
          </AnimatePresence>
        </div>

      </div>
    </section>
  );
}
