export function storedPanelWidth(
  key: string,
  fallback: number,
  min: number,
  max: number,
) {
  try {
    const saved = localStorage.getItem(key);
    if (saved === null) return fallback;
    const value = Number(saved);
    return Number.isFinite(value)
      ? Math.max(min, Math.min(max, value))
      : fallback;
  } catch {
    return fallback;
  }
}

export function clampPanelWidth(value: number, min: number, max: number) {
  return Math.round(Math.max(min, Math.min(Math.max(min, max), value)));
}

export function panelWidthFromPointer(
  startWidth: number,
  pointerDelta: number,
  direction: 1 | -1,
  min: number,
  max: number,
) {
  return clampPanelWidth(startWidth + pointerDelta * direction, min, max);
}

export function panelWidthFromKey(
  key: string,
  width: number,
  direction: 1 | -1,
  min: number,
  max: number,
) {
  const changes: Record<string, number> = {
    ArrowLeft: width - 12 * direction,
    ArrowRight: width + 12 * direction,
    Home: min,
    End: max,
  };
  return key in changes
    ? clampPanelWidth(changes[key]!, min, max)
    : undefined;
}

