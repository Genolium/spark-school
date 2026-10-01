import type { Config } from "tailwindcss";

const config: Config = {
  darkMode: ["class"],
  content: [
    "./src/pages/**/*.{js,ts,jsx,tsx,mdx}",
    "./src/components/**/*.{js,ts,jsx,tsx,mdx}",
    "./src/app/**/*.{js,ts,jsx,tsx,mdx}",
  ],
  theme: {
    container: {
      center: true,
      padding: "2rem",
      screens: {
        "2xl": "1520px",
      },
    },
    extend: {
      fontFamily: {
        sans: ["var(--font-sans)", "PT Root UI", "-apple-system", "BlinkMacSystemFont", "Segoe UI", "sans-serif"],
        display: ["var(--font-heading)", "var(--font-sans)", "Golos Text", "PT Root UI", "sans-serif"],
        heading: ["var(--font-heading)", "var(--font-sans)", "Golos Text", "PT Root UI", "sans-serif"],
        editorial: ["var(--font-heading)", "var(--font-sans)", "Golos Text", "PT Root UI", "sans-serif"],
        mono: ["var(--font-mono)", "IBM Plex Mono", "ui-monospace", "monospace"],
        serif: ["var(--font-serif)", "Spectral", "Georgia", "serif"],
        doc: ["var(--font-doc)", "PT Astra Sans", "sans-serif"],
      },
      colors: {
        charcoal: {
          DEFAULT: "#121316",
          dark: "#0E1015",
          surface: "#181A20",
        },
        cloud: "#F8F9FA",
        sand: "#EFECE6",
        tint: {
          lavender: "#D8D2FB",
          olive: "#D5E8D2",
          amber: "#FCE1AD",
          sky: "#CDE6FF",
        },
      },
      borderRadius: {
        bento: "40px",
        card: "32px",
        pill: "9999px",
      },
      boxShadow: {
        "spatial-glass": "0 25px 60px -15px rgba(0, 0, 0, 0.4), inset 0 1px 1px rgba(255, 255, 255, 0.2)",
        "spatial-light": "0 20px 45px -10px rgba(0, 0, 0, 0.08), inset 0 1px 1px rgba(255, 255, 255, 0.8)",
      },
    },
  },
  plugins: [require("tailwindcss-animate")],
};

export default config;
