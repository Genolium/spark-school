import { ImageResponse } from "next/og";

export const runtime = "edge";
export const alt = "так называемый SPARK — Грант $20,000 на учёбу в США";
export const size = {
  width: 1200,
  height: 630,
};
export const contentType = "image/png";

export default async function Image() {
  return new ImageResponse(
    (
      <div
        style={{
          height: "100%",
          width: "100%",
          display: "flex",
          flexDirection: "column",
          justifyContent: "space-between",
          padding: "60px 80px",
          backgroundColor: "#090B0E",
          color: "#FFFFFF",
          fontFamily: "sans-serif",
          position: "relative",
          overflow: "hidden",
        }}
      >
        {/* Glow ambient background spots */}
        <div
          style={{
            position: "absolute",
            top: "-80px",
            right: "-80px",
            width: "550px",
            height: "550px",
            borderRadius: "50%",
            backgroundColor: "rgba(16, 185, 129, 0.22)",
            filter: "blur(110px)",
          }}
        />
        <div
          style={{
            position: "absolute",
            bottom: "-100px",
            left: "-60px",
            width: "550px",
            height: "550px",
            borderRadius: "50%",
            backgroundColor: "rgba(56, 189, 248, 0.16)",
            filter: "blur(110px)",
          }}
        />

        {/* Top Header Pill */}
        <div
          style={{
            display: "flex",
            alignItems: "center",
            justifyContent: "space-between",
            width: "100%",
            zIndex: 10,
          }}
        >
          <div style={{ display: "flex", alignItems: "center", gap: "16px" }}>
            {/* Spark Logo Icon SVG */}
            <div
              style={{
                display: "flex",
                alignItems: "center",
                justifyContent: "center",
                width: "46px",
                height: "46px",
                borderRadius: "14px",
                backgroundColor: "#10B981",
                color: "#FFFFFF",
                boxShadow: "0 0 25px rgba(16, 185, 129, 0.4)",
              }}
            >
              <svg width="26" height="26" viewBox="0 0 24 24" fill="none">
                <path
                  d="M12 2L14.8 9.2L22 12L14.8 14.8L12 22L9.2 14.8L2 12L9.2 9.2L12 2Z"
                  fill="#FFFFFF"
                />
              </svg>
            </div>
            <span
              style={{
                fontSize: "26px",
                fontWeight: 800,
                color: "#FFFFFF",
                letterSpacing: "-0.5px",
              }}
            >
              так называемый SPARK
            </span>
          </div>

          {/* Status Badge */}
          <div
            style={{
              display: "flex",
              alignItems: "center",
              gap: "8px",
              padding: "8px 20px",
              borderRadius: "9999px",
              backgroundColor: "rgba(16, 185, 129, 0.12)",
              border: "1px solid rgba(16, 185, 129, 0.35)",
              color: "#34D399",
              fontSize: "15px",
              fontWeight: 700,
            }}
          >
            <div
              style={{
                width: "8px",
                height: "8px",
                borderRadius: "50%",
                backgroundColor: "#34D399",
              }}
            />
            <span>Набор на поток 2027 открыт</span>
          </div>
        </div>

        {/* Central Bold Editorial Headline */}
        <div
          style={{
            display: "flex",
            flexDirection: "column",
            gap: "10px",
            zIndex: 10,
            marginTop: "10px",
          }}
        >
          <div
            style={{
              fontSize: "68px",
              fontWeight: 900,
              lineHeight: 1.04,
              letterSpacing: "-2px",
              color: "#FFFFFF",
            }}
          >
            ВЫИГРАЙ ГРАНТ $20,000
          </div>
          <div
            style={{
              fontSize: "68px",
              fontWeight: 900,
              lineHeight: 1.04,
              letterSpacing: "-2px",
              color: "#34D399",
            }}
          >
            НА УЧЁБУ В США ЭТИМ ЛЕТОМ
          </div>
          <div
            style={{
              fontSize: "23px",
              fontWeight: 400,
              color: "#E2E8F0",
              marginTop: "16px",
              maxWidth: "920px",
              lineHeight: 1.45,
            }}
          >
            Практический акселератор от финалиста SPARK 2026. Разборы победных заявок,
            резюме Гарварда, симуляция 45-мин интервью в Zoom и инсайды отбора.
          </div>
        </div>

        {/* Bottom Feature Badges Bar */}
        <div
          style={{
            display: "flex",
            alignItems: "center",
            justifyContent: "space-between",
            zIndex: 10,
            paddingTop: "28px",
            borderTop: "1px solid rgba(255, 255, 255, 0.12)",
          }}
        >
          <div style={{ display: "flex", gap: "14px" }}>
            <div
              style={{
                display: "flex",
                alignItems: "center",
                gap: "8px",
                padding: "7px 14px",
                borderRadius: "10px",
                backgroundColor: "rgba(255, 255, 255, 0.05)",
                border: "1px solid rgba(255, 255, 255, 0.1)",
                color: "#FBBF24",
                fontSize: "14px",
                fontWeight: 700,
              }}
            >
              <svg width="17" height="17" viewBox="0 0 24 24" fill="#FBBF24">
                <polygon points="12 2 15.09 8.26 22 9.27 17 14.14 18.18 21.02 12 17.77 5.82 21.02 7 14.14 2 9.27 8.91 8.26 12 2" />
              </svg>
              <span>Грант $20,000 (100% покрытие)</span>
            </div>

            <div
              style={{
                display: "flex",
                alignItems: "center",
                gap: "8px",
                padding: "7px 14px",
                borderRadius: "10px",
                backgroundColor: "rgba(255, 255, 255, 0.05)",
                border: "1px solid rgba(255, 255, 255, 0.1)",
                color: "#34D399",
                fontSize: "14px",
                fontWeight: 700,
              }}
            >
              <svg width="17" height="17" viewBox="0 0 24 24" fill="none" stroke="#34D399" strokeWidth="3" strokeLinecap="round" strokeLinejoin="round">
                <polyline points="20 6 9 17 4 12" />
              </svg>
              <span>Мок-интервью 1-на-1 в Zoom</span>
            </div>

            <div
              style={{
                display: "flex",
                alignItems: "center",
                gap: "8px",
                padding: "7px 14px",
                borderRadius: "10px",
                backgroundColor: "rgba(255, 255, 255, 0.05)",
                border: "1px solid rgba(255, 255, 255, 0.1)",
                color: "#38BDF8",
                fontSize: "14px",
                fontWeight: 700,
              }}
            >
              <svg width="17" height="17" viewBox="0 0 24 24" fill="#38BDF8">
                <polygon points="13 2 3 14 12 14 11 22 21 10 12 10 13 2" />
              </svg>
              <span>Duolingo 130+ & Резюме</span>
            </div>
          </div>

          <div
            style={{
              fontSize: "19px",
              fontWeight: 800,
              color: "#FFFFFF",
              letterSpacing: "-0.5px",
              whiteSpace: "nowrap",
            }}
          >
            so-called-spark.ru
          </div>
        </div>
      </div>
    ),
    {
      ...size,
    }
  );
}
