// API Client for Sub2API Backend

/** Strip trailing slashes and optional `/auth` suffix so paths like `/auth/login` resolve once. */
function normalizeApiBase(raw: string): string {
  let s = raw.trim().replace(/\/+$/, "");
  if (s.toLowerCase().endsWith("/auth")) {
    s = s.slice(0, -"/auth".length).replace(/\/+$/, "");
  }
  return s;
}

const API_BASE = normalizeApiBase(
  process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080"
);

function httpMessageFromBody(text: string, status: number): string {
  const trimmed = text.trim();
  if (!trimmed) return `Request failed (${status})`;
  try {
    const err = JSON.parse(trimmed) as {
      error?: { message?: string; type?: string } | string;
      message?: string;
    };
    if (typeof err.error === "string") return err.error;
    if (err.error && typeof err.error === "object" && err.error.message) {
      return err.error.message;
    }
    if (err.message) return err.message;
  } catch {
    /* Gin 404 等为纯文本 "404 page not found"，非 JSON */
  }
  return trimmed.length > 300 ? `${trimmed.slice(0, 300)}…` : trimmed;
}

function parseJsonBody<T>(text: string, what: string): T {
  const trimmed = text.trim();
  if (!trimmed) throw new Error(`${what}: empty response`);
  try {
    return JSON.parse(trimmed) as T;
  } catch {
    throw new Error(
      `${what}: ${trimmed.slice(0, 120)}${trimmed.length > 120 ? "…" : ""}`
    );
  }
}

// ==================== Auth Types ====================

export interface User {
  id: string;
  email: string;
  name: string;
  status: string;
  created_at: string;
}

export interface UserSubscriptionView {
  plan_id: string;
  monthly_price_usd: number;
  monthly_spend_cap_usd: number;
  spent_this_period: number;
  remaining_cap_usd: number;
  allowed_models: string[];
  period_end: string;
  active: boolean;
}

export interface SubscriptionPlan {
  id: string;
  monthly_price_usd: number;
  monthly_spend_cap_usd: number;
  included_balance_usd: number;
  allowed_models: string[];
}

export interface UserProfile {
  id: string;
  email: string;
  name: string;
  status: string;
  balance: number;
  spendable_balance?: number;
  has_paid: boolean;
  can_create_key: boolean;
  currency: string;
  subscription?: UserSubscriptionView | null;
}

export interface AuthResponse {
  token?: string;
  user?: User;
  api_key?: string; // 已不再在注册时返回；创建 Key 接口返回
}

export interface APIKey {
  id: string;
  user_id: string;
  name: string;
  key_hash: string;
  key_prefix: string;
  balance: number;
  status: string;
  rate_limit: number;
  spend_limit?: number | null;
  spent_total?: number;
  ip_whitelist?: string[];
  created_at: string;
}

// ==================== Usage Types ====================

export interface UsageResponse {
  balance: number;
  total_used: number;
  request_count: number;
  last_request_at?: string;
}

export interface Transaction {
  id: string;
  user_id?: string;
  key_id?: string;
  type: string;
  amount: number;
  balance_before: number;
  balance_after: number;
  model?: string;
  input_tokens?: number;
  output_tokens?: number;
  request_id?: string;
  created_at: string;
}

export interface TransactionsResponse {
  transactions: Transaction[];
  total: number;
  limit: number;
  offset: number;
}

export interface Model {
  id: string;
  object: string;
  owned_by: string;
}

export interface ModelsResponse {
  object: string;
  data: Model[];
}

export interface DailyUsagePoint {
  date: string;
  total_consumed: number;
  request_count: number;
}

export interface UsageDailyResponse {
  key_id: string;
  days: number;
  points: DailyUsagePoint[];
}

export interface RequestLogEntry {
  id: string;
  key_id: string;
  request_id: string;
  model: string;
  stream: boolean;
  outcome: string;
  latency_ms: number;
  created_at: string;
}

export interface RequestLogsResponse {
  logs: RequestLogEntry[];
  total: number;
  limit: number;
  offset: number;
}

export interface APIError {
  error: {
    message: string;
    type: string;
    code?: string;
  };
}

class APIClient {
  private apiKey: string | null = null;
  private jwtToken: string | null = null;

  setApiKey(key: string) {
    this.apiKey = key;
  }

  getApiKey(): string | null {
    return this.apiKey;
  }

  clearApiKey() {
    this.apiKey = null;
  }

  setToken(token: string) {
    this.jwtToken = token;
  }

  getToken(): string | null {
    return this.jwtToken;
  }

  clearToken() {
    this.jwtToken = null;
  }

  clearAll() {
    this.apiKey = null;
    this.jwtToken = null;
  }

  // Request with API Key auth (for OpenAI compatible endpoints)
  private async request<T>(
    endpoint: string,
    options: RequestInit = {}
  ): Promise<T> {
    if (!this.apiKey) {
      throw new Error("API Key not set");
    }

    const response = await fetch(`${API_BASE}${endpoint}`, {
      ...options,
      headers: {
        Authorization: `Bearer ${this.apiKey}`,
        "Content-Type": "application/json",
        ...options.headers,
      },
    });

    const text = await response.text();
    if (!response.ok) {
      throw new Error(httpMessageFromBody(text, response.status));
    }

    return parseJsonBody<T>(text, "Invalid JSON");
  }

  // Request with JWT auth (for dashboard endpoints)
  private async authRequest<T>(
    endpoint: string,
    options: RequestInit = {}
  ): Promise<T> {
    const headers: Record<string, string> = {
      "Content-Type": "application/json",
      ...((options.headers as Record<string, string>) || {}),
    };

    if (this.jwtToken) {
      headers.Authorization = `Bearer ${this.jwtToken}`;
    }

    const response = await fetch(`${API_BASE}${endpoint}`, {
      ...options,
      headers,
    });

    const text = await response.text();
    if (!response.ok) {
      throw new Error(httpMessageFromBody(text, response.status));
    }

    return parseJsonBody<T>(text, "Invalid JSON");
  }

  // ==================== Auth Endpoints ====================

  // GET /auth/config — public; no JWT
  async getAuthConfig(): Promise<{
    email_verify_enabled: boolean;
    invite_required?: boolean;
    terms_version?: string;
    terms_required?: boolean;
    chat_enabled_models?: string[];
    currency?: string;
    subscriptions_enabled?: boolean;
    subscription_period_days?: number;
    subscription_plans?: SubscriptionPlan[];
  }> {
    const response = await fetch(`${API_BASE}/auth/config`, {
      method: "GET",
      headers: { Accept: "application/json" },
    });
    const text = await response.text();
    if (!response.ok) {
      throw new Error(httpMessageFromBody(text, response.status));
    }
    return parseJsonBody(text, "Invalid auth config JSON");
  }

  async register(
    email: string,
    password: string,
    options?: {
      name?: string;
      verificationCode?: string;
      termsAccepted: boolean;
      termsVersion: string;
    }
  ): Promise<AuthResponse> {
    return this.authRequest<AuthResponse>("/auth/register", {
      method: "POST",
      body: JSON.stringify({
        email,
        password,
        name: options?.name,
        verification_code: options?.verificationCode || undefined,
        terms_accepted: options?.termsAccepted ?? false,
        terms_version: options?.termsVersion ?? "",
      }),
    });
  }

  async login(email: string, password: string): Promise<AuthResponse> {
    return this.authRequest<AuthResponse>("/auth/login", {
      method: "POST",
      body: JSON.stringify({ email, password }),
    });
  }

  async sendRegisterCode(email: string): Promise<{ message: string }> {
    return this.authRequest<{ message: string }>("/auth/send-register-code", {
      method: "POST",
      body: JSON.stringify({ email }),
    });
  }

  async sendResetPasswordCode(email: string): Promise<{ message: string }> {
    return this.authRequest<{ message: string }>("/auth/send-reset-password-code", {
      method: "POST",
      body: JSON.stringify({ email }),
    });
  }

  async resetPassword(
    email: string,
    verificationCode: string,
    newPassword: string
  ): Promise<{ message: string }> {
    return this.authRequest<{ message: string }>("/auth/reset-password", {
      method: "POST",
      body: JSON.stringify({
        email,
        verification_code: verificationCode,
        new_password: newPassword,
      }),
    });
  }

  async getMe(): Promise<UserProfile> {
    return this.authRequest<UserProfile>("/dashboard/me");
  }

  async updateProfile(name: string): Promise<{ message: string }> {
    return this.authRequest("/dashboard/me", {
      method: "PATCH",
      body: JSON.stringify({ name }),
    });
  }

  async changePassword(currentPassword: string, newPassword: string): Promise<{ message: string }> {
    return this.authRequest("/dashboard/change-password", {
      method: "POST",
      body: JSON.stringify({
        current_password: currentPassword,
        new_password: newPassword,
      }),
    });
  }

  async createAccountCheckout(amount: number): Promise<{ checkout_url: string; session_id: string }> {
    return this.authRequest("/dashboard/payment/checkout", {
      method: "POST",
      body: JSON.stringify({ amount }),
    });
  }

  async getAccountTransactions(limit = 20, offset = 0): Promise<TransactionsResponse> {
    return this.authRequest<TransactionsResponse>(
      `/dashboard/account/transactions?limit=${limit}&offset=${offset}`
    );
  }

  async getMyKeys(): Promise<{ keys: APIKey[] }> {
    const res = await this.authRequest<{ keys: APIKey[] | null }>("/dashboard/keys");
    return { keys: Array.isArray(res.keys) ? res.keys : [] };
  }

  // 创建 Key（需密码二次验证）
  async createKey(
    password: string,
    name?: string,
    rateLimit?: number,
    spendLimit?: number
  ): Promise<{
    key: string;
    key_id: string;
    key_prefix: string;
    name: string;
    balance: number;
    rate_limit: number;
    warning: string;
  }> {
    return this.authRequest("/dashboard/keys", {
      method: "POST",
      body: JSON.stringify({
        password,
        name,
        rate_limit: rateLimit,
        spend_limit: spendLimit,
      }),
    });
  }

  // 更新 Key 设置
  async updateKeySettings(
    keyId: string,
    ipWhitelist?: string[],
    rateLimit?: number,
    spendLimit?: number | null
  ): Promise<APIKey> {
    return this.authRequest<APIKey>(`/dashboard/keys/${keyId}`, {
      method: "PATCH",
      body: JSON.stringify({
        ip_whitelist: ipWhitelist,
        rate_limit: rateLimit,
        spend_limit: spendLimit,
      }),
    });
  }

  // 删除 Key
  async deleteKey(keyId: string): Promise<void> {
    return this.authRequest(`/dashboard/keys/${keyId}`, {
      method: "DELETE",
    });
  }

  async getUsageDaily(keyId: string, days = 14): Promise<UsageDailyResponse> {
    const q = new URLSearchParams({ key_id: keyId, days: String(days) });
    return this.authRequest<UsageDailyResponse>(`/dashboard/usage-daily?${q.toString()}`);
  }

  async getRequestLogs(
    keyId: string,
    limit = 20,
    offset = 0
  ): Promise<RequestLogsResponse> {
    const q = new URLSearchParams({
      key_id: keyId,
      limit: String(limit),
      offset: String(offset),
    });
    return this.authRequest<RequestLogsResponse>(`/dashboard/request-logs?${q.toString()}`);
  }

  // ==================== API Key Endpoints ====================

  async getUsage(): Promise<UsageResponse> {
    return this.request<UsageResponse>("/v1/usage");
  }

  async getTransactions(
    limit = 20,
    offset = 0
  ): Promise<TransactionsResponse> {
    return this.request<TransactionsResponse>(
      `/v1/transactions?limit=${limit}&offset=${offset}`
    );
  }

  async getModels(): Promise<ModelsResponse> {
    return this.request<ModelsResponse>("/v1/models");
  }

  /** JWT-only model list (no API key). */
  async listDashboardModels(): Promise<ModelsResponse> {
    return this.authRequest<ModelsResponse>("/dashboard/models");
  }

  /** Probe API key via GET /v1/models; returns model count on success. */
  async testApiKeyConnection(): Promise<{ ok: true; modelCount: number } | { ok: false; message: string }> {
    try {
      const res = await this.getModels();
      const count = res.data?.length ?? 0;
      return { ok: true, modelCount: count };
    } catch (e) {
      const message = e instanceof Error ? e.message : "Connection failed";
      return { ok: false, message };
    }
  }

  // Validate API key by trying to get models
  async validateKey(): Promise<boolean> {
    const r = await this.testApiKeyConnection();
    return r.ok;
  }

  async getSubscription(): Promise<{ subscription: UserSubscriptionView | null }> {
    return this.authRequest("/dashboard/subscription");
  }

  async listSubscriptionPlans(): Promise<{
    enabled: boolean;
    period_days: number;
    plans: SubscriptionPlan[];
    currency: string;
  }> {
    return this.authRequest("/dashboard/subscription/plans");
  }

  async createSubscriptionCheckout(
    planId: string
  ): Promise<{ checkout_url?: string; session_id?: string; plan_id?: string; activated?: boolean; message?: string }> {
    return this.authRequest("/dashboard/subscription/checkout", {
      method: "POST",
      body: JSON.stringify({ plan_id: planId }),
    });
  }
}

export const apiClient = new APIClient();
