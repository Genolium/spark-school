"use client";

import React, { useEffect, useState } from "react";
import { BOT_URL, withBotAttribution } from "@/lib/analytics";

interface BotLinkProps extends React.AnchorHTMLAttributes<HTMLAnchorElement> {
  /** Full bot URL (for example a diagnostic plan link). */
  href?: string;
  /** Telegram start payload without the attribution token. */
  start?: string;
}

/**
 * Telegram bot link that carries the anonymous attribution token.
 *
 * The token lives in localStorage, so the href is upgraded after mount; the
 * first render matches the server output and does not change the layout.
 */
export function BotLink({ href, start = "", children, ...props }: BotLinkProps) {
  const base = href || (start ? `${BOT_URL}?start=${start}` : BOT_URL);
  const [resolved, setResolved] = useState(base);

  useEffect(() => {
    setResolved(withBotAttribution(base));
  }, [base]);

  return (
    <a {...props} href={resolved}>
      {children}
    </a>
  );
}
