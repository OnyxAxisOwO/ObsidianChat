<script setup lang="ts">
import { ref, nextTick, onMounted, onBeforeUnmount } from "vue";
import { setChallengeHandler, type Challenge } from "./api";
interface Turnstile {
  render(element: HTMLElement, options: Record<string, unknown>): string;
  remove(id: string): void;
  reset(id: string): void;
}
declare global {
  interface Window {
    turnstile?: Turnstile;
  }
}
const dialog = ref<HTMLDialogElement>(),
  target = ref<HTMLElement>(),
  error = ref("");
let widget: string | undefined,
  resolve: ((token: string) => void) | undefined,
  reject: ((error: Error) => void) | undefined,
  version = 0;
let scriptLoading: Promise<void> | undefined;
function loadScript() {
  if (window.turnstile) return Promise.resolve();
  if (!scriptLoading)
    scriptLoading = new Promise<void>((done, fail) => {
      const script = document.createElement("script");
      script.src =
        "https://challenges.cloudflare.com/turnstile/v0/api.js?render=explicit";
      script.async = true;
      const timeout = window.setTimeout(() => {
        script.remove();
        fail(new Error("验证码加载超时，请重试"));
      }, 15000);
      script.onload = () => {
        clearTimeout(timeout);
        done();
      };
      script.onerror = () => {
        clearTimeout(timeout);
        script.remove();
        fail(new Error("验证码加载失败，请检查网络"));
      };
      document.head.append(script);
    }).catch((e) => {
      scriptLoading = undefined;
      throw e;
    });
  return scriptLoading;
}
function cleanup() {
  version++;
  if (widget !== undefined) window.turnstile?.remove(widget);
  widget = undefined;
  resolve = undefined;
  reject = undefined;
  dialog.value?.close();
}
function cancel() {
  reject?.(new Error("已取消人机验证"));
  cleanup();
}
async function render(challenge: Challenge) {
  const epoch = ++version;
  error.value = "";
  await nextTick();
  dialog.value?.showModal();
  try {
    await loadScript();
    if (epoch !== version || !target.value || !window.turnstile) return;
    widget = window.turnstile.render(target.value, {
      sitekey: challenge.site_key,
      action: challenge.action,
      theme: "auto",
      callback: (token: string) => {
        resolve?.(token);
        cleanup();
      },
      "error-callback": () => {
        error.value = "验证失败，请点击重试";
      },
      "expired-callback": () => {
        if (widget !== undefined) window.turnstile?.reset(widget);
      },
    });
  } catch (e) {
    if (epoch === version)
      error.value = e instanceof Error ? e.message : "验证加载失败";
  }
}
let active: Challenge | undefined;
function retry() {
  if (!active) return;
  if (widget !== undefined) window.turnstile?.remove(widget);
  widget = undefined;
  void render(active);
}
onMounted(() =>
  setChallengeHandler((challenge) => {
    if (resolve) return Promise.reject(new Error("请先完成当前验证"));
    active = challenge;
    return new Promise<string>((done, fail) => {
      resolve = done;
      reject = fail;
      void render(challenge);
    });
  }),
);
onBeforeUnmount(() => {
  cancel();
  setChallengeHandler();
});
</script>
<template>
  <dialog
    ref="dialog"
    class="app-dialog"
    aria-labelledby="challenge-title"
    @cancel.prevent="cancel"
  >
    <h2 id="challenge-title">请完成人机验证</h2>
    <p>验证通过后将继续刚才的操作。</p>
    <div ref="target" class="challenge-widget" />
    <p v-if="error" class="error" role="alert">{{ error }}</p>
    <div class="dialog-actions">
      <button @click="cancel">取消</button
      ><button v-if="error" class="primary" @click="retry">重试</button>
    </div>
  </dialog>
</template>
