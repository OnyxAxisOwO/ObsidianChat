<script setup lang="ts">
defineProps<{ label: string; min: number; max: number; unit: string }>();
const model = defineModel<number>({ required: true });
</script>

<template>
  <label class="settings-range">
    <span class="range-heading">
      <span>{{ label }}</span>
      <output
        >{{ model }}<small>{{ unit }}</small></output
      >
    </span>
    <input
      v-model.number="model"
      type="range"
      :aria-label="label"
      :min="min"
      :max="max"
      :style="{ '--range-progress': ((model - min) / (max - min)) * 100 + '%' }"
    />
  </label>
</template>
