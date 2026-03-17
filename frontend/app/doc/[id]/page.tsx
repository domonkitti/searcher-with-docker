"use client";

import { useEffect, useMemo, useState } from "react";
import { useParams } from "next/navigation";
import Navbar from "../../components/Navbar";
import BackButton from "../../components/BackButton";

const API = process.env.NEXT_PUBLIC_API_BASE || "";

type LinkItem = {
  realLink?: string;
  displayLine?: string;
  lineNo?: number;
};

type Doc = {
  id: string;
  title: string;
  meta?: any;
};

function esc(s: any) {
  return (s ?? "").toString();
}

function hasMeaningfulValue(v: any) {
  const s = esc(v).trim();
  return s !== "" && s !== "-" && s.toLowerCase() !== "null" && s.toLowerCase() !== "undefined";
}

export default function DocDetailPage() {
  const params = useParams();
  const sourceId = ((params?.id as string) || "").trim().toLowerCase();

  const [doc, setDoc] = useState<Doc | null>(null);
  const [loading, setLoading] = useState(true);

  const [groupOpen, setGroupOpen] = useState(false);
  const [groupLoading, setGroupLoading] = useState(false);
  const [groupItems, setGroupItems] = useState<any[]>([]);

  useEffect(() => {
    if (!sourceId) return;
    setLoading(true);

    fetch(`${API}/api/doc/${encodeURIComponent(sourceId)}`, { cache: "no-store" })
      .then((r) => (r.ok ? r.json() : null))
      .then((d) => {
        setDoc(d);
        setGroupOpen(false);
        setGroupItems([]);
        setGroupLoading(false);
      })
      .finally(() => setLoading(false));
  }, [sourceId]);

  const m = useMemo(() => doc?.meta || {}, [doc]);

  const catMain = esc(m.categoryMain || "-");
  const catSub = esc(m.categorySub || "");
  const groupName = esc(m.group || "");
  const docSourceId = esc(m.sourceId || "");
  const description = esc(m.description || "");
  const page = esc(m.page || "");
  const row = esc(m.row || "");
  const budgetUse = esc(m.budgetUse || "");
  const emergency = esc(m.emergency || "");
  const special = esc(m.special || "");
  const approvalCondition = esc(m.approvalCondition || "");
  const links: LinkItem[] = Array.isArray(m.links) ? m.links : [];

  const showPage = hasMeaningfulValue(page);
  const showRow = hasMeaningfulValue(row);
  const showReference = showPage || showRow;

  async function toggleGroup(main: string, sub: string, group: string) {
    if (!group) return;
    if (groupOpen) {
      setGroupOpen(false);
      return;
    }
    setGroupOpen(true);
    if (groupItems.length > 0) return;

    setGroupLoading(true);
    setGroupItems([]);

    try {
      const res = await fetch(`${API}/api/group?main=${encodeURIComponent(main)}&sub=${encodeURIComponent(sub)}&group=${encodeURIComponent(group)}`, { cache: "no-store" });
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
        {loading && <div className="small" style={{ marginTop: 10 }}>กำลังโหลด…</div>}
        {!loading && !doc && <div className="small" style={{ marginTop: 10 }}>ไม่พบข้อมูล</div>}

        {doc && (
          <>
            <div className="title" style={{ marginTop: 10 }}>{esc(doc.title)}</div>

            {!!description.trim() && (
              <div style={{ marginTop: 10, color: "#4b5563", lineHeight: 1.7, whiteSpace: "pre-line" }}>
                {description}
              </div>
            )}

            <div className="small" style={{ marginTop: 12 }}>
              <div><b>หมวด:</b> {catMain}</div>
              {!!catSub.trim() && <div style={{ marginTop: 4 }}><b>หมวดย่อย:</b> {catSub}</div>}

              {!!groupName.trim() && (
                <div style={{ marginTop: 6 }}>
                  <b>กลุ่มรายการ:</b> {groupName}{" "}
                  <button
                    type="button"
                    onClick={() => toggleGroup(catMain, catSub, groupName)}
                    aria-expanded={groupOpen}
                    style={{ background: "none", border: "none", padding: 0, marginLeft: 8, cursor: "pointer", font: "inherit", fontSize: 13, opacity: 0.7, color: "var(--blue)" }}
                  >
                    {groupOpen ? "ซ่อนรายการ ▲" : "ดูรายการ ▼"}
                  </button>

                  {groupOpen && (
                    <div style={{ marginTop: 8, paddingLeft: 22 }}>
                      {groupLoading && <div className="small">กำลังโหลด…</div>}
                      {!groupLoading && groupItems.length === 0 && <div className="small">ไม่พบรายการในกลุ่มนี้</div>}
                      {!groupLoading && groupItems.length > 0 && (
                        <ul style={{ margin: 0, paddingLeft: 18 }}>
                          {groupItems.map((it, idx) => {
                            const targetSourceId = esc(it.sourceId || "").trim().toLowerCase();
                            return (
                              <li key={targetSourceId || idx} style={{ marginTop: 6 }}>
                                {targetSourceId ? (
                                  <a href={`/doc/${encodeURIComponent(targetSourceId)}`} style={{ textDecoration: "none" }}>
                                    {esc(it.title)}
                                  </a>
                                ) : (
                                  <span>{esc(it.title)}</span>
                                )}
                              </li>
                            );
                          })}
                        </ul>
                      )}
                    </div>
                  )}
                </div>
              )}
            </div>

            {showReference && (
              <div className="small" style={{ marginTop: 6 }}>
                <b>อ้างอิง:</b> {showPage && <>หน้า {page}</>}{showPage && showRow && " "}{showRow && <>ลำดับ {row}</>}
              </div>
            )}

            {!!budgetUse.trim() && <div className="small" style={{ marginTop: 6 }}><b>การใช้งบ:</b> {budgetUse}</div>}
            {!!emergency.trim() && <div className="small" style={{ marginTop: 6, whiteSpace: "pre-wrap" }}><b>อำนาจอนุมัติ:</b> {emergency}</div>}
            {!!special.trim() && <div className="small" style={{ marginTop: 6, whiteSpace: "pre-wrap" }}><b>หมายเหตุ:</b> {special}</div>}
            {!!approvalCondition.trim() && <div className="small" style={{ marginTop: 6, whiteSpace: "pre-wrap" }}><b>เงื่อนไขการอนุมัติ:</b> <br />{approvalCondition}</div>}

            {links.length > 0 && (
              <div style={{ marginTop: 14 }}>
                <div className="small" style={{ marginBottom: 6 }}><b>ลิงก์อ้างอิง</b></div>
                <ul style={{ margin: 0, paddingLeft: 18 }}>
                  {links.map((link, idx) => (
                    <li key={idx} style={{ marginTop: 6 }}>
                      <a href={esc(link.realLink)} target="_blank" rel="noreferrer">
                        {esc(link.displayLine || link.realLink)}
                      </a>
                    </li>
                  ))}
                </ul>
              </div>
            )}
          </>
        )}
      </div>
    </main>
  );
}
