// API Client for Sub2API Backend

const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

// ==================== Auth Types ====================

export interface User {
  id: string;
  email: string;
  name: string;
  status: string;
  created_at: string;
}

export interface AuthResponse {
  token: string;
  user: User;
  api_key?: string;  // First registration returns API key
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
  key_id: string;
  type: "consume" | "topup" | "refund";
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

    if (!response.ok) {
      const error: APIError = await response.json();
      throw new Error(error.error?.message || "Request failed");
    }

    return response.json();
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

    if (!response.ok) {
      const error = await response.json();
      // Handle both { error: "message" } and { error: { message: "xxx" } } formats
      const errorMessage = typeof error.error === "string" 
        ? error.error 
        : error.error?.message || error.message || "Request failed";
      throw new Error(errorMessage);
    }

    return response.json();
  }

  // ==================== Auth Endpoints ====================

  async register(email: string, password: string, name?: string, inviteCode?: string): Promise<AuthResponse> {
    return this.authRequest<AuthResponse>("/auth/register", {
      method: "POST",
      body: JSON.stringify({ email, password, name, invite_code: inviteCode }),
    });
  }

  async login(email: string, password: string): Promise<AuthResponse> {
    return this.authRequest<AuthResponse>("/auth/login", {
      method: "POST",
      body: JSON.stringify({ email, password }),
    });
  }

  async getMe(): Promise<User> {
    return this.authRequest<User>("/dashboard/me");
  }

  async getMyKeys(): Promise<{ keys: APIKey[] }> {
    return this.authRequest<{ keys: APIKey[] }>("/dashboard/keys");
  }

  // 创建 Key（需密码二次验证）
  async createKey(
    password: string,
    name?: string,
    rateLimit?: number
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
      body: JSON.stringify({ password, name, rate_limit: rateLimit }),
    });
  }

  // 更新 Key 设置
  async updateKeySettings(
    keyId: string,
    ipWhitelist?: string[],
    rateLimit?: number
  ): Promise<APIKey> {
    return this.authRequest<APIKey>(`/dashboard/keys/${keyId}`, {
      method: "PATCH",
      body: JSON.stringify({ ip_whitelist: ipWhitelist, rate_limit: rateLimit }),
    });
  }

  // 删除 Key
  async deleteKey(keyId: string): Promise<void> {
    return this.authRequest(`/dashboard/keys/${keyId}`, {
      method: "DELETE",
    });
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

  // Validate API key by trying to get usage
  async validateKey(): Promise<boolean> {
    try {
      await this.getUsage();
      return true;
    } catch {
      return false;
    }
  }
}

export const apiClient = new APIClient();
