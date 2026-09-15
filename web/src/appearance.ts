import { reactive, watch } from "vue";
const defaults = {
  padding: 8,
  radius: 16,
  font: 14,
  width: 80,
  panelOpacity: 88,
  panelBlur: 14,
  backgroundBlur: 0,
  dim: 0,
  wallpaper: "",
  background: "mint",
  accent: "#8fb49c",
};
export const appearance = reactive({ ...defaults });
try {
  const saved = JSON.parse(localStorage.getItem("oc-appearance") || "{}");
  for (const key of Object.keys(defaults) as (keyof typeof defaults)[]) {
    if (typeof saved[key] === typeof defaults[key])
      Object.assign(appearance, { [key]: saved[key] });
  }
  const legacyBackgrounds: Record<string, string> = {
    paper: "vanilla",
    forest: "mint",
    sunset: "peach",
    night: "lavender",
  };
  if (legacyBackgrounds[appearance.background]) {
    appearance.background = legacyBackgrounds[appearance.background];
    // The old photographic gradients needed a dark overlay. The new solid
    // pastel themes are designed to remain clear without one.
    if (saved.dim === 20) appearance.dim = 0;
  }
} catch {
  /* Defaults remain usable when browser storage is unavailable. */
}
export const appearanceRanges = [
  { key: "padding", label: "气泡留白", min: 2, max: 24, unit: "px" },
  { key: "radius", label: "气泡圆角", min: 0, max: 36, unit: "px" },
  { key: "font", label: "消息字号", min: 12, max: 26, unit: "px" },
  { key: "width", label: "气泡最大宽度", min: 40, max: 95, unit: "%" },
  { key: "panelOpacity", label: "控件不透明度", min: 10, max: 100, unit: "%" },
  { key: "panelBlur", label: "控件模糊", min: 0, max: 40, unit: "px" },
  { key: "backgroundBlur", label: "背景模糊", min: 0, max: 40, unit: "px" },
  { key: "dim", label: "背景压暗", min: 0, max: 90, unit: "%" },
] as const;
for (const r of appearanceRanges)
  appearance[r.key] = Math.max(
    r.min,
    Math.min(
      r.max,
      Number.isFinite(appearance[r.key]) ? appearance[r.key] : defaults[r.key],
    ),
  );
if (!/^#[0-9a-f]{6}$/i.test(appearance.accent))
  appearance.accent = defaults.accent;
export function resetAppearance() {
  Object.assign(appearance, defaults);
}
watch(
  appearance,
  (value) => {
    try {
      localStorage.setItem("oc-appearance", JSON.stringify(value));
    } catch {
      /* Ignore full/blocked storage. */
    }
    const root = document.documentElement.style;
    for (const r of appearanceRanges)
      root.setProperty("--chat-" + r.key, value[r.key] + r.unit);
    root.setProperty("--accent-seed", value.accent);
  },
  { deep: true, immediate: true },
);
