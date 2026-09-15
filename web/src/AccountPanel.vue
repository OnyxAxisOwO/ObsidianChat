<script setup lang="ts">
import { ref, onMounted } from "vue";
import { X } from "lucide-vue-next";
import { api, errorText, uploadFile } from "./api";
import { appearance, appearanceRanges, resetAppearance } from "./appearance";
import { refreshAvatar } from "./avatars";
import AvatarContent from "./AvatarContent.vue";
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
const limits = ref({ avatar_mb: 2, image_mb: 10, file_mb: 25 });
onMounted(async () => {
  try {
    limits.value = await api<typeof limits.value>("/upload-limits");
  } catch (e) {
    error.value = errorText(e);
  }
});
async function upload(event: Event, avatar: boolean) {
  const input = event.target as HTMLInputElement,
    file = input.files?.[0];
  if (!file) return;
  busy.value = true;
  error.value = "";
  note.value = "";
  try {
    const limit = avatar ? limits.value.avatar_mb : limits.value.image_mb;
    if (file.size > limit * 1048576) throw new Error(`图片最大 ${limit} MiB`);
    const result = await uploadFile(file, avatar ? "avatar" : "image");
    if (avatar) refreshAvatar(props.me.id);
    else appearance.wallpaper = "/api/uploads/" + result.id;
    note.value = avatar ? "头像已更新" : "壁纸已更新";
  } catch (e) {
    error.value = errorText(e);
  } finally {
    busy.value = false;
    input.value = "";
  }
}
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
      <p class="muted-note">本站管理员可查看聊天记录以处理违规内容。</p>
      <p v-if="error" class="error" role="alert">{{ error }}</p>
      <p v-if="note" role="status">{{ note }}</p>
      <div class="avatar-settings">
        <span class="avatar"
          ><AvatarContent :user="me.id" :name="me.name" /></span
        ><label
          >更换头像（最大 {{ limits.avatar_mb }} MiB）<input
            type="file"
            accept="image/png,image/jpeg,image/gif"
            :disabled="busy"
            @change="upload($event, true)"
        /></label>
      </div>
      <div class="setting-row notification-setting">
        <div>
          <strong>新消息通知</strong>
          <p v-if="!notificationsSupported">此浏览器不支持系统通知</p>
          <p v-else-if="notificationPermission === 'denied'">
            浏览器已阻止通知，请在网站权限中重新允许
          </p>
          <p v-else-if="notificationsEnabled">页面在后台时会显示系统通知</p>
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
      <details class="appearance-settings" open>
        <summary>外观与消息气泡</summary>
        <div class="stack">
          <label
            >背景主题<select v-model="appearance.background">
              <option value="paper">素色</option>
              <option value="forest">森林</option>
              <option value="sunset">日落</option>
              <option value="night">星夜</option>
            </select></label
          >
          <label
            >上传壁纸（最大 {{ limits.image_mb }} MiB）<input
              type="file"
              accept="image/png,image/jpeg,image/gif"
              :disabled="busy"
              @change="upload($event, false)"
          /></label>
          <button
            v-if="appearance.wallpaper"
            type="button"
            @click="appearance.wallpaper = ''"
          >
            移除壁纸
          </button>
          <label v-for="range in appearanceRanges" :key="range.key"
            >{{ range.label }}
            <output>{{ appearance[range.key] }}{{ range.unit }}</output
            ><input
              v-model.number="appearance[range.key]"
              type="range"
              :aria-label="range.label"
              :min="range.min"
              :max="range.max"
          /></label>
          <div class="bubble appearance-preview">消息气泡预览</div>
          <button type="button" @click="resetAppearance">恢复默认外观</button>
        </div>
      </details>
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
