<script setup lang="ts">
import { ref, onMounted } from "vue";
import { api, errorText } from "./api";
import type { Policy } from "./types";
const policy = ref<Policy>(),
  error = ref(""),
  note = ref(""),
  busy = ref(false);
const toggles = [
  { key: "register", label: "注册账号" },
  { key: "login", label: "登录账号" },
  { key: "friend", label: "添加好友" },
  { key: "group", label: "创建群聊" },
] as const;
const limits = [
  { key: "avatar_mb", label: "头像最大大小（MiB）", min: 1, max: 10 },
  { key: "image_mb", label: "图片最大大小（MiB）", min: 1, max: 30 },
  { key: "file_mb", label: "文件最大大小（MiB）", min: 1, max: 100 },
  { key: "friends_minute", label: "每人每分钟最多好友申请", min: 1, max: 200 },
  { key: "messages_minute", label: "每人每分钟最多消息", min: 1, max: 1000 },
  {
    key: "challenge_minute",
    label: "每分钟发送多少条后需要验证（0 关闭）",
    min: 0,
    max: 1000,
  },
  {
    key: "challenge_hour",
    label: "每小时发送多少条后需要验证（0 关闭）",
    min: 0,
    max: 60000,
  },
] as const;
async function load() {
  try {
    policy.value = await api<Policy>("/admin/policy");
  } catch (e) {
    error.value = errorText(e);
  }
}
async function save() {
  if (!policy.value) return;
  busy.value = true;
  error.value = "";
  note.value = "";
  try {
    policy.value = await api<Policy>("/admin/policy", "PATCH", policy.value);
    note.value = "已保存，立即生效";
  } catch (e) {
    error.value = errorText(e);
  } finally {
    busy.value = false;
  }
}
onMounted(load);
</script>
<template>
  <p v-if="error" class="error" role="alert">{{ error }}</p>
  <p v-if="note" role="status">{{ note }}</p>
  <form v-if="policy" class="policy-form stack" @submit.prevent="save">
    <h3>Cloudflare Turnstile</h3>
    <p>
      在 Cloudflare 创建本站的 Turnstile
      组件，填写密钥并选择需要验证的操作。服务端密钥保存后不会回显。
    </p>
    <label
      >站点密钥（Site key）<input
        v-model="policy.site_key"
        autocomplete="off"
        maxlength="256"
    /></label>
    <label
      >服务端密钥（Secret key）<input
        v-model="policy.secret_key"
        type="password"
        autocomplete="new-password"
        maxlength="256"
        :placeholder="
          policy.secret_configured ? '已配置；留空保留原密钥' : '尚未配置'
        "
    /></label>
    <div class="policy-checks">
      <label v-for="toggle in toggles" :key="toggle.key"
        ><input v-model="policy[toggle.key]" type="checkbox" />{{
          toggle.label
        }}</label
      >
    </div>
    <h3>上传与反垃圾</h3>
    <p>
      达到消息上限会暂停发送。达到任一验证阈值后，每次发送需重新验证，直到最近一分钟／一小时内的消息数下降。创建群聊每人每分钟最多
      10 次。
    </p>
    <label v-for="limit in limits" :key="limit.key"
      >{{ limit.label
      }}<input
        v-model.number="policy[limit.key]"
        type="number"
        required
        :min="limit.min"
        :max="limit.max"
        step="1"
    /></label>
    <button class="primary" :disabled="busy">
      {{ busy ? "保存中…" : "保存设置" }}
    </button>
  </form>
</template>
