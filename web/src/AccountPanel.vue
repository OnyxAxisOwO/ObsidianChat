<script setup lang="ts">
import { ref, onMounted } from "vue";
import {
  X,
  Camera,
  Check,
  Palette,
  MessageCircle,
  UserRound,
  ImagePlus,
  ChevronRight,
  RotateCcw,
  Bell,
  ShieldCheck,
  LogOut,
  LockKeyhole,
  SlidersHorizontal,
  Layers,
  LoaderCircle,
} from "lucide-vue-next";
import { api, errorText, uploadFile } from "./api";
import { appearance, appearanceRanges, resetAppearance } from "./appearance";
import { refreshAvatar } from "./avatars";
import AvatarContent from "./AvatarContent.vue";
import SettingsRange from "./SettingsRange.vue";
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
const section = ref("appearance");
const avatarInput = ref<HTMLInputElement>();
const wallpaperInput = ref<HTMLInputElement>();
const sections = [
  { key: "appearance", label: "外观", icon: Palette },
  { key: "bubbles", label: "气泡", icon: MessageCircle },
  { key: "account", label: "账号", icon: UserRound },
];
const backgrounds = [
  { key: "paper", label: "原色", caption: "简单一点，也很好。" },
  { key: "forest", label: "青野", caption: "把一片自然，带进聊天。" },
  { key: "sunset", label: "暮光", caption: "留住日落的温柔。" },
  { key: "night", label: "星夜", caption: "让灵感，在夜色中相遇。" },
];
const bubbleRanges = appearanceRanges.slice(0, 4);
const surfaceRanges = appearanceRanges.slice(4, 6);
const backgroundRanges = appearanceRanges.slice(6);
function selectBackground(key: string) {
  appearance.background = key;
  appearance.wallpaper = "";
}
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
  <aside class="side-panel panel account-settings" aria-label="个人设置">
    <header class="settings-header">
      <div>
        <span class="settings-eyebrow">MAKE IT YOURS</span>
        <h2>个人设置<span>.</span></h2>
      </div>
      <button
        type="button"
        class="settings-close icon-btn"
        aria-label="关闭设置"
        @click="emit('close')"
      >
        <X :size="19" />
      </button>
    </header>

    <div class="settings-profile">
      <button
        class="profile-avatar"
        type="button"
        aria-label="更换头像"
        :disabled="busy"
        @click="avatarInput?.click()"
      >
        <AvatarContent :user="me.id" :name="me.name" />
        <span class="avatar-camera"><Camera :size="12" /></span>
      </button>
      <div class="profile-copy">
        <strong>{{ me.name }}</strong
        ><span>@{{ me.username }}</span>
      </div>
      <span class="profile-role">{{
        me.role === "admin" ? "管理员" : "成员"
      }}</span>
      <input
        ref="avatarInput"
        hidden
        type="file"
        accept="image/png,image/jpeg,image/gif"
        :disabled="busy"
        @change="upload($event, true)"
      />
    </div>

    <nav class="settings-tabs" aria-label="设置分类">
      <button
        v-for="item in sections"
        :key="item.key"
        type="button"
        :class="{ selected: section === item.key }"
        :aria-pressed="section === item.key"
        @click="section = item.key"
      >
        <component :is="item.icon" :size="15" /><span>{{ item.label }}</span>
      </button>
    </nav>

    <div class="settings-scroll">
      <p v-if="error" class="settings-feedback error" role="alert">
        {{ error }}
      </p>
      <p v-if="note" class="settings-feedback" role="status">
        <Check :size="14" />{{ note }}
      </p>
      <div v-if="section !== 'account'" class="settings-content">
        <div
          class="settings-preview"
          :class="'background-' + appearance.background"
        >
          <div
            class="preview-wallpaper"
            :style="
              appearance.wallpaper
                ? { backgroundImage: 'url(' + appearance.wallpaper + ')' }
                : undefined
            "
          />
          <div class="preview-shade" />
          <div class="preview-inner">
            <span class="preview-label"><span />实时预览</span>
            <div class="preview-message incoming">嗨，今天也要开心。</div>
            <div class="preview-message outgoing">
              从喜欢的样子开始 <span aria-hidden="true">✦</span>
            </div>
          </div>
        </div>

        <template v-if="section === 'appearance'">
          <section class="settings-section">
            <div class="settings-section-title">
              <h3>你的聊天背景</h3>
              <span>一点色彩，一点心情</span>
            </div>
            <div class="theme-options" aria-label="背景主题">
              <button
                v-for="background in backgrounds"
                :key="background.key"
                type="button"
                class="theme-option"
                :class="{
                  selected:
                    appearance.background === background.key &&
                    !appearance.wallpaper,
                }"
                :aria-label="'背景：' + background.label"
                :aria-pressed="
                  appearance.background === background.key &&
                  !appearance.wallpaper
                "
                @click="selectBackground(background.key)"
              >
                <span
                  class="theme-swatch"
                  :class="'background-' + background.key"
                  ><span
                    v-if="
                      appearance.background === background.key &&
                      !appearance.wallpaper
                    "
                    class="theme-check"
                    ><Check :size="12" :stroke-width="3" /></span
                ></span>
                <span>{{ background.label }}</span>
              </button>
            </div>
            <p class="theme-caption">
              {{
                appearance.wallpaper
                  ? "用一张喜欢的照片，装点这里。"
                  : backgrounds.find((b) => b.key === appearance.background)
                      ?.caption
              }}
            </p>
            <input
              ref="wallpaperInput"
              hidden
              type="file"
              accept="image/png,image/jpeg,image/gif"
              :disabled="busy"
              @change="upload($event, false)"
            />
            <button
              type="button"
              class="wallpaper-upload"
              :disabled="busy"
              @click="wallpaperInput?.click()"
            >
              <span
                class="upload-art"
                :style="
                  appearance.wallpaper
                    ? { backgroundImage: 'url(' + appearance.wallpaper + ')' }
                    : undefined
                "
                ><ImagePlus v-if="!appearance.wallpaper" :size="21"
              /></span>
              <span class="upload-copy"
                ><strong>{{
                  appearance.wallpaper ? "更换自定义壁纸" : "上传自己的壁纸"
                }}</strong
                ><small
                  >JPG、PNG、GIF · 最大 {{ limits.image_mb }} MiB</small
                ></span
              >
              <LoaderCircle
                v-if="busy"
                class="settings-spinner"
                :size="16"
              /><ChevronRight v-else :size="16" />
            </button>
            <button
              v-if="appearance.wallpaper"
              class="settings-text-button"
              type="button"
              @click="appearance.wallpaper = ''"
            >
              移除壁纸，使用主题背景
            </button>
          </section>

          <section class="settings-control-card">
            <div class="control-card-title">
              <SlidersHorizontal :size="15" />
              <h3>背景氛围</h3>
            </div>
            <SettingsRange
              v-for="range in backgroundRanges"
              :key="range.key"
              v-model="appearance[range.key]"
              :label="range.label"
              :min="range.min"
              :max="range.max"
              :unit="range.unit"
            />
          </section>
          <section class="settings-control-card">
            <div class="control-card-title">
              <Layers :size="15" />
              <h3>面板质感</h3>
            </div>
            <SettingsRange
              v-for="range in surfaceRanges"
              :key="range.key"
              v-model="appearance[range.key]"
              :label="range.label"
              :min="range.min"
              :max="range.max"
              :unit="range.unit"
            />
          </section>
        </template>

        <section v-else class="settings-control-card bubble-controls">
          <div class="control-card-title">
            <MessageCircle :size="15" />
            <h3>让对话更合心意</h3>
          </div>
          <p class="settings-hint">拖动滑块，上方预览会一起变化。</p>
          <SettingsRange
            v-for="range in bubbleRanges"
            :key="range.key"
            v-model="appearance[range.key]"
            :label="range.label"
            :min="range.min"
            :max="range.max"
            :unit="range.unit"
          />
        </section>

        <div class="settings-bottom">
          <span><Check :size="12" />外观自动保存</span
          ><button type="button" @click="resetAppearance">
            <RotateCcw :size="13" />恢复默认
          </button>
        </div>
      </div>

      <form v-else class="settings-content account-form" @submit.prevent="save">
        <section class="settings-control-card settings-notifications">
          <div class="notification-heading">
            <span class="settings-icon-tile"><Bell :size="18" /></span>
            <div>
              <h3>新消息通知</h3>
              <span>{{
                notificationsEnabled ? "已开启" : "不错过每一次对话"
              }}</span>
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
          <p v-if="!notificationsSupported" class="settings-hint">
            此浏览器不支持系统通知
          </p>
          <p
            v-else-if="notificationPermission === 'denied'"
            class="settings-hint"
          >
            浏览器已阻止通知，请在网站权限中重新允许。
          </p>
          <p v-else class="settings-hint">
            开启后，页面在后台时也能收到系统提醒。
          </p>
        </section>

        <section class="settings-control-card">
          <div class="control-card-title">
            <UserRound :size="15" />
            <h3>个人资料</h3>
          </div>
          <label class="settings-field"
            >显示昵称<input
              v-model="name"
              maxlength="32"
              required
              autocomplete="nickname"
              placeholder="让朋友认出你"
          /></label>
          <button
            type="button"
            class="settings-secondary-button"
            :disabled="busy"
            @click="avatarInput?.click()"
          >
            <Camera :size="15" />更换头像<span
              >最大 {{ limits.avatar_mb }} MiB</span
            >
          </button>
        </section>

        <section class="settings-control-card">
          <div class="control-card-title">
            <LockKeyhole :size="15" />
            <h3>登录与安全</h3>
          </div>
          <label class="settings-field"
            >当前密码<input
              v-model="oldPassword"
              type="password"
              autocomplete="current-password"
              :required="!!password"
              placeholder="输入当前密码"
          /></label>
          <label class="settings-field"
            >新密码<input
              v-model="password"
              type="password"
              minlength="10"
              maxlength="128"
              autocomplete="new-password"
              placeholder="至少 10 位，不修改请留空"
          /></label>
        </section>
        <button class="settings-save primary" :disabled="busy">
          <LoaderCircle v-if="busy" class="settings-spinner" :size="16" /><Check
            v-else
            :size="16"
          />{{ busy ? "正在保存…" : "保存修改" }}
        </button>
        <p class="settings-privacy">
          <ShieldCheck :size="14" /><span
            >为维护社区安全，管理员可查看聊天记录以处理违规内容。</span
          >
        </p>
        <button type="button" class="settings-logout" @click="emit('logout')">
          <LogOut :size="15" />退出登录
        </button>
      </form>
    </div>
  </aside>
</template>
