export class APIError extends Error {
  constructor(
    public status: number,
    message: string,
  ) {
    super(message);
  }
}
export interface Challenge {
  site_key: string;
  action: string;
}
let challengeHandler: ((challenge: Challenge) => Promise<string>) | undefined;
export function setChallengeHandler(
  handler?: (challenge: Challenge) => Promise<string>,
) {
  challengeHandler = handler;
}
export function newClientID(): string {
  // getRandomValues also works on an HTTP IP address before a domain is configured.
  const bytes = crypto.getRandomValues(new Uint8Array(16));
  return Array.from(bytes, (byte) => byte.toString(16).padStart(2, "0")).join(
    "",
  );
}
export async function api<T>(
  path: string,
  method = "GET",
  body?: unknown,
  challengeToken?: string,
): Promise<T> {
  const res = await fetch("/api" + path, {
    method,
    credentials: "same-origin",
    headers:
      method === "GET"
        ? {}
        : {
            "Content-Type": "application/json",
            ...(challengeToken ? { "X-Turnstile-Token": challengeToken } : {}),
          },
    body: body === undefined ? undefined : JSON.stringify(body),
    signal: AbortSignal.timeout(15000),
  });
  const data = await res.json();
  if (
    res.status === 428 &&
    data.challenge &&
    challengeHandler &&
    !challengeToken
  ) {
    const token = await challengeHandler(data.challenge);
    return api<T>(path, method, body, token);
  }
  if (!res.ok) throw new APIError(res.status, data.error || "请求失败");
  return data as T;
}
export async function uploadFile(
  file: File,
  purpose: "avatar" | "image" | "file",
) {
  const body = new FormData();
  body.append("file", file);
  const res = await fetch("/api/uploads?purpose=" + purpose, {
    method: "POST",
    credentials: "same-origin",
    body,
    signal: AbortSignal.timeout(180000),
  });
  const data = await res.json();
  if (!res.ok) throw new APIError(res.status, data.error || "上传失败");
  return data as import("./types").Attachment;
}
export function errorText(error: unknown): string {
  return error instanceof Error ? error.message : "请求失败";
}
export const time = (value: number) =>
  new Date(value).toLocaleTimeString("zh-CN", {
    hour: "2-digit",
    minute: "2-digit",
  });
export const dateTime = (value: number) =>
  new Date(value).toLocaleString("zh-CN", { hour12: false });
