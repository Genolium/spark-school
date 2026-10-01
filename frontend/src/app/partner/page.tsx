import { redirect } from "next/navigation";

// Партнёрская программа временно отключена по запросу пользователя.
// Исходный компонент сохранён в page.original.tsx.bak
export default function PartnerPage() {
  redirect("/");
}
