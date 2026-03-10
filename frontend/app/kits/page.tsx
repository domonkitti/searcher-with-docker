"use client";

import { useEffect, useMemo, useState } from "react";
import Link from "next/link";
import Navbar from "../components/Navbar";

const API = process.env.NEXT_PUBLIC_API_BASE || "";
type KitDetail = { kitId: string; kitName: string; category?: string; page?: string };

function esc(s: any) {
  return (s ?? "").toString();
}

export default function KitsPage() {
  const [kits, setKits] = useState<KitDetail[]>([]);

  useEffect(() => {
    fetch(`${API}/api/kits`)
      .then((r) => r.json())
      .then((d) => setKits(d.kits || []));
  }, []);

  const kitNames = useMemo(() => {
    return [...kits].sort((a, b) => (a.kitName || "").localeCompare(b.kitName || ""));
  }, [kits]);

  return (
    <main className="wrap">
      <Navbar />

      <div className="card">
        <div className="title">ชุดเครื่องมือ</div>
        <div className="small" style={{ marginTop: 6 }}>
          - กรณีจัดซื้อชุดเครื่องมือแบบครบชุด ให้เบิกจ่ายจากงบลงทุน
          <br />- กรณีจัดซื้อรายชิ้น พิจารณา ดังนี้
          <br />- ถ้าเป็นสิ่งของที่ระบุไว้ในรายการครุภัณฑ์ ให้ถือว่าสิ่งของนั้นเป็น "ครุภัณฑ์" โดยไม่ต้อง
            คํานึงถึงราคาแต่อย่างใด ค่าใช้จ่ายให้เบิกจ่ายจากงบลงทุน
          <br />- ถ้าเป็นสิ่งของที่มิได้ระบุไว้ในรายการครุภัณฑ์ จะต้องพิจารณาถึงราคาของสิ่งของนั้น ดังนี้
          <br />    1. ถ้ามีราคาต่อหน่วยหรือชุดหนึ่งเกินกว่า 10,000.- บาท และอายุการใช้งานเกิน
            กว่า 1 ปี ให้ถือว่าสิ่งของนั้นเป็น "ครุภัณฑ์" ค่าใช้จ่ายให้เบิกจ่ายจากงบลงทุน
          <br />    2. ถ้ามีราคาต่อหน่วยหรือชุดหนึ่งไม่เกิน 10,000.- บาท หรือเกินกว่า 10,000.- บาท
            แต่อายุการใช้งานไม่เกิน 1 ปี ให้ถือว่าสิ่งของนั้นเป็น "วัสดุ" ค่าใช้จ่ายให้เบิกจ่ายจากงบทําการ
        </div>

        <div className="card" style={{ marginTop: 12 }}>
          <div className="small">
            <b>รายการชุดทั้งหมด:</b> {kitNames.length}
          </div>

          <div style={{ display: "flex", flexDirection: "column", gap: 8, marginTop: 10 }}>
            {kitNames.map((k) => (
              <Link
                key={k.kitId}
                className="pill"
                href={`/kits/${encodeURIComponent(k.kitId)}`}
                title={esc(k.category || "")}
                style={{ whiteSpace: "pre-wrap" }}
              >
                {esc(k.kitName)}
              </Link>
            ))}
          </div>
        </div>
      </div>
    </main>
  );
}