import "./globals.css";
import type { ReactNode } from "react";

export const metadata = { title: "ค้นหา หนังสือจำแนก" };

export default function RootLayout({ children }: { children: ReactNode }) {
  return (
    <html lang="th">
      <body>{children}</body>
    </html>
  );
}
