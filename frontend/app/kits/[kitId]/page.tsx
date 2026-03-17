"use client";

import { useEffect, useMemo, useState } from "react";
import { useParams } from "next/navigation";
import Navbar from "../../components/Navbar";
import BackButton from "../../components/BackButton";

const API = process.env.NEXT_PUBLIC_API_BASE || "";

type KitLine = {
  item: string;
  subItem?: string;
  unit?: string;
  linkedItemSourceId?: string;
  linkedItemTitle?: string;
};

type KitDetail = {
  sourceId?: string;
  category?: string;
  kitName: string;
  page?: string;
  order?: string;
  special?: string;
  lines: KitLine[];
};

function esc(s: any) {
  return (s ?? "").toString();
}

export default function KitDetailPage() {
  const params = useParams();
  const kitRef = (params?.kitId as string) || "";
  const [kit, setKit] = useState<KitDetail | null>(null);

  useEffect(() => {
    if (!kitRef) return;
    fetch(`${API}/api/kits/${encodeURIComponent(kitRef)}`).then((r) => r.json()).then((d) => setKit(d.kit || null));
  }, [kitRef]);

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
        {!kit && <div className="small" style={{ marginTop: 10 }}>กำลังโหลด…</div>}

        {kit && (
          <>
            <div className="title no-hover" style={{ marginTop: 10, whiteSpace: "pre-wrap" }}>{esc(kit.kitName)}</div>
            <div className="small" style={{ marginTop: 6 }}><b>อ้างอิง:</b> หน้า {esc(kit.page || "-")}</div>

            <div className="kitTable" style={{ marginTop: 12 }}>
              <div className="kitHead">
                <div>รายการ</div>
                <div style={{ textAlign: "right" }}>หน่วย</div>
              </div>

              {grouped.map((g, idx) => {
                const subLines = (g.lines || []).filter((x) => (x.subItem || "").trim() !== "");
                const mainLine = (g.lines || []).find((x) => (x.subItem || "").trim() === "");
                const mainUnit = mainLine?.unit;
                const mainLinkedSourceId = (mainLine?.linkedItemSourceId || "").trim().toLowerCase();
                const mainLinkedTitle = (mainLine?.linkedItemTitle || "").trim();
                const no = idx + 1;

                return (
                  <div className="kitRow" key={idx} style={{ display: "block" }}>
                    <div style={{ display: "flex", justifyContent: "space-between", gap: 12 }}>
                      <div>
                        <span style={{ opacity: 0.7, marginRight: 8 }}>{no}.</span>
                        <span>{esc(g.item)}</span>
                        {!!mainLinkedSourceId && (
                          <div style={{ marginTop: 6, paddingLeft: 20 }}>
                            <a
                              href={`/doc/${encodeURIComponent(mainLinkedSourceId)}`}
                              style={{
                                display: "inline-block",
                                padding: "8px 14px",
                                backgroundColor: "#DBEAFE",
                                color: "#1E3A8A",
                                textDecoration: "none",
                                borderRadius: 8,
                                fontSize: 14,
                                fontWeight: 500,
                                cursor: "pointer",
                              }}
                            >
                              อยู่ในรายการครุภัณฑ์
                            </a>
                          </div>
                        )}
                      </div>

                      {!!(mainUnit || "").trim() && <div style={{ textAlign: "right", whiteSpace: "nowrap" }}>{esc(mainUnit)}</div>}
                    </div>

                    {subLines.map((sl, sidx) => {
                      const subLinkedSourceId = (sl.linkedItemSourceId || "").trim().toLowerCase();
                      const subLinkedTitle = (sl.linkedItemTitle || "").trim();
                      return (
                        <div key={sidx} style={{ marginTop: 6, paddingLeft: 28 }}>
                          <div style={{ display: "flex", justifyContent: "space-between", gap: 12 }}>
                            <div style={{ opacity: 0.92 }}>• {esc(sl.subItem)}</div>
                            {!!(sl.unit || "").trim() && <div style={{ textAlign: "right", whiteSpace: "nowrap", opacity: 0.92 }}>{esc(sl.unit)}</div>}
                          </div>

                          {!!subLinkedSourceId && (
                            <div style={{ marginTop: 4, paddingLeft: 14 }}>
                              <a href={`/doc/${encodeURIComponent(subLinkedSourceId)}`} style={{ display: "inline-block", textDecoration: "none", color: "#2563eb", fontSize: 14 }}>
                                มี item นี้ กดเพื่อไปดู{subLinkedTitle ? `: ${subLinkedTitle}` : ""} ({subLinkedSourceId})
                              </a>
                            </div>
                          )}
                        </div>
                      );
                    })}
                  </div>
                );
              })}
            </div>
          </>
        )}
      </div>
    </main>
  );
}
