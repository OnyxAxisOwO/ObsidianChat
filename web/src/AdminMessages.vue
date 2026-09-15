<script setup lang="ts">
import { ref, onMounted } from "vue";
import { api, dateTime, errorText } from "./api";
import type { Message } from "./types";
import MessageContent from "./MessageContent.vue";
const props = defineProps<{ user?: string }>();
const user = ref(props.user || ""),
  room = ref(""),
  query = ref(""),
  offset = ref(0),
  messages = ref<Message[]>([]),
  busy = ref(false),
  error = ref("");
async function load() {
  busy.value = true;
  error.value = "";
  try {
    messages.value = await api<Message[]>(
      "/admin/messages?" +
        new URLSearchParams({
          user: user.value,
          room: room.value,
          q: query.value,
          offset: String(offset.value),
        }),
    );
  } catch (e) {
    error.value = errorText(e);
  } finally {
    busy.value = false;
  }
}
function page(delta: number) {
  offset.value = Math.max(0, offset.value + delta);
  void load();
}
onMounted(load);
</script>
<template>
  <p>查询记录会写入管理员操作日志。用户筛选包含该用户所在会话的收发记录。</p>
  <form
    class="record-search"
    @submit.prevent="
      offset = 0;
      load();
    "
  >
    <label
      >用户 ID<input
        v-model.trim="user"
        placeholder="留空查询所有用户" /></label
    ><label
      >会话 ID<input
        v-model.trim="room"
        placeholder="留空查询所有会话" /></label
    ><label
      >消息内容<input
        v-model.trim="query"
        placeholder="关键词"
        maxlength="200" /></label
    ><button class="primary" :disabled="busy">查询</button>
  </form>
  <p v-if="error" class="error" role="alert">{{ error }}</p>
  <div class="admin-message-list">
    <article v-for="m in messages" :key="m.id">
      <header>
        <strong>{{ m.name }}</strong> · {{ dateTime(m.created_at)
        }}<small
          >用户 {{ m.sender }} · 会话 {{ m.room_id }} · #{{ m.id }}</small
        >
      </header>
      <div class="bubble"><MessageContent :message="m" /></div>
    </article>
  </div>
  <p v-if="!busy && !messages.length">暂无记录</p>
  <div class="pagination">
    <span
      >{{ offset + (messages.length ? 1 : 0) }}–{{
        offset + messages.length
      }}</span
    ><button :disabled="busy || offset === 0" @click="page(-50)">上一页</button
    ><button :disabled="busy || messages.length < 50" @click="page(50)">
      下一页
    </button>
  </div>
</template>
