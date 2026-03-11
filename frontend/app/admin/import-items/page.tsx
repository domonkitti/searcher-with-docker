"use client";

import { useState } from "react";
import Navbar from "../../components/Navbar";

const API = process.env.NEXT_PUBLIC_API_BASE || "";

type ImportCardProps = {
  title: string;
  note: string;
  endpoint: string;
};

function ImportCard({ title, note, endpoint }: ImportCardProps) {
  const [file, setFile] = useState<File | null>(null);
  const [message, setMessage] = useState("");
  const [loading, setLoading] = useState(false);

  async function upload() {
    if (!file) {
      setMessage("กรุณาเลือกไฟล์ Excel ก่อน");
      return;
    }

    const form = new FormData();
    form.append("file", file);

    setLoading(true);
    setMessage("");

    try {
      const res = await fetch(`${API}${endpoint}`, {
        method: "POST",
        body: form,
      });
      const data = await res.json();
      if (!res.ok) throw new Error(data?.error || data?.detail || "นำเข้าไม่สำเร็จ");
      setMessage(`นำเข้าสำเร็จ ${data.inserted ?? 0} รายการ`);
    } catch (err: any) {
      setMessage(err?.message || "นำเข้าไม่สำเร็จ");
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="result">
      <div className="title" style={{ fontSize: 20 }}>{title}</div>
      <div className="meta" style={{ marginTop: 8, marginBottom: 12 }}>{note}</div>
      <input
        type="file"
        accept=".xlsx,.xls"
        onChange={(e) => setFile(e.target.files?.[0] || null)}
      />
      <div style={{ marginTop: 16 }}>
        <button id="btn" onClick={upload} disabled={loading}>
          {loading ? "กำลังนำเข้า..." : "อัปโหลดและนำเข้า"}
        </button>
      </div>
      {!!message && <div className="meta" style={{ marginTop: 16 }}>{message}</div>}
    </div>
  );
}

export default function ImportItemsPage() {
  return (
    <main className="wrap">
      <Navbar />
      <div className="logo">
        นำเข้า<span>Excel เข้าระบบค้นหา</span>
      </div>

      <section
        className="results"
        style={{ maxWidth: 820, margin: "24px auto", gap: 16 }}
      >
        <ImportCard
          title="นำเข้ารายการหลัก"
          note="ไฟล์รายการหลักจะลบข้อมูลรายการเดิมทั้งหมดในฐานข้อมูล แล้วแทนที่ด้วยข้อมูลใหม่จาก Excel โดยไฟล์นี้ต้องมีคอลัมน์ ID เป็นคอลัมน์แรก"
          endpoint="/api/admin/import/items"
        />

        <ImportCard
          title="นำเข้าลิงก์ประกอบรายการ"
          note="ไฟล์นี้ใช้จับคู่กับ ID ของไฟล์รายการหลัก โดยต้องมี 3 คอลัมน์คือ ID, realLink และ displayLine และควรนำเข้าหลังจากนำเข้ารายการหลักแล้ว"
          endpoint="/api/admin/import/item-links"
        />

        <ImportCard
          title="นำเข้าชุดพัสดุ"
          note="ไฟล์ที่อัปโหลดจะลบข้อมูลชุดพัสดุเดิมทั้งหมดในฐานข้อมูล แล้วแทนที่ด้วยข้อมูลใหม่จาก Excel"
          endpoint="/api/admin/import/kits"
        />
      </section>
    </main>
  );
}