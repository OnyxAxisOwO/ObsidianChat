import { reactive } from "vue";
export const avatarVersions = reactive<Record<string, number>>({});
export function refreshAvatar(user: string) {
  avatarVersions[user] = Date.now();
}
