"use client";

import { useEffect, useMemo, useState } from "react";
import { useParams } from "next/navigation";
import Navbar from "../../components/Navbar";
import BackButton from "../../components/BackButton";

const API = process.env.NEXT_PUBLIC_API_BASE || "";
type KitLine = { item: string; subItem?: string; unit?: string };
type KitDetail = {
  category?: string;
  kitName: string;
  page?: string;
  order?: string;     // เก็บไว้ก่อน
  special?: string;   // เก็บไว้ก่อน
  lines: KitLine[];
};

function esc(s: any) {
  return (s ?? "").toString();
}

export default function KitDetailPage() {
  const params = useParams();
  const kitId = (params?.kitId as string) || "";

  const [kit, setKit] = useState<KitDetail | null>(null);

  useEffect(() => {
  if (!kitId) return;
  fetch(`${API}/api/kits/${encodeURIComponent(kitId)}`)
    .then(r => r.json())
    .then(d => setKit(d.kit || null));
  }, [kitId]);

  const grouped = useMemo(() => {
    if (!kit) return [];
    const map: Record<string, KitLine[]> = {};
    for (const ln of kit.lines || []) {
      const key = ln.item || "-";
      map[key] ??= [];
      map[key].push(ln);
    }
    const keys = Object.keys(map).sort((a, b) => a.localeCompare(b));
    return keys.map((k) => ({ item: k, lines: map[k] }));
  }, [kit]);

  return (
    <main className="wrap">
      <Navbar />
      <BackButton />
      <div className="card">

        {!kit && (
          <div className="small" style={{ marginTop: 10 }}>
            กำลังโหลด…
          </div>
        )}

        {kit && (
          <>
            <div className="title no-hover" style={{ marginTop: 10, whiteSpace: "pre-wrap" }}>
              {esc(kit.kitName)}
              </div>

            <div className="small" style={{ marginTop: 6 }}>
              <b>อ้างอิง:</b> หน้า {esc(kit.page || "-")}
            </div>

            {/* NOTE: order/special เก็บไว้ก่อน ยังไม่โชว์ตามที่ขอ */}
            {/* <div className="small">ลำดับ: {esc(kit.order)} | เงื่อนไขพิเศษ: {esc(kit.special)}</div> */}

            <div className="kitTable" style={{ marginTop: 12 }}>
              <div className="kitHead">
                <div>รายการ</div>
                <div style={{ textAlign: "right" }}>หน่วย</div>
              </div>

              {grouped.map((g, idx) => {
                const subLines = (g.lines || []).filter((x) => (x.subItem || "").trim() !== "");
                const mainLine = (g.lines || []).find((x) => (x.subItem || "").trim() === "");
                const mainUnit = mainLine?.unit;

                // ✅ ลำดับเริ่มที่ 1 เฉพาะ "รายการหลัก"
                const no = idx + 1;

                return (
                  <div className="kitRow" key={idx} style={{ display: "block" }}>
                    {/* รายการหลัก (มีเลขลำดับ) */}
                    <div style={{ display: "flex", justifyContent: "space-between", gap: 12 }}>
                      <div>
                        <span style={{ opacity: 0.7, marginRight: 8 }}>{no}.</span>
                        <span>{esc(g.item)}</span>
                      </div>

                      {!!(mainUnit || "").trim() && (
                        <div style={{ textAlign: "right", whiteSpace: "nowrap" }}>
                          {esc(mainUnit)}
                        </div>
                      )}
                    </div>

                    {/* รายการย่อย (ไม่ใส่เลข, เยื้องขวาให้รู้ว่าเป็น sub) */}
                    {subLines.map((sl, sidx) => (
                      <div
                        key={sidx}
                        style={{
                          display: "flex",
                          justifyContent: "space-between",
                          gap: 12,
                          marginTop: 6,
                          paddingLeft: 28, // ✅ เยื้องขวาเพิ่มจากเดิมให้ดูเป็น sub ชัดขึ้น
                        }}
                      >
                        <div style={{ opacity: 0.92 }}>• {esc(sl.subItem)}</div>

                        {!!(sl.unit || "").trim() && (
                          <div style={{ textAlign: "right", whiteSpace: "nowrap", opacity: 0.92 }}>
                            {esc(sl.unit)}
                          </div>
                        )}
                      </div>
                    ))}
                  </div>
                );
              })}
            </div>

            <div className="small" style={{ marginTop: 10, opacity: 0.7 }}>
              
            </div>
          </>
        )}
      </div>
    </main>
  );
}