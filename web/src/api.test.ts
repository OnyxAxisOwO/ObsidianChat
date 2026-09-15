import { afterEach, expect, it, vi } from "vitest";
import { api, newClientID, setChallengeHandler } from "./api";

afterEach(() => { vi.unstubAllGlobals(); setChallengeHandler(); });

it("generates message IDs when randomUUID is unavailable on HTTP", () => {
  const getRandomValues = crypto.getRandomValues.bind(crypto);
  vi.stubGlobal("crypto", { getRandomValues });
  const first = newClientID();
  const second = newClientID();
  expect(first).toMatch(/^[a-f0-9]{32}$/);
  expect(first).not.toBe(second);
});
it("resumes the original operation only after human verification completes", async () => {
 const challenge = { site_key: "public-test-key", action: "message" };
 const request = vi.fn()
  .mockResolvedValueOnce({ok:false,status:428,json:async()=>({error:"验证",challenge})})
  .mockResolvedValueOnce({ok:true,status:200,json:async()=>({id:42})});
 vi.stubGlobal("fetch",request);
 let finish!: (token:string)=>void;
 const verify=vi.fn(()=>new Promise<string>(resolve=>finish=resolve));
 setChallengeHandler(verify);
 const body={body:"message",client_id:"stable-client-id",reply_to:7};
 const pending=api("/rooms/one/messages","POST",body);
 await vi.waitFor(()=>expect(verify).toHaveBeenCalledWith(challenge));
 expect(request).toHaveBeenCalledTimes(1);
 finish("one-time-proof");expect(await pending).toEqual({id:42});
 const retry=request.mock.calls[1]!;
 expect(retry[1].headers["X-Turnstile-Token"]).toBe("one-time-proof");
 expect(JSON.parse(retry[1].body)).toEqual(body);
});
it("does not submit again after the verification dialog is cancelled",async()=>{
 const request=vi.fn().mockResolvedValue({ok:false,status:428,json:async()=>({challenge:{site_key:"key",action:"friend"}})});
 vi.stubGlobal("fetch",request);setChallengeHandler(async()=>{throw new Error("cancelled")});
 await expect(api("/friends/requests","POST",{user_id:"friend"})).rejects.toThrow("cancelled");
 expect(request).toHaveBeenCalledTimes(1);
});
