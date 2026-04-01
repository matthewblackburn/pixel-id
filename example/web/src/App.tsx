import { useState } from "react";
import { PixelAvatar } from "pixel-id/react";
import { derive, maxGridSize } from "pixel-id";

interface GeneratedId {
  id: string;
  timestamp: string;
  machineId: number;
  sequence: number;
}

interface AvatarSettings {
  grid: number;
  colors: number;
  curves: boolean;
}

export function App() {
  const [ids, setIds] = useState<GeneratedId[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [settings, setSettings] = useState<AvatarSettings>({
    grid: 5,
    colors: 1,
    curves: false,
  });

  const maxGrid = maxGridSize(settings.colors, settings.curves);

  async function generate() {
    setLoading(true);
    setError(null);
    try {
      const res = await fetch("/api/id", { method: "POST" });
      if (!res.ok) throw new Error(await res.text());
      const data: GeneratedId = await res.json();
      setIds((prev) => [data, ...prev]);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to generate ID");
    } finally {
      setLoading(false);
    }
  }

  async function generateBatch() {
    setLoading(true);
    setError(null);
    try {
      const results = await Promise.all(
        Array.from({ length: 10 }, () =>
          fetch("/api/id", { method: "POST" }).then((r) => r.json()),
        ),
      );
      setIds((prev) => [...results, ...prev]);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to generate IDs");
    } finally {
      setLoading(false);
    }
  }

  function updateGrid(v: number) {
    const max = maxGridSize(settings.colors, settings.curves);
    setSettings((s) => ({ ...s, grid: Math.min(v, max) }));
  }

  function updateColors(v: number) {
    const max = maxGridSize(v, settings.curves);
    setSettings((s) => ({
      ...s,
      colors: v,
      grid: Math.min(s.grid, max),
    }));
  }

  function updateCurves(v: boolean) {
    const max = maxGridSize(settings.colors, v);
    setSettings((s) => ({
      ...s,
      curves: v,
      grid: Math.min(s.grid, max),
    }));
  }

  return (
    <div style={{ maxWidth: 960, margin: "0 auto", padding: "40px 20px" }}>
      <header style={{ marginBottom: 32 }}>
        <h1 style={{ fontSize: 28, fontWeight: 700, marginBottom: 8 }}>
          pixel-id
        </h1>
        <p style={{ color: "#666", fontSize: 15, lineHeight: 1.5 }}>
          Generate unique snowflake IDs with deterministic pixel avatars.
          Each card shows the same ID rendered by both the <strong>Go backend</strong> and
          the <strong>TypeScript package</strong> — they should be identical.
        </p>
      </header>

      {/* Settings */}
      <div style={settingsBarStyle}>
        <div style={settingStyle}>
          <label style={settingLabelStyle}>Grid</label>
          <input
            type="range"
            min={3}
            max={maxGrid}
            value={settings.grid}
            onChange={(e) => updateGrid(Number(e.target.value))}
            style={{ width: 100, outline: "none" }}
          />
          <span style={settingValueStyle}>{settings.grid}x{settings.grid}</span>
          <span style={settingHintStyle}>max {maxGrid}</span>
        </div>
        <div style={settingStyle}>
          <label style={settingLabelStyle}>Colors</label>
          {[1, 2, 3, 4].map((n) => (
            <button
              key={n}
              onClick={() => updateColors(n)}
              style={{
                ...chipStyle,
                ...(settings.colors === n ? chipActiveStyle : {}),
              }}
            >
              {n}
            </button>
          ))}
        </div>
        <div style={settingStyle}>
          <label style={settingLabelStyle}>Curves</label>
          <button
            onClick={() => updateCurves(!settings.curves)}
            style={{
              ...chipStyle,
              ...(settings.curves ? chipActiveStyle : {}),
            }}
          >
            {settings.curves ? "On" : "Off"}
          </button>
        </div>
      </div>

      <div style={{ display: "flex", gap: 12, marginBottom: 32 }}>
        <button onClick={generate} disabled={loading} style={btnStyle}>
          {loading ? "Generating..." : "Generate ID"}
        </button>
        <button onClick={generateBatch} disabled={loading} style={btnSecondaryStyle}>
          Generate 10
        </button>
        {ids.length > 0 && (
          <button onClick={() => setIds([])} style={btnSecondaryStyle}>
            Clear
          </button>
        )}
      </div>

      {error && (
        <div style={{ background: "#fee", border: "1px solid #fcc", borderRadius: 8, padding: "12px 16px", marginBottom: 24, color: "#c33" }}>
          {error}
        </div>
      )}

      {ids.length === 0 && (
        <div style={{ textAlign: "center", padding: "60px 0", color: "#999" }}>
          Click "Generate ID" to create your first pixel-id
        </div>
      )}

      <div style={{ display: "grid", gap: 16 }}>
        {ids.map((item) => (
          <IdCard key={item.id} data={item} settings={settings} />
        ))}
      </div>
    </div>
  );
}

function IdCard({
  data,
  settings,
}: {
  data: GeneratedId;
  settings: AvatarSettings;
}) {
  const params = new URLSearchParams();
  params.set("grid", String(settings.grid));
  params.set("colors", String(settings.colors));
  if (settings.curves) params.set("curves", "true");
  const qs = params.toString();

  const svgUrl = `/api/avatar/${data.id}.svg?${qs}`;
  const pngUrl = `/api/avatar/${data.id}.png?${qs}`;

  const avatarData = derive(data.id, {
    gridWidth: settings.grid,
    gridHeight: settings.grid,
    numColors: settings.colors,
    curves: settings.curves,
  });

  return (
    <div style={cardStyle}>
      <div style={{ display: "flex", gap: 20, alignItems: "flex-start" }}>
        {/* Go-rendered avatars */}
        <div>
          <div style={implLabelStyle}>Go (server)</div>
          <div style={{ display: "flex", gap: 12 }}>
            <div style={{ textAlign: "center" }}>
              <img
                src={svgUrl}
                alt="Go SVG"
                width={80}
                height={80}
                style={{ borderRadius: 10, display: "block" }}
              />
              <span style={formatLabel}>SVG</span>
            </div>
            <div style={{ textAlign: "center" }}>
              <img
                src={pngUrl}
                alt="Go PNG"
                width={80}
                height={80}
                style={{ borderRadius: 10, display: "block" }}
              />
              <span style={formatLabel}>PNG</span>
            </div>
          </div>
        </div>

        {/* TS-rendered avatar */}
        <div>
          <div style={implLabelStyle}>TypeScript (client)</div>
          <div style={{ textAlign: "center" }}>
            <PixelAvatar
              id={data.id}
              size={80}
              gridWidth={settings.grid}
              gridHeight={settings.grid}
              numColors={settings.colors}
              curves={settings.curves}
              style={{ borderRadius: 10, overflow: "hidden" }}
            />
            <span style={formatLabel}>SVG</span>
          </div>
        </div>

        {/* ID details */}
        <div style={{ flex: 1, minWidth: 0 }}>
          <div style={{ fontFamily: "monospace", fontSize: 14, fontWeight: 600, marginBottom: 8, wordBreak: "break-all" }}>
            {data.id}
          </div>
          <div style={{ display: "grid", gridTemplateColumns: "auto 1fr", gap: "3px 12px", fontSize: 13, color: "#666" }}>
            <span style={{ fontWeight: 500 }}>Timestamp</span>
            <span style={{ fontFamily: "monospace" }}>{data.timestamp}</span>
            <span style={{ fontWeight: 500 }}>Machine</span>
            <span style={{ fontFamily: "monospace" }}>{data.machineId}</span>
            <span style={{ fontWeight: 500 }}>Sequence</span>
            <span style={{ fontFamily: "monospace" }}>{data.sequence}</span>
            <span style={{ fontWeight: 500 }}>Colors</span>
            <span style={{ display: "flex", gap: 4 }}>
              {avatarData.fgColors.map((c, i) => (
                <span
                  key={i}
                  style={{
                    width: 14,
                    height: 14,
                    borderRadius: 3,
                    background: c,
                    display: "inline-block",
                    border: "1px solid #0001",
                  }}
                  title={c}
                />
              ))}
              <span
                style={{
                  width: 14,
                  height: 14,
                  borderRadius: 3,
                  background: avatarData.bgColor,
                  display: "inline-block",
                  border: "1px solid #ddd",
                }}
                title={`BG: ${avatarData.bgColor}`}
              />
            </span>
          </div>
          <div style={{ marginTop: 10, display: "flex", gap: 8 }}>
            <a href={svgUrl} target="_blank" rel="noopener" style={linkStyle}>
              Open SVG
            </a>
            <a href={pngUrl} target="_blank" rel="noopener" style={linkStyle}>
              Open PNG
            </a>
          </div>
        </div>
      </div>
    </div>
  );
}

const settingsBarStyle: React.CSSProperties = {
  display: "flex",
  gap: 24,
  alignItems: "center",
  flexWrap: "wrap",
  background: "#fff",
  border: "1px solid #e8e8e8",
  borderRadius: 10,
  padding: "12px 20px",
  marginBottom: 20,
};

const settingStyle: React.CSSProperties = {
  display: "flex",
  alignItems: "center",
  gap: 8,
};

const settingLabelStyle: React.CSSProperties = {
  fontSize: 12,
  fontWeight: 600,
  textTransform: "uppercase",
  letterSpacing: "0.05em",
  color: "#999",
};

const settingValueStyle: React.CSSProperties = {
  fontFamily: "monospace",
  fontSize: 13,
  fontWeight: 600,
  minWidth: 36,
};

const settingHintStyle: React.CSSProperties = {
  fontSize: 11,
  color: "#bbb",
};

const chipStyle: React.CSSProperties = {
  padding: "4px 10px",
  fontSize: 12,
  fontWeight: 600,
  background: "#f5f5f5",
  color: "#666",
  border: "1px solid #e0e0e0",
  borderRadius: 6,
  cursor: "pointer",
};

const chipActiveStyle: React.CSSProperties = {
  background: "#1a1a1a",
  color: "#fff",
  borderColor: "#1a1a1a",
};

const btnStyle: React.CSSProperties = {
  padding: "10px 20px",
  fontSize: 14,
  fontWeight: 600,
  background: "#1a1a1a",
  color: "#fff",
  border: "none",
  borderRadius: 8,
  cursor: "pointer",
};

const btnSecondaryStyle: React.CSSProperties = {
  ...btnStyle,
  background: "#fff",
  color: "#1a1a1a",
  border: "1px solid #ddd",
};

const cardStyle: React.CSSProperties = {
  background: "#fff",
  border: "1px solid #e8e8e8",
  borderRadius: 12,
  padding: 20,
};

const implLabelStyle: React.CSSProperties = {
  fontSize: 11,
  fontWeight: 600,
  textTransform: "uppercase",
  letterSpacing: "0.05em",
  color: "#999",
  marginBottom: 8,
};

const formatLabel: React.CSSProperties = {
  fontSize: 10,
  color: "#bbb",
  marginTop: 4,
  display: "block",
};

const linkStyle: React.CSSProperties = {
  fontSize: 12,
  color: "#3498db",
  textDecoration: "none",
  padding: "4px 8px",
  border: "1px solid #3498db33",
  borderRadius: 4,
};
