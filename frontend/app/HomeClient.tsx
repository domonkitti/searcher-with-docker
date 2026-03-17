"use client";

import { useEffect, useMemo, useRef, useState } from "react";
import Navbar from "./components/Navbar";
import { useRouter, useSearchParams } from "next/navigation";

const API = process.env.NEXT_PUBLIC_API_BASE || "";

type SearchResult = { id: string; title: string; score: number; meta: any };
type SuggestItem = { text: string; score: number };

function esc(s: any) {
  return (s ?? "").toString();
}

function hasMeaningfulValue(v: any) {
  const s = (v ?? "").toString().trim();
  return s !== "" && s !== "-" && s.toLowerCase() !== "null" && s.toLowerCase() !== "undefined";
}

function normalizeMultiline(s: any) {
  return esc(s).replace(/\r\n/g, "\n").replace(/\r/g, "\n").trim();
}

function escapeRegExp(s: string) {
  return s.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
}

function tokenizeQuery(q: string) {
  return q
    .trim()
    .toLowerCase()
    .split(/\s+/)
    .map((x) => x.trim())
    .filter(Boolean);
}

function buildSnippet(text: string, query: string, radius = 60) {
  const raw = normalizeMultiline(text);
  if (!raw) return "";

  const flat = raw.replace(/\n+/g, " ");
  const tokens = tokenizeQuery(query);

  if (tokens.length === 0) {
    return flat.length > radius * 2 ? flat.slice(0, radius * 2).trimEnd() + "..." : flat;
  }

  let bestIdx = -1;
  for (const t of tokens) {
    const idx = flat.toLowerCase().indexOf(t.toLowerCase());
    if (idx >= 0 && (bestIdx === -1 || idx < bestIdx)) {
      bestIdx = idx;
    }
  }

  if (bestIdx === -1) {
    return flat.length > radius * 2 ? flat.slice(0, radius * 2).trimEnd() + "..." : flat;
  }

  const start = Math.max(0, bestIdx - radius);
  const end = Math.min(flat.length, bestIdx + radius);
  let snippet = flat.slice(start, end).trim();

  if (start > 0) snippet = "..." + snippet;
  if (end < flat.length) snippet = snippet + "...";

  return snippet;
}

function HighlightedSnippet({ snippet, query }: { snippet: string; query: string }) {
  const tokens = tokenizeQuery(query).filter((t) => t.length > 0);

  if (!snippet) return null;
  if (tokens.length === 0) return <>{snippet}</>;

  const pattern = new RegExp(`(${tokens.map(escapeRegExp).join("|")})`, "gi");
  const parts = snippet.split(pattern);

  return (
    <>
      {parts.map((part, i) => {
        const matched = tokens.some((t) => part.toLowerCase() === t.toLowerCase());
        return matched ? (
          <mark
            key={i}
            style={{
              backgroundColor: "#fff3b0",
              padding: "0 2px",
              borderRadius: 2,
            }}
          >
            {part}
          </mark>
        ) : (
          <span key={i}>{part}</span>
        );
      })}
    </>
  );
}
export default function HomeClient() {
  const [q, setQ] = useState("");
  const [items, setItems] = useState<SearchResult[]>([]);
  const [done, setDone] = useState(false);

  const [suggestions, setSuggestions] = useState<SuggestItem[]>([]);
  const [activeSuggest, setActiveSuggest] = useState(-1);
  const [isSuggestOpen, setIsSuggestOpen] = useState(false);

  const router = useRouter();
  const searchParams = useSearchParams();

  const suggestTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const blurTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const boxRef = useRef<HTMLDivElement | null>(null);

  const suppressSuggestRef = useRef(false);

  async function runSearch(term: string) {
    setDone(false);
    const res = await fetch(`${API}/api/search?q=${encodeURIComponent(term)}&k=20`);
    const data = await res.json();
    setItems(data.results || []);
    setDone(true);
  }

  async function runSuggest(term: string) {
    const clean = term.trim();

    if (!clean) {
      setSuggestions([]);
      setActiveSuggest(-1);
      setIsSuggestOpen(false);
      return;
    }

    try {
      const res = await fetch(`${API}/api/suggest?q=${encodeURIComponent(clean)}&k=8`);
      const data = await res.json();
      const nextItems = data.items || [];

      setSuggestions(nextItems);
      setActiveSuggest(-1);
      setIsSuggestOpen(nextItems.length > 0);
    } catch {
      setSuggestions([]);
      setActiveSuggest(-1);
      setIsSuggestOpen(false);
    }
  }

  async function search(termArg?: string) {
    const term = (termArg ?? q).trim();
    if (!term) return;

    setQ(term);
    setSuggestions([]);
    setActiveSuggest(-1);
    setIsSuggestOpen(false);

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

  useEffect(() => {
    if (suggestTimer.current) clearTimeout(suggestTimer.current);

    if (suppressSuggestRef.current) {
      suppressSuggestRef.current = false;
      return;
    }

    suggestTimer.current = setTimeout(() => {
      runSuggest(q);
    }, 180);

    return () => {
      if (suggestTimer.current) clearTimeout(suggestTimer.current);
    };
  }, [q]);

  useEffect(() => {
    function handleOutsideClick(e: MouseEvent) {
      const target = e.target as Node | null;
      if (!boxRef.current || !target) return;

      if (!boxRef.current.contains(target)) {
        setIsSuggestOpen(false);
        setActiveSuggest(-1);
      }
    }

    document.addEventListener("mousedown", handleOutsideClick);
    return () => {
      document.removeEventListener("mousedown", handleOutsideClick);
    };
  }, []);

  useEffect(() => {
    return () => {
      if (suggestTimer.current) clearTimeout(suggestTimer.current);
      if (blurTimer.current) clearTimeout(blurTimer.current);
    };
  }, []);

  const showSuggest = useMemo(() => {
    return isSuggestOpen && q.trim().length > 0 && suggestions.length > 0;
  }, [isSuggestOpen, q, suggestions]);

  return (
    <main className="wrap">
      <Navbar />

      <div className="logo">
        ค้นหา<span>รายการในหนังสือจำแนก</span>
      </div>

      <div className="searchBox" style={{ position: "relative" }} ref={boxRef}>
        <input
          id="q"
          type="text"
          placeholder="พิมพ์เพื่อค้นหา…"
          autoComplete="off"
          value={q}
          onFocus={() => {
            if (blurTimer.current) clearTimeout(blurTimer.current);
            if (suggestions.length > 0) {
              setIsSuggestOpen(true);
            }
          }}
          onBlur={() => {
            if (blurTimer.current) clearTimeout(blurTimer.current);
            blurTimer.current = setTimeout(() => {
              setIsSuggestOpen(false);
              setActiveSuggest(-1);
            }, 120);
          }}
          onChange={(e) => {
            suppressSuggestRef.current = false;
            setQ(e.target.value);
            setIsSuggestOpen(true);
          }}
          onKeyDown={(e) => {
            if (e.key === "ArrowDown") {
              e.preventDefault();
              setIsSuggestOpen(true);
              setActiveSuggest((prev) => Math.min(prev + 1, suggestions.length - 1));
              return;
            }

            if (e.key === "ArrowUp") {
              e.preventDefault();
              setActiveSuggest((prev) => Math.max(prev - 1, -1));
              return;
            }

            if (e.key === "Enter") {
              e.preventDefault();

              if (activeSuggest >= 0 && suggestions[activeSuggest]) {
                suppressSuggestRef.current = true;
                search(suggestions[activeSuggest].text);
              } else {
                search();
              }
              return;
            }

            if (e.key === "Escape") {
              setSuggestions([]);
              setActiveSuggest(-1);
              setIsSuggestOpen(false);
            }
          }}
        />

        <button id="btn" onClick={() => search()}>
          ค้นหา
        </button>

        {showSuggest && (
          <div
            style={{
              position: "absolute",
              top: "100%",
              left: 0,
              right: 0,
              background: "white",
              border: "1px solid #e5e7eb",
              borderRadius: 12,
              marginTop: 8,
              boxShadow: "0 10px 30px rgba(0,0,0,0.08)",
              zIndex: 30,
              overflow: "hidden",
            }}
          >
            {suggestions.map((item, idx) => (
              <button
                key={`${item.text}-${idx}`}
                type="button"
                onMouseDown={(e) => {
                  e.preventDefault();
                  suppressSuggestRef.current = true;
                }}
                onClick={() => {
                  setSuggestions([]);
                  setActiveSuggest(-1);
                  setIsSuggestOpen(false);
                  search(item.text);
                }}
                style={{
                  display: "block",
                  width: "100%",
                  textAlign: "left",
                  padding: "12px 14px",
                  border: "none",
                  cursor: "pointer",
                  background: idx === activeSuggest ? "#f3f4f6" : "white",
                  color: "#111827",
                  fontSize: 16,
                }}
              >
                {item.text}
              </button>
            ))}
          </div>
        )}
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
          const description = esc(m.description || "");
          const page = esc(m.page || "-");
          const row = esc(m.row || "-");
          const score = esc(r.score);
          const sourceId = esc(m.sourceId || "").trim().toLowerCase();
          const snippet = buildSnippet(description, q);
          return (
            <div className="result" key={r.id}>
              {sourceId ? (
                <a className="title" href={`/doc/${encodeURIComponent(sourceId)}`}>
                  {esc(r.title)}
                </a>
              ) : (
                <div className="title">{esc(r.title)}</div>
              )}

              {!!snippet && (
                <div
                  style={{
                    marginTop: 8,
                    color: "#4b5563",
                    lineHeight: 1.6,
                    fontSize: 15,
                  }}
                >
                  <HighlightedSnippet snippet={description} query={q} />
                </div>
              )}

              <div className="meta" style={{ marginTop: snippet ? 10 : 6 }}>
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