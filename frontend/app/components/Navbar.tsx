"use client";

import Link from "next/link";

export default function Navbar() {
  return (
    <div className="topbar">
      <div className="nav">
        <Link href="/">Search</Link>
        <Link href="/kits">ชุดพัสดุ</Link>
        <Link href="/admin/import-items">Import Excel</Link>
      </div>
    </div>
  );
}
