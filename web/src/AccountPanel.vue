<script setup lang="ts">
import { ref } from "vue";
import { X } from "lucide-vue-next";
import { api, errorText } from "./api";
import type { User } from "./types";
const props = defineProps<{
  me: User;
  notificationsSupported: boolean;
  notificationsEnabled: boolean;
  notificationPermission: NotificationPermission | "unsupported";
  notificationBusy: boolean;
}>();
const emit = defineEmits<{
  close: [];
  updated: [user: User];
  logout: [];
  notifications: [enabled: boolean];
}>();
const name = ref(props.me.name),
  password = ref(""),
  oldPassword = ref(""),
  busy = ref(false),
  error = ref(""),
  note = ref("");
async function save() {
  busy.value = true;
  error.value = "";
  note.value = "";
  try {
    const user = await api<User>("/me", "PATCH", {
      name: name.value,
      password: password.value,
      oldPassword: oldPassword.value,
    });
    password.value = "";
    oldPassword.value = "";
    emit("updated", user);
    note.value = "已保存";
  } catch (e) {
    error.value = errorText(e);
  } finally {
    busy.value = false;
  }
}
</script>
<template>
  <aside class="side-panel panel">
    <div class="section-head">
      <h2>账号设置</h2>
      <button class="icon-btn" aria-label="关闭" @click="emit('close')">
        <X :size="18" />
      </button>
    </div>
    <form class="side-body stack" @submit.prevent="save">
      <p>@{{ me.username }} · {{ me.role === "admin" ? "管理员" : "用户" }}</p>
      <p v-if="error" class="error" role="alert">{{ error }}</p>
      <p v-if="note" role="status">{{ note }}</p>
      <div class="setting-row notification-setting">
        <div>
          <strong>新消息通知</strong>
          <p v-if="!notificationsSupported">此浏览器不支持系统通知</p>
          <p v-else-if="notificationPermission === 'denied'">
            浏览器已阻止通知，请在网站权限中重新允许
          </p>
          <p v-else-if="notificationsEnabled">
            页面在后台时会显示系统通知
          </p>
          <p v-else>开启后，页面在后台时显示系统通知</p>
        </div>
        <button
          type="button"
          class="switch"
          role="switch"
          aria-label="新消息通知"
          :aria-checked="notificationsEnabled"
          :disabled="
            notificationBusy ||
            !notificationsSupported ||
            notificationPermission === 'denied'
          "
          @click="emit('notifications', !notificationsEnabled)"
        >
          <span />
        </button>
      </div>
      <label
        >昵称<input
          v-model="name"
          maxlength="32"
          required
          autocomplete="nickname" /></label
      ><label
        >原密码<input
          v-model="oldPassword"
          type="password"
          autocomplete="current-password"
          :required="!!password" /></label
      ><label
        >新密码<input
          v-model="password"
          type="password"
          minlength="10"
          maxlength="128"
          autocomplete="new-password"
          placeholder="不修改请留空" /></label
      ><button class="primary" :disabled="busy">
        {{ busy ? "保存中…" : "保存" }}</button
      ><button type="button" @click="emit('logout')">退出登录</button>
    </form>
  </aside>
</template>
