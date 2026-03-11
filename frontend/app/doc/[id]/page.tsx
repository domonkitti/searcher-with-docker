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
  const id = (params?.id as string) || "";

  const [doc, setDoc] = useState<Doc | null>(null);
  const [loading, setLoading] = useState(true);

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
  const page = esc(m.page || "");
  const row = esc(m.row || "");
  const budgetUse = esc(m.budgetUse || "");
  const emergency = esc(m.emergency || "");
  const special = esc(m.special || "");

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

            <div className="small" style={{ marginTop: 8 }}>
              <div>
                <b>หมวด:</b> {catMain}
              </div>

              {!!catSub.trim() && (
                <div style={{ marginTop: 4 }}>
                  <b>หมวดย่อย:</b> {catSub}
                </div>
              )}

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

            {showReference && (
              <div className="small" style={{ marginTop: 6 }}>
                <b>อ้างอิง:</b>{" "}
                {showPage && <>หน้า {page}</>}
                {showPage && showRow && " "}
                {showRow && <>ลำดับ {row}</>}
              </div>
            )}

            {!!budgetUse.trim() && (
              <div className="small" style={{ marginTop: 6 }}>
                <b>การใช้งบ:</b> {budgetUse}
              </div>
            )}

            {!!emergency.trim() && (
              <div className="small" style={{ marginTop: 6, whiteSpace: "pre-wrap" }}>
                <b>อำนาจอนุมัติ:</b> {emergency}
              </div>
            )}

            {!!special.trim() && (
              <div className="card" style={{ marginTop: 12 }}>
                <div className="small">
                  <b>หมายเหตุ</b>
                </div>
                <div className="small" style={{ marginTop: 8, whiteSpace: "pre-wrap" }}>
                  {special}
                </div>
              </div>
            )}

            {links.length > 0 && (
              <div className="card" style={{ marginTop: 12 }}>
                <div className="small">
                  <b>ลิงก์ที่เกี่ยวข้อง</b>
                </div>

                <ul style={{ marginTop: 8, paddingLeft: 18 }}>
                  {links.map((link, idx) => {
                    const href = esc(link.realLink).trim();
                    const label = esc(link.displayLine).trim();

                    if (!href || !label) return null;

                    return (
                      <li key={`${href}-${idx}`} style={{ marginTop: 6 }}>
                        <a
                          href={href}
                          target="_blank"
                          rel="noopener noreferrer"
                        >
                          {label}
                        </a>
                      </li>
                    );
                  })}
                </ul>
              </div>
            )}
          </>
        )}
      </div>
    </main>
  );
}