<script setup lang="ts">
import type { Message } from "./types";
defineProps<{ message: Message }>();
</script>
<template>
  <div v-if="message.recalled_at" class="recalled-message">消息已撤回</div>
  <template v-else>
    <span v-if="message.forward_from" class="forward-label">转发的消息</span>
    <blockquote v-if="message.reply" class="reply-quote">
      <strong>{{ message.reply.name }}</strong
      ><span>{{ message.reply.body }}</span>
    </blockquote>
    <a
      v-if="message.attachment"
      :href="'/api/uploads/' + message.attachment.id"
      target="_blank"
      rel="noopener"
      class="message-attachment"
    >
      <img
        v-if="message.attachment.mime.startsWith('image/')"
        :src="'/api/uploads/' + message.attachment.id"
        :alt="message.attachment.name"
        loading="lazy"
      />
      <span v-else
        >📎 {{ message.attachment.name }} ·
        {{ (message.attachment.size / 1048576).toFixed(2) }} MiB</span
      >
    </a>
    <span
      v-if="
        !message.attachment ||
        message.body !== '[文件] ' + message.attachment.name
      "
      >{{ message.body }}</span
    >
  </template>
</template>
