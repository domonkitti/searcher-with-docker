"use client";

import { useState } from "react";
import styles from "./ChecklistAssistant.module.css";

export default function ChecklistAssistant() {
  const [isOpen, setIsOpen] = useState(false);

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
          </div>

          <div className={styles.body}>
            {/* เว้นว่างไว้ก่อน */}
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