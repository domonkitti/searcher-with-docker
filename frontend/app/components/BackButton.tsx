"use client";

import { useRouter } from "next/navigation";
import React from "react";

type Props = { className?: string; children?: React.ReactNode };

export default function BackButton({ className, children }: Props) {
  const router = useRouter();

  return (
    <button
      type="button"
      className={className ?? "backBtnModern"}
      onClick={() => {
        try {
          router.back();
        } catch {
          if (typeof window !== "undefined") window.history.back();
        }
      }}
      aria-label="ย้อนกลับ"
    >
      <svg
        width="20"
        height="20"
        viewBox="0 0 24 24"
        fill="none"
        xmlns="http://www.w3.org/2000/svg"
        aria-hidden="true"
      >
        <path
          d="M14 6L8 12L14 18"
          stroke="currentColor"
          strokeWidth="2.5"
          strokeLinecap="round"
          strokeLinejoin="round"
        />
      </svg>

      <span className="backLabelModern">
        {children ?? "ย้อนกลับ"}
      </span>
    </button>
  );
}