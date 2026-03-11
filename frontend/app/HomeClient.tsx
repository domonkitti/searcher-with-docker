"use client";

import { useEffect, useState } from "react";
import Navbar from "./components/Navbar";
import { useRouter, useSearchParams } from "next/navigation";

const API = process.env.NEXT_PUBLIC_API_BASE || "";

type SearchResult = { id: string; title: string; score: number; meta: any };

function esc(s: any) {
  return (s ?? "").toString();
}

function hasMeaningfulValue(v: any) {
  const s = (v ?? "").toString().trim();
  return s !== "" && s !== "-" && s.toLowerCase() !== "null" && s.toLowerCase() !== "undefined";
}

export default function HomeClient() {
  const [q, setQ] = useState("");
  const [items, setItems] = useState<SearchResult[]>([]);
  const [done, setDone] = useState(false);
  const router = useRouter();
  const searchParams = useSearchParams();

  async function runSearch(term: string) {
    setDone(false);
    const res = await fetch(`${API}/api/search?q=${encodeURIComponent(term)}&k=20`);
    const data = await res.json();
    setItems(data.results || []);
    setDone(true);
  }

  async function search() {
    const term = q.trim();
    if (!term) return;

    await runSearch(term);
    try {
      const url = new URL(window.location.href);
      url.searchParams.set("q", term);
      router.push(url.pathname + url.search);
    } catch {}
  }

  useEffect(() => {
    const initial = searchParams.get("q") || "";
    if (!initial) return;
    setQ(initial);
    runSearch(initial);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  return (
    <main className="wrap">
      <Navbar />

      <div className="logo">
        ค้นหา<span>รายการในหนังสือจำแนก</span>
      </div>

      <div className="searchBox">
        <input
          id="q"
          type="text"
          placeholder="พิมพ์เพื่อค้นหา…"
          autoComplete="off"
          value={q}
          onChange={(e) => setQ(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === "Enter") search();
          }}
        />
        <button id="btn" onClick={search}>
          ค้นหา
        </button>
      </div>

      <div className="topLinks">
        <a className="linkish" href="/rules">
          เงื่อนไขกรณีที่ไม่พบผลลัพธ์
        </a>
      </div>

      <section className="results">
        {done && items.length === 0 && (
          <div className="result">
            <div className="meta">ไม่พบผลลัพธ์</div>
          </div>
        )}

        {items.map((r) => {
          const m = r.meta || {};
          const categoryMain = esc(m.categoryMain || "-");
          const categorySub = esc(m.categorySub || "");
          const group = esc(m.group || "");
          const page = esc(m.page || "-");
          const row = esc(m.row || "-");
          const score = esc(r.score);

          return (
            <div className="result" key={r.id}>
              <a className="title" href={`/doc/${encodeURIComponent(r.id)}`}>
                {esc(r.title)}
              </a>

              <div className="meta">
                <div>หมวด: {categoryMain}</div>
                {!!categorySub.trim() && <div>หมวดย่อย: {categorySub}</div>}
                {!!group.trim() && <div>กลุ่มรายการ: {group}</div>}

                {(hasMeaningfulValue(page) || hasMeaningfulValue(row) || hasMeaningfulValue(score)) && (
                  <div>
                    อ้างอิง:
                    {hasMeaningfulValue(page) && <> หน้า {page}</>}
                    {hasMeaningfulValue(row) && <> ลำดับ {row}</>}
                    {hasMeaningfulValue(score) && <> | score={score}</>}
                  </div>
                )}
              </div>
            </div>
          );
        })}
      </section>
    </main>
  );
}