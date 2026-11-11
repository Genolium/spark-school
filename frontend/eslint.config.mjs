import { defineConfig, globalIgnores } from "eslint/config";
import nextVitals from "eslint-config-next/core-web-vitals";
import nextTs from "eslint-config-next/typescript";

export default defineConfig([
  ...nextVitals,
  ...nextTs,
  {
    rules: {
      // Stylistic: the API layer deliberately uses loose JSON typing.
      "@typescript-eslint/no-explicit-any": "off",
      // React Compiler advisory rules: the codebase uses the standard fetch-in-effect / hydrate-from-localStorage patterns.
      "react-hooks/set-state-in-effect": "off",
      "react-hooks/immutability": "off",
    },
  },
  {
    // Tailwind config is loaded by Node as a config file; require() for plugins is intentional.
    files: ["tailwind.config.ts"],
    rules: { "@typescript-eslint/no-require-imports": "off" },
  },
  globalIgnores([".next/**", "node_modules/**", "out/**", "build/**", "next-env.d.ts"]),
]);
