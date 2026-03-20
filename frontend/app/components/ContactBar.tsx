"use client";

export default function ContactBar() {
  return (
    <div
      style={{
        position: "fixed",
        bottom: 0,
        left: 0,
        right: 0,
        zIndex: 1000,
        background: "#111827",
        color: "#fff",
        borderTop: "1px solid rgba(255,255,255,0.1)",
        padding: "10px 16px",
      }}
    >
      <div
        style={{
          maxWidth: 1200,
          margin: "0 auto",
          display: "flex",
          justifyContent: "space-between",
          alignItems: "center",
          gap: 12,
          flexWrap: "wrap",
        }}
      >
        <div>ติดต่อสอบถาม: 02-590-5370-5377</div>
        <div>
        </div>
      </div>
    </div>
  );
}