import "./globals.css";
import type { ReactNode } from "react";
import ContactBar from "./components/ContactBar";
import ChecklistAssistant from "./components/ChecklistAssistant";

export const metadata = { title: "ค้นหา หนังสือจำแนก" };

export default function RootLayout({ children }: { children: ReactNode }) {
  return (
    <html lang="th">
      <body style={{ paddingBottom: "64px" }}>
        {children}
        <ChecklistAssistant />
        <ContactBar />
      </body>
    </html>
  );
}