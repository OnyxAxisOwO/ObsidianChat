<script setup lang="ts">
import { computed, ref, watch } from "vue";
import { avatarVersions } from "./avatars";
const props = defineProps<{ user?: string; name: string }>();
const failed = ref(false);
const src = computed(() =>
  props.user
    ? `/api/avatars/${encodeURIComponent(props.user)}?v=${avatarVersions[props.user] || 0}`
    : "",
);
watch(src, () => (failed.value = false));
</script>
<template>
  <img
    v-if="src && !failed"
    :src="src"
    alt=""
    class="avatar-image"
    loading="lazy"
    @error="failed = true"
  /><template v-else>{{ name.slice(0, 1) }}</template>
</template>
