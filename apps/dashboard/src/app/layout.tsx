/**
 * Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
 * SPDX-License-Identifier: Apache-2.0
 */

import type { Metadata } from 'next';
import '@/app/globals.css';

export const metadata: Metadata = {
  title: 'triage Studio Dashboard',
  description: 'Enterprise Go Crash Isolation & AI Diagnostic Dashboard',
  icons: {
    icon: '/favicon.svg',
  },
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en">
      <body>{children}</body>
    </html>
  );
}
