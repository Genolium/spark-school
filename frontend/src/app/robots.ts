import type { MetadataRoute } from "next";

export default function robots(): MetadataRoute.Robots {
  return {
    rules: [
      {
        userAgent: "*",
        allow: ["/", "/partner", "/offer", "/privacy", "/llms.txt", "/llms-full.txt"],
        disallow: ["/admin", "/admin/*", "/api/*", "/dashboard", "/dashboard/*"],
      },
      {
        userAgent: [
          "GPTBot",
          "ChatGPT-User",
          "ClaudeBot",
          "PerplexityBot",
          "Google-Extended",
          "Applebot-Extended",
          "cohere-ai",
        ],
        allow: ["/", "/partner", "/offer", "/privacy", "/llms.txt", "/llms-full.txt"],
        disallow: ["/admin", "/admin/*", "/api/*", "/dashboard", "/dashboard/*"],
      },
    ],
    sitemap: "https://so-called-spark.ru/sitemap.xml",
  };
}
