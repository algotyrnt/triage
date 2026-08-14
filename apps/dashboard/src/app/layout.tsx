import type { Metadata } from "next";
import "@/app/globals.css";

export const metadata: Metadata = {
  title: "triage Studio Dashboard",
  description: "Enterprise Go Crash Isolation & AI Diagnostic Dashboard",
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en">
      <body>{children}</body>
    </html>
  );
}
