/**
 * Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
 * SPDX-License-Identifier: Apache-2.0
 */

import type { ReactNode } from "react";
import "./globals.css";

export const metadata = {
  title: "Triage - Go Crash Isolation",
  description: "Go crash isolation and AI diagnostic platform",
};

export default function RootLayout({ children }: { children: ReactNode }) {
  return (
    <html lang="en">
      <body>{children}</body>
    </html>
  );
}
