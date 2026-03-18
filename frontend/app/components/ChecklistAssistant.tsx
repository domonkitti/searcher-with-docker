"use client";

import { useMemo, useState } from "react";
import styles from "./ChecklistAssistant.module.css";

type Step = 1 | 2 | 3 | 4;

export default function ChecklistAssistant() {
  const [isOpen, setIsOpen] = useState(false);
  const [step, setStep] = useState<Step>(1);

  const [foundInItemSearch, setFoundInItemSearch] = useState<boolean | null>(null);
  const [foundInCriteria, setFoundInCriteria] = useState<boolean | null>(null);
  const [lifeOverOneYear, setLifeOverOneYear] = useState(false);
  const [priceOverTenThousand, setPriceOverTenThousand] = useState(false);

  const progressPercent = step === 1 ? 25 : step === 2 ? 50 : step === 3 ? 75 : 100;

  const summary = useMemo(() => {
    if (foundInItemSearch === true) {
      return {
        variant: "info",
        result: "พบรายการที่ต้องการแล้ว",
        details: [
          "ข้อ 1: พบรายการที่ต้องการจากการค้นหา",
          "ข้อ 2: ไม่จำเป็นต้องตรวจต่อ",
          "ข้อ 3: ไม่จำเป็นต้องตรวจต่อ",
          "คำแนะนำ: ให้ดำเนินการตามผลจากรายการที่ค้นพบ",
        ],
      };
    }

    if (foundInItemSearch === false && foundInCriteria === true) {
      return {
        variant: "info",
        result: "พบในหลักเกณฑ์พิจารณา",
        details: [
          "ข้อ 1: ไม่พบรายการที่ต้องการจากการค้นหา",
          "ข้อ 2: พบในหลักเกณฑ์พิจารณา",
          "ข้อ 3: ไม่จำเป็นต้องตรวจต่อ",
          "คำแนะนำ: ให้ดำเนินการตามผลในหลักเกณฑ์พิจารณา",
        ],
      };
    }

    const isInvestment = lifeOverOneYear && priceOverTenThousand;

    return {
      variant: isInvestment ? "success" : "warning",
      result: isInvestment ? "เป็นงบลงทุน" : "เป็นงบทำการ",
      details: [
        "ข้อ 1: ไม่พบรายการที่ต้องการจากการค้นหา",
        "ข้อ 2: ไม่พบในหลักเกณฑ์พิจารณา",
        `ข้อ 3.1: อายุใช้งานมากกว่า 1 ปี = ${lifeOverOneYear ? "ใช่" : "ไม่ใช่"}`,
        `ข้อ 3.2: ราคาต่อชิ้นมากกว่า 10,000 บาท = ${priceOverTenThousand ? "ใช่" : "ไม่ใช่"}`,
        isInvestment
          ? "คำแนะนำ: เข้าหลักเกณฑ์เป็นงบลงทุน เพราะผ่านทั้ง 2 เงื่อนไข"
          : "คำแนะนำ: ไม่ผ่านครบทั้ง 2 เงื่อนไข จึงเป็นงบทำการ",
      ],
    };
  }, [foundInItemSearch, foundInCriteria, lifeOverOneYear, priceOverTenThousand]);

  const resetAll = () => {
    setStep(1);
    setFoundInItemSearch(null);
    setFoundInCriteria(null);
    setLifeOverOneYear(false);
    setPriceOverTenThousand(false);
  };

  const goBack = () => {
    if (step === 4) {
      if (foundInItemSearch === true) {
        setStep(1);
        return;
      }
      if (foundInCriteria === true) {
        setStep(2);
        return;
      }
      setStep(3);
      return;
    }

    if (step === 3) {
      setStep(2);
      return;
    }

    if (step === 2) {
      setStep(1);
    }
  };

  return (
    <>
      {isOpen && (
        <button
          type="button"
          aria-label="ปิดตัวช่วยตรวจสอบ"
          className={styles.overlay}
          onClick={() => setIsOpen(false)}
        />
      )}

      {isOpen ? (
        <aside className={styles.panel}>
          <div className={styles.header}>
            <div className={styles.headerTop}>
              <div>
                <h2 className={styles.title}>ตัวช่วยตรวจสอบงบประมาณ</h2>
                <p className={styles.subtitle}>ใช้เป็นตัวช่วยไล่ดูว่าควรตรวจตรงไหนก่อน</p>
              </div>

              <button
                type="button"
                className={styles.closeButton}
                onClick={() => setIsOpen(false)}
                aria-label="ปิด"
              >
                ×
              </button>
            </div>

            <div className={styles.progressTrack}>
              <div
                className={styles.progressFill}
                style={{ width: `${progressPercent}%` }}
              />
            </div>
          </div>

          <div className={styles.body}>
            {step === 1 && (
              <section className={styles.card}>
                <div className={styles.sectionHead}>
                  <div className={styles.iconBox}>🔎</div>
                  <div>
                    <p className={styles.eyebrow}>ข้อ 1</p>
                    <h3 className={styles.sectionTitle}>ลองค้นหารายการที่ต้องการก่อน</h3>
                    <p className={styles.sectionDesc}>
                      หากพบรายการที่ตรงแล้ว สามารถสรุปผลได้ทันที
                    </p>
                  </div>
                </div>

                <div className={styles.actionRow}>
                  <button
                    type="button"
                    className={styles.primaryButton}
                    onClick={() => {
                      setFoundInItemSearch(true);
                      setFoundInCriteria(null);
                      setLifeOverOneYear(false);
                      setPriceOverTenThousand(false);
                      setStep(4);
                    }}
                  >
                    เจอรายการ
                  </button>

                  <button
                    type="button"
                    className={styles.secondaryButton}
                    onClick={() => {
                      setFoundInItemSearch(false);
                      setStep(2);
                    }}
                  >
                    ไม่เจอ
                  </button>
                </div>
              </section>
            )}

            {step === 2 && (
              <section className={styles.card}>
                <div className={styles.sectionHead}>
                  <div className={styles.iconBox}>📋</div>
                  <div>
                    <p className={styles.eyebrow}>ข้อ 2</p>
                    <h3 className={styles.sectionTitle}>ลองค้นหาในหลักเกณฑ์พิจารณา</h3>
                    <p className={styles.sectionDesc}>
                      ถ้าพบเงื่อนไขที่ตรง ให้ไปสรุปผลได้เลย
                    </p>
                  </div>
                </div>

                <div className={styles.actionRow}>
                  <button
                    type="button"
                    className={styles.primaryButton}
                    onClick={() => {
                      setFoundInCriteria(true);
                      setLifeOverOneYear(false);
                      setPriceOverTenThousand(false);
                      setStep(4);
                    }}
                  >
                    เจอ
                  </button>

                  <button
                    type="button"
                    className={styles.secondaryButton}
                    onClick={() => {
                      setFoundInCriteria(false);
                      setStep(3);
                    }}
                  >
                    ไม่เจอ
                  </button>
                </div>
              </section>
            )}

            {step === 3 && (
              <section className={styles.card}>
                <div className={styles.sectionHead}>
                  <div className={styles.iconBox}>💼</div>
                  <div>
                    <p className={styles.eyebrow}>ข้อ 3</p>
                    <h3 className={styles.sectionTitle}>ตรวจสอบเงื่อนไขงบลงทุน</h3>
                    <p className={styles.sectionDesc}>
                      ต้องผ่านทั้ง 2 ข้อ จึงจะสรุปเป็นงบลงทุน
                    </p>
                  </div>
                </div>

                <label className={styles.checkItem}>
                  <input
                    type="checkbox"
                    checked={lifeOverOneYear}
                    onChange={(e) => setLifeOverOneYear(e.target.checked)}
                  />
                  <div>
                    <p className={styles.checkTitle}>อายุใช้งานมากกว่า 1 ปี</p>
                    <p className={styles.checkDesc}>ติ๊กเมื่อรายการมีอายุใช้งานเกิน 1 ปี</p>
                  </div>
                </label>

                <label className={styles.checkItem}>
                  <input
                    type="checkbox"
                    checked={priceOverTenThousand}
                    onChange={(e) => setPriceOverTenThousand(e.target.checked)}
                  />
                  <div>
                    <p className={styles.checkTitle}>ราคาต่อชิ้นมากกว่า 10,000 บาท</p>
                    <p className={styles.checkDesc}>ติ๊กเมื่อราคาต่อชิ้นเกิน 10,000 บาท</p>
                  </div>
                </label>

                <button
                  type="button"
                  className={`${styles.primaryButton} ${styles.fullWidth}`}
                  onClick={() => setStep(4)}
                >
                  สรุปผล
                </button>
              </section>
            )}

            {step === 4 && (
              <section className={styles.card}>
                <div className={styles.sectionHead}>
                  <div className={styles.iconBox}>✅</div>
                  <div>
                    <p className={styles.eyebrow}>Summary</p>
                    <h3 className={styles.sectionTitle}>สรุปผลการตรวจสอบ</h3>
                  </div>
                </div>

                <div className={`${styles.summaryBox} ${styles[summary.variant]}`}>
                  {summary.result}
                </div>

                <div className={styles.summaryList}>
                  {summary.details.map((detail, index) => (
                    <div key={`${detail}-${index}`} className={styles.summaryItem}>
                      {detail}
                    </div>
                  ))}
                </div>
              </section>
            )}
          </div>

          <div className={styles.footer}>
            <button
              type="button"
              className={styles.footerSecondary}
              onClick={goBack}
              disabled={step === 1}
            >
              ← ย้อนกลับ
            </button>

            <button
              type="button"
              className={styles.footerPrimary}
              onClick={resetAll}
            >
              ↻ เริ่มใหม่
            </button>
          </div>
        </aside>
      ) : (
        <div className={styles.collapsedWrap}>
          <button
            type="button"
            className={styles.collapsedButton}
            onClick={() => setIsOpen(true)}
          >
            ตัวช่วยตรวจสอบ
          </button>
        </div>
      )}
    </>
  );
}