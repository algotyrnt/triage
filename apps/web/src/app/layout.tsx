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
