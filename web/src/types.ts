export interface User {
  id: string;
  username: string;
  name: string;
  role: "admin" | "user";
  disabled: boolean;
  online: boolean;
  created_at: number;
}
export interface Room {
  id: string;
  kind: "direct" | "group";
  name: string;
  owner: string;
  archived: boolean;
  member_count: number;
  last_message: string;
  last_at: number;
  last_id: number;
  unread: number;
  peer_id: string;
  online: boolean;
}
export interface Message {
  id: number;
  room_id: string;
  sender: string;
  name: string;
  body: string;
  client_id: string;
  created_at: number;
  reply_to?: number;
  reply?: { id: number; name: string; body: string };
  forward_from?: number;
  upload_id?: string;
  attachment?: Attachment;
  recalled_at?: number;
}
export interface Attachment {
  id: string;
  name: string;
  mime: string;
  size: number;
}
export interface Policy {
  site_key: string;
  secret_key?: string;
  secret_configured: boolean;
  register: boolean;
  login: boolean;
  friend: boolean;
  group: boolean;
  avatar_mb: number;
  image_mb: number;
  file_mb: number;
  friends_minute: number;
  messages_minute: number;
  challenge_minute: number;
  challenge_hour: number;
}
export interface FriendRequest {
  id: string;
  sender: string;
  recipient: string;
  name: string;
  username: string;
  created_at: number;
}
export interface FriendData {
  friends: User[];
  requests: FriendRequest[];
}
export interface Stats {
  users: number;
  groups: number;
  messages: number;
  connections: number;
  online: number;
  dropped_connections: number;
  heap_bytes: number;
  goroutines: number;
  uptime_seconds: number;
  registration: boolean;
}
