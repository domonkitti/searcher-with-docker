"use client";

import Link from "next/link";

export default function Navbar() {
  return (
    <div className="topbar">
      <div className="nav">
        <Link href="/">Search</Link>
        <Link href="/kits">เครื่องมือ</Link>
        <Link href="/admin/import-items">Import Excel</Link>
        <Link href="https://forms.gle/fiiwtQbShh7n2U6B8" target="_blank" rel="noopener noreferrer">
          ให้ข้อเสนอเนะ
        </Link>
      </div>
    </div>
  );
}
