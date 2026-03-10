"use client";

import { useEffect, useState } from "react";
import Navbar from "../components/Navbar";

const API = process.env.NEXT_PUBLIC_API_BASE || "";
type RuleInput = { key: string; label: string; type: string; unit?: string; default?: number };
type RuleConfig = {
  inputs: RuleInput[];
  rules: any[];
  budgetAllTrue: string;
  budgetOtherwise: string;
  logicNote: string;
};

export default function RulesPage() {
  const [cfg, setCfg] = useState<RuleConfig | null>(null);
  const [values, setValues] = useState<Record<string, number>>({});
  const [result, setResult] = useState<any>(null);

  useEffect(() => {
    fetch(`${API}/api/rules/config`).then(r => r.json()).then((d: RuleConfig) => {
      setCfg(d);
      const init: Record<string, number> = {};
      for (const i of (d.inputs || [])) init[i.key] = Number(i.default ?? 0);
      setValues(init);
    });
  }, []);

  async function evalNow() {
    const res = await fetch(`${API}/api/rules/eval`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(values),
    });
    const data = await res.json();
    setResult(data);
  }

  return (
    <main className="wrap">
      <Navbar />

      <div className="card">
        <div className="title">กรณีไม่พบผลลัพธ์</div>
        {/* <div className="small" style={{ marginTop: 6 }}>กรอกข้อมูลเพื่อช่วยตัดสินประเภทงบ</div> */}

        <div className="card" style={{ marginTop: 12 }}>
          {/*
          {!cfg ? (
            <div className="small">Loading…</div>
          ) : (
            <>
              {(cfg.inputs || []).map((inp) => (
                <div className="field" key={inp.key}>
                  <div className="label">{inp.label}</div>
                  <div className="row">
                    <input
                      type="number"
                      value={values[inp.key] ?? 0}
                      onChange={(e) => setValues({ ...values, [inp.key]: Number(e.target.value) })}
                    />
                    <div className="unit">{inp.unit || ""}</div>
                  </div>
                </div>
              ))}

              <button className="btnPrimary" onClick={evalNow}>ประเมิน</button>
            </>
          )}
          */}

          {result && (
            <div className="card" style={{ marginTop: 12 }}>
              <div className="small"><b>ผลลัพธ์:</b> {result.budgetType}</div>
              <div className="small" style={{ marginTop: 6 }}><b>เงื่อนไข:</b></div>
              <ul className="small">
                {(result.conditions || []).map((c: any, idx: number) => (
                  <li key={idx}>
                    {c.label} {c.op} {c.value} → {c.ok ? "true" : "false"}
                  </li>
                ))}
              </ul>

              <div className="small" style={{ marginTop: 10 }}>
                <b>Logic note:</b> {result.logicNote || "wait for more information"}
              </div>
            </div>
          )}

          <div className="small" style={{ marginTop: 12 }}>
            หากไม่พบรายการที่ตรงกับของที่ค้นให้ให้ใช้ ราคาและอายุการใช้งานในการประเมินโดย
            <br />- ราคาเกิน 10,000 บาท ต่อ ชิ้น และ อายุการใช้งานเกิน 1 ปี ให้จัดเป็นงบลงทุน
            <br />- ราคาไม่เกิน 10,000 บาท ต่อ ชิ้น ให้จัดเป็นงบทำการ
            <br />- อายุการใช้งานไม่เกิน 1 ปี ให้จัดเป็นงบทำการ
          </div>
        </div>
      </div>
    </main>
  );
}
