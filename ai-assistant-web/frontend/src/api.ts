export type Role = "user" | "assistant";

export type AiConfig = {
  id: number;
  name: string;
  baseUrl: string;
  modelName: string;
};

export type PromptTemplate = {
  ID?: number;
  id?: number;
  name?: string;
  content?: string;
  type?: string;
};

export type SessionMessage = {
  role: Role;
  content: string;
  reasoning?: string;
  time?: string;
};

export type VipStatus = {
  ok: boolean;
  vipLevel: number;
  active: boolean;
  role?: string;
  message?: string;
};

// 从 localStorage 获取 JWT token
function getAuthToken(): string | null {
  return localStorage.getItem('token');
}

// 创建带有认证头的请求选项
function createAuthHeaders(headers: HeadersInit = {}): HeadersInit {
  const token = getAuthToken();
  return {
    ...headers,
    ...(token ? { 'Authorization': `Bearer ${token}` } : {}),
  };
}

/** 获取当前用户的VIP状态 */
export async function getVipStatus(): Promise<VipStatus> {
  const res = await fetch("/api/vip-status", {
    headers: createAuthHeaders(),
  });
  if (!res.ok) throw new Error(await res.text());
  return await res.json();
}

export async function getAiConfigs(): Promise<AiConfig[]> {
  const res = await fetch("/api/ai-configs", {
    headers: createAuthHeaders(),
  });
  if (!res.ok) throw new Error(await res.text());
  return await res.json();
}

export async function getPrompts(): Promise<PromptTemplate[]> {
  const res = await fetch("/api/prompts?name=&type=", {
    headers: createAuthHeaders(),
  });
  if (!res.ok) throw new Error(await res.text());
  return await res.json();
}

export async function getSession(): Promise<SessionMessage[]> {
  const res = await fetch("/api/session", {
    headers: createAuthHeaders(),
  });
  if (!res.ok) throw new Error(await res.text());
  return await res.json();
}

export async function saveSession(messages: SessionMessage[]): Promise<void> {
  const res = await fetch("/api/session", {
    method: "POST",
    headers: createAuthHeaders({ "Content-Type": "application/json" }),
    body: JSON.stringify({ messages }),
  });
  if (!res.ok) throw new Error(await res.text());
}

export async function shareText(text: string, title = "AI助手"): Promise<string> {
  const res = await fetch("/api/share", {
    method: "POST",
    headers: createAuthHeaders({ "Content-Type": "application/json" }),
    body: JSON.stringify({ text, title }),
  });
  const data = await res.json();
  if (!res.ok) throw new Error(data?.error || "分享失败");
  return data?.message || "已分享";
}

