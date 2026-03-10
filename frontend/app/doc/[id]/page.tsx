"use client";

import { useEffect, useMemo, useState } from "react";
import { useParams } from "next/navigation";
import Navbar from "../../components/Navbar";
import BackButton from "../../components/BackButton";

const API = process.env.NEXT_PUBLIC_API_BASE || "";

type Doc = {
  id: string;
  title: string;
  meta?: any;
};

function esc(s: any) {
  return (s ?? "").toString();
}

export default function DocDetailPage() {
  const params = useParams();
  const id = (params?.id as string) || "";

  const [doc, setDoc] = useState<Doc | null>(null);
  const [loading, setLoading] = useState(true);

  // ---- group expand ----
  const [groupOpen, setGroupOpen] = useState(false);
  const [groupLoading, setGroupLoading] = useState(false);
  const [groupItems, setGroupItems] = useState<any[]>([]);

  useEffect(() => {
    if (!id) return;
    setLoading(true);

    fetch(`${API}/api/doc/${encodeURIComponent(id)}`, { cache: "no-store" })
      .then((r) => (r.ok ? r.json() : null))
      .then((d) => {
        setDoc(d);
        // reset group state when doc changes
        setGroupOpen(false);
        setGroupItems([]);
        setGroupLoading(false);
      })
      .finally(() => setLoading(false));
  }, [id]);

  const m = useMemo(() => doc?.meta || {}, [doc]);

  const catMain = esc(m.categoryMain || "-");
  const catSub = esc(m.categorySub || "");
  const groupName = esc(m.group || "");

  async function toggleGroup(main: string, sub: string, group: string) {
    if (!group) return;

    // collapse
    if (groupOpen) {
      setGroupOpen(false);
      return;
    }

    // expand: load if not loaded
    setGroupOpen(true);

    // ถ้าเคยโหลดแล้ว ไม่ต้องยิงซ้ำ
    if (groupItems.length > 0) return;

    setGroupLoading(true);
    setGroupItems([]);

    try {
      const res = await fetch(
        `${API}/api/group?main=${encodeURIComponent(main)}&sub=${encodeURIComponent(sub)}&group=${encodeURIComponent(group)}`,
        { cache: "no-store" }
      );
      const data = await res.json();
      setGroupItems(data.items || []);
    } finally {
      setGroupLoading(false);
    }
  }

  return (
    <main className="wrap">
      <Navbar />
      <BackButton />
      <div className="card">

        {loading && (
          <div className="small" style={{ marginTop: 10 }}>
            กำลังโหลด…
          </div>
        )}

        {!loading && !doc && (
          <div className="small" style={{ marginTop: 10 }}>
            ไม่พบข้อมูล
          </div>
        )}

        {doc && (
          <>
            <div className="title" style={{ marginTop: 10 }}>
              {esc(doc.title)}
            </div>

            {/* หมวด/หมวดย่อย (ไม่คลิก) */}
            <div className="small" style={{ marginTop: 8 }}>
              <div>
                <b>หมวด:</b> {catMain}
              </div>

              {!!catSub.trim() && (
                <div style={{ marginTop: 4 }}>
                  <b>หมวดย่อย:</b> {catSub}
                </div>
              )}

              {/* กลุ่มรายการ (กดดูรายการ) */}
              {!!groupName.trim() && (
                <div style={{ marginTop: 6 }}>
                  <b>กลุ่มรายการ:</b> {groupName}{" "}
                  <button
                    type="button"
                    onClick={() => toggleGroup(catMain, catSub, groupName)}
                    aria-expanded={groupOpen}
                    style={{
                      background: "none",
                      border: "none",
                      padding: 0,
                      marginLeft: 8,
                      cursor: "pointer",
                      font: "inherit",
                      fontSize: 13,
                      opacity: 0.7,
                      color: "var(--blue)",
                    }}
                  >
                    {groupOpen ? "ซ่อนรายการ ▲" : "ดูรายการ ▼"}
                  </button>

                  {groupOpen && (
                    <div style={{ marginTop: 8, paddingLeft: 22 }}>
                      {groupLoading && <div className="small">กำลังโหลด…</div>}

                      {!groupLoading && groupItems.length === 0 && (
                        <div className="small">ไม่พบรายการในกลุ่มนี้</div>
                      )}

                      {!groupLoading && groupItems.length > 0 && (
                        <ul style={{ margin: 0, paddingLeft: 18 }}>
                          {groupItems.map((it, idx) => (
                            <li key={it.id ?? idx} style={{ marginTop: 6 }}>
                              <a
                                href={`/doc/${encodeURIComponent(it.id)}`}
                                style={{ textDecoration: "none" }}
                              >
                                {esc(it.title)}
                              </a>
                            </li>
                          ))}
                        </ul>
                      )}
                    </div>
                  )}
                </div>
              )}
            </div>

            <div className="small" style={{ marginTop: 6 }}>
              <b>อ้างอิง:</b> หน้า {esc(m.page || "-")} ลำดับ {esc(m.row || "-")}
            </div>

            {!!esc(m.budgetUse).trim() && (
              <div className="small" style={{ marginTop: 6 }}>
                <b>การใช้งบ:</b> {esc(m.budgetUse)}
              </div>
            )}

            {!!esc(m.emergency).trim() && (
              <div className="small" style={{ marginTop: 6, whiteSpace: "pre-wrap" }}>
                <b>ครุภัณฑ์ที่ให้ส่วนภูมิภาคเป็นผู้อนุมัติหลักการฯ:</b> {esc(m.emergency)}
              </div>
            )}

            {!!esc(m.special).trim() && (
              <div className="card" style={{ marginTop: 12 }}>
                <div className="small">
                  <b>หมายเหตุ</b>
                </div>
                <div className="small" style={{ marginTop: 8, whiteSpace: "pre-wrap" }}>
                  {esc(m.special)}
                </div>
              </div>
            )}
          </>
        )}
      </div>
    </main>
  );
}