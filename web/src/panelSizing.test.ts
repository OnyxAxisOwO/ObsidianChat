import { describe, expect, it } from "vitest";
import {
  clampPanelWidth,
  panelWidthFromKey,
  panelWidthFromPointer,
  storedPanelWidth,
} from "./panelSizing";

describe("panel sizing", () => {
  it("uses the fallback when browser storage is unavailable", () => {
    expect(storedPanelWidth("missing", 300, 220, 480)).toBe(300);
  });

  it("clamps and rounds widths", () => {
    expect(clampPanelWidth(219, 220, 480)).toBe(220);
    expect(clampPanelWidth(333.6, 220, 480)).toBe(334);
    expect(clampPanelWidth(600, 220, 480)).toBe(480);
  });

  it("resizes left and right panels in the expected direction", () => {
    expect(panelWidthFromPointer(300, 70, 1, 220, 480)).toBe(370);
    expect(panelWidthFromPointer(360, -70, -1, 280, 560)).toBe(430);
    expect(panelWidthFromPointer(360, 120, -1, 280, 560)).toBe(280);
  });

  it("maps keyboard movement to the visual divider direction", () => {
    expect(panelWidthFromKey("ArrowRight", 300, 1, 220, 480)).toBe(312);
    expect(panelWidthFromKey("ArrowLeft", 360, -1, 280, 560)).toBe(372);
    expect(panelWidthFromKey("Home", 360, -1, 280, 560)).toBe(280);
    expect(panelWidthFromKey("Escape", 360, -1, 280, 560)).toBeUndefined();
  });
});

