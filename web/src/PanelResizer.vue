<script setup lang="ts">
import { onBeforeUnmount, ref } from "vue";
import { GripVertical } from "lucide-vue-next";
import {
  clampPanelWidth,
  panelWidthFromKey,
  panelWidthFromPointer,
} from "./panelSizing";

const props = withDefaults(
  defineProps<{
    label: string;
    min: number;
    max: number;
    defaultValue: number;
    storageKey: string;
    direction?: 1 | -1;
    disabled?: boolean;
  }>(),
  { direction: 1, disabled: false },
);
const model = defineModel<number>({ required: true });
const emit = defineEmits<{ dragging: [value: boolean] }>();
const dragging = ref(false);
let startX = 0;
let startWidth = 0;

function clamp(value: number) {
  return clampPanelWidth(value, props.min, props.max);
}
function persist() {
  try {
    localStorage.setItem(props.storageKey, String(model.value));
  } catch {
    /* Resizing remains available when browser storage is blocked. */
  }
}
function move(event: PointerEvent) {
  if (!dragging.value) return;
  model.value = panelWidthFromPointer(
    startWidth,
    event.clientX - startX,
    props.direction,
    props.min,
    props.max,
  );
}
function finish() {
  if (!dragging.value) return;
  dragging.value = false;
  persist();
  emit("dragging", false);
  window.removeEventListener("pointermove", move);
  window.removeEventListener("pointerup", finish);
  window.removeEventListener("pointercancel", finish);
}
function begin(event: PointerEvent) {
  if (props.disabled || event.button !== 0) return;
  event.preventDefault();
  startX = event.clientX;
  startWidth = model.value;
  dragging.value = true;
  emit("dragging", true);
  window.addEventListener("pointermove", move);
  window.addEventListener("pointerup", finish);
  window.addEventListener("pointercancel", finish);
}
function keydown(event: KeyboardEvent) {
  if (props.disabled) return;
  const value = panelWidthFromKey(
    event.key,
    model.value,
    props.direction,
    props.min,
    props.max,
  );
  if (value === undefined) return;
  event.preventDefault();
  model.value = value;
  persist();
}
function reset() {
  if (props.disabled) return;
  model.value = clamp(props.defaultValue);
  persist();
}
onBeforeUnmount(finish);
</script>

<template>
  <div
    class="panel-resizer"
    :class="{ active: dragging, disabled }"
    role="separator"
    :tabindex="disabled ? -1 : 0"
    :aria-label="label"
    aria-orientation="vertical"
    :aria-disabled="disabled"
    :aria-valuemin="min"
    :aria-valuemax="Math.max(min, max)"
    :aria-valuenow="Math.round(model)"
    :title="label + '；双击恢复默认'"
    @pointerdown="begin"
    @keydown="keydown"
    @dblclick="reset"
  >
    <span class="resize-handle" aria-hidden="true"
      ><GripVertical :size="16"
    /></span>
  </div>
</template>
