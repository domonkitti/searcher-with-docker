"use client";

import { useMemo, useState } from "react";
import styles from "./ChecklistAssistant.module.css";
import { useRouter } from "next/navigation";

type Step = 1 | 2 | 3 | 4;

export default function ChecklistAssistant() {
  const [isOpen, setIsOpen] = useState(false);
  const [step, setStep] = useState<Step>(1);

  const [foundInItemSearch, setFoundInItemSearch] = useState<boolean | null>(null);
  const [foundInCriteria, setFoundInCriteria] = useState<boolean | null>(null);
  const [lifeOverOneYear, setLifeOverOneYear] = useState(false);
  const [priceOverTenThousand, setPriceOverTenThousand] = useState(false);

  const router = useRouter();

  const progressPercent = step === 1 ? 25 : step === 2 ? 50 : step === 3 ? 75 : 100;
  const isInvestment = lifeOverOneYear && priceOverTenThousand;

  const showApprovalSection =
    foundInItemSearch === true ||
    foundInCriteria === true ||
    (foundInItemSearch === false && foundInCriteria === false && isInvestment);

  const summary = useMemo(() => {
    const principleNotes = [
      'หากเขตมีอำนาจตามที่ระบุไว้ใน "อำนาจอนุมัติ"',
      "หากมีการระบุให้พิจารณาความเหมาะสม ให้ส่งให้หน่วยงานนั้นๆ ก่อน",
    ];

    const principleApproval = [
      "ผู้มีอำนาจอนุมัติหลักการตามวงเงิน",
      "ผชก. ≤ 1 ล้านบาท",
      "รผก.(น,ฉ,ก,ต) > 1 ล้านบาท แต่ ≤ 5 ล้านบาท",
      "ผวก. วงเงิน > 5 ล้านบาท",
    ];

    const budgetNotes = ['หากเขตมีอำนาจตามที่ระบุไว้ใน "อำนาจอนุมัติ"'];

    const budgetApproval = [
      "ผู้มีอำนาจอนุมัติงบเงิน คือ",
      "ผชก.เขต / รผก.(น,ฉ,ก,ต)",
    ];

    if (foundInItemSearch === true) {
      return {
        variant: "info",
        result: "ดำเนินการตามเงื่อนไขที่ระบุภายในรายการที่พบ",
        principleNotes,
        principleApproval,
        budgetNotes,
        budgetApproval,
      };
    }

    if (foundInItemSearch === false && foundInCriteria === true) {
      return {
        variant: "info",
        result: "ดำเนินการตามเงื่อนไขที่ระบุภายในรายการที่พบ",
        principleNotes,
        principleApproval,
        budgetNotes,
        budgetApproval,
      };
    }

    return {
      variant: isInvestment ? "success" : "warning",
      result: isInvestment ? "เป็นงบลงทุน" : "เป็นงบทำการ",
      principleNotes,
      principleApproval,
      budgetNotes,
      budgetApproval,
    };
  }, [foundInItemSearch, foundInCriteria, isInvestment]);

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
                <p className={styles.subtitle}>ติดต่อสอบถาม 5370-5377</p>
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
                    <button
                      type="button"
                      className={styles.sectionDescLink}
                      onClick={() => {
                        setIsOpen(false);
                        router.push("/");
                      }}
                    >
                      ไปหน้าค้นหา
                    </button>
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
                    พบรายการ
                  </button>

                  <button
                    type="button"
                    className={styles.secondaryButton}
                    onClick={() => {
                      setFoundInItemSearch(false);
                      setStep(2);
                    }}
                  >
                    ไม่พบรายการ
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
                    <h3 className={styles.sectionTitle}>
                      ลองค้นหาหนังสือหลักเกณฑ์การจำแนกประเภทรายจ่ายตามงบประมาณ
                    </h3>
                    <button
                      type="button"
                      className={styles.sectionDescLink}
                      onClick={() => {
                        setIsOpen(false);
                        window.open(
                          "https://drive.google.com/file/d/1peCJ03dLvH7Z6ZMjyXJIOw_n_H2RvhjJ/view?pli=1"
                        );
                      }}
                    >
                      เปิดหนังสือหลักเกณฑ์ฯ
                    </button>
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
                    พบรายการ
                  </button>

                  <button
                    type="button"
                    className={styles.secondaryButton}
                    onClick={() => {
                      setFoundInCriteria(false);
                      setStep(3);
                    }}
                  >
                    ไม่พบรายการ
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
                    <p className={styles.sectionDesc}>ในกรณีไม่พบรายการ</p>
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

                {showApprovalSection && (
                  <div className={styles.approvalSection}>
                    <div className={styles.approvalBox}>
                      <h4 className={styles.approvalTitle}>อำนาจอนุมัติหลักการ</h4>

                      {summary.principleNotes.map((item, index) => (
                        <div key={`principle-note-${index}`} className={styles.approvalNote}>
                          {item}
                        </div>
                      ))}

                      <div className={styles.approvalInnerBox}>
                        {summary.principleApproval.map((item, index) => (
                          <div key={`principle-${index}`} className={styles.approvalItem}>
                            {item}
                          </div>
                        ))}
                      </div>
                    </div>

                    <div className={styles.approvalBox}>
                      <h4 className={styles.approvalTitle}>อำนาจอนุมัติงบเงิน</h4>

                      {summary.budgetNotes.map((item, index) => (
                        <div key={`budget-note-${index}`} className={styles.approvalNote}>
                          {item}
                        </div>
                      ))}

                      <div className={styles.approvalInnerBox}>
                        {summary.budgetApproval.map((item, index) => (
                          <div key={`budget-${index}`} className={styles.approvalItem}>
                            {item}
                          </div>
                        ))}
                      </div>
                    </div>
                  </div>
                )}
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