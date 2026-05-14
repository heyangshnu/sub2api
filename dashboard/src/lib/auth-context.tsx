"use client";

import { createContext, useContext, useState, useEffect, ReactNode } from "react";
import { apiClient, User, APIKey } from "@/lib/api";

type AuthMode = "none" | "api_key" | "jwt";

interface AuthContextType {
  isAuthenticated: boolean;
  isLoading: boolean;
  authMode: AuthMode;
  user: User | null;
  apiKey: string | null;
  apiKeys: APIKey[];
  
  // API Key login (legacy)
  loginWithApiKey: (apiKey: string) => Promise<boolean>;
  
  // JWT login/register
  loginWithEmail: (email: string, password: string) => Promise<{ success: boolean; error?: string }>;
  register: (
    email: string,
    password: string,
    name?: string,
    inviteCode?: string,
    verificationCode?: string
  ) => Promise<{ success: boolean; error?: string }>;
  
  logout: () => void;
  refreshKeys: () => Promise<void>;
  /** After creating an API key, attach it for /v1/* usage & balance queries */
  bindUsageApiKey: (rawKey: string) => void;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

const API_KEY_STORAGE_KEY = "sub2api_key";
const JWT_TOKEN_STORAGE_KEY = "sub2api_token";
const USER_STORAGE_KEY = "sub2api_user";

export function AuthProvider({ children }: { children: ReactNode }) {
  const [isAuthenticated, setIsAuthenticated] = useState(false);
  const [isLoading, setIsLoading] = useState(true);
  const [authMode, setAuthMode] = useState<AuthMode>("none");
  const [user, setUser] = useState<User | null>(null);
  const [apiKey, setApiKey] = useState<string | null>(null);
  const [apiKeys, setApiKeys] = useState<APIKey[]>([]);

  // Check for saved auth on mount
  useEffect(() => {
    const init = async () => {
      // Check for JWT token first
      const savedToken = localStorage.getItem(JWT_TOKEN_STORAGE_KEY);
      const savedUser = localStorage.getItem(USER_STORAGE_KEY);
      
      if (savedToken && savedUser) {
        apiClient.setToken(savedToken);
        try {
          // Verify token is still valid
          const currentUser = await apiClient.getMe();
          setUser(currentUser);
          setIsAuthenticated(true);
          setAuthMode("jwt");
          
          // Load user's API keys
          const { keys } = await apiClient.getMyKeys();
          setApiKeys(keys);
          
          // If user has keys, set the first one for usage queries
          if (keys.length > 0) {
            // We don't have the raw key, but we can use JWT for dashboard
          }
        } catch {
          // Token invalid, clear storage
          localStorage.removeItem(JWT_TOKEN_STORAGE_KEY);
          localStorage.removeItem(USER_STORAGE_KEY);
          apiClient.clearToken();
        }
        setIsLoading(false);
        return;
      }

      // Fall back to API key auth
      const savedApiKey = localStorage.getItem(API_KEY_STORAGE_KEY);
      if (savedApiKey) {
        apiClient.setApiKey(savedApiKey);
        const valid = await apiClient.validateKey();
        if (valid) {
          setApiKey(savedApiKey);
          setIsAuthenticated(true);
          setAuthMode("api_key");
        } else {
          localStorage.removeItem(API_KEY_STORAGE_KEY);
          apiClient.clearApiKey();
        }
      }
      
      setIsLoading(false);
    };
    
    init();
  }, []);

  // Login with API Key (legacy method)
  const loginWithApiKey = async (key: string): Promise<boolean> => {
    apiClient.setApiKey(key);
    const valid = await apiClient.validateKey();
    if (valid) {
      localStorage.setItem(API_KEY_STORAGE_KEY, key);
      setApiKey(key);
      setIsAuthenticated(true);
      setAuthMode("api_key");
    } else {
      apiClient.clearApiKey();
    }
    return valid;
  };

  // Login with email/password
  const loginWithEmail = async (email: string, password: string): Promise<{ success: boolean; error?: string }> => {
    try {
      const response = await apiClient.login(email, password);

      if (!response.token || !response.user) {
        return { success: false, error: "Login did not return a valid session" };
      }

      // Save token and user
      localStorage.setItem(JWT_TOKEN_STORAGE_KEY, response.token);
      localStorage.setItem(USER_STORAGE_KEY, JSON.stringify(response.user));

      apiClient.setToken(response.token);
      setUser(response.user);
      setIsAuthenticated(true);
      setAuthMode("jwt");
      
      // Load user's API keys
      try {
        const { keys } = await apiClient.getMyKeys();
        setApiKeys(keys);
      } catch {
        // Ignore error, user might not have keys yet
      }
      
      return { success: true };
    } catch (err) {
      return { success: false, error: err instanceof Error ? err.message : "Login failed" };
    }
  };

  // Register: creates account on server but does not open a session — user must log in with email/password.
  const register = async (
    email: string,
    password: string,
    name?: string,
    inviteCode?: string,
    verificationCode?: string
  ): Promise<{ success: boolean; error?: string; apiKey?: string }> => {
    try {
      const response = await apiClient.register(
        email,
        password,
        name,
        inviteCode,
        verificationCode
      );

      if (!response.user) {
        return { success: false, error: "Registration did not return user info" };
      }

      return { success: true };
    } catch (err) {
      return { success: false, error: err instanceof Error ? err.message : "Registration failed" };
    }
  };

  const logout = () => {
    localStorage.removeItem(API_KEY_STORAGE_KEY);
    localStorage.removeItem(JWT_TOKEN_STORAGE_KEY);
    localStorage.removeItem(USER_STORAGE_KEY);
    apiClient.clearAll();
    setApiKey(null);
    setUser(null);
    setApiKeys([]);
    setIsAuthenticated(false);
    setAuthMode("none");
  };

  const refreshKeys = async () => {
    if (authMode === "jwt") {
      try {
        const { keys } = await apiClient.getMyKeys();
        setApiKeys(keys);
      } catch {
        // Ignore
      }
    }
  };

  const bindUsageApiKey = (rawKey: string) => {
    localStorage.setItem(API_KEY_STORAGE_KEY, rawKey);
    apiClient.setApiKey(rawKey);
    setApiKey(rawKey);
  };

  return (
    <AuthContext.Provider value={{
      isAuthenticated,
      isLoading,
      authMode,
      user,
      apiKey,
      apiKeys,
      loginWithApiKey,
      loginWithEmail,
      register,
      logout,
      refreshKeys,
      bindUsageApiKey,
    }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  const context = useContext(AuthContext);
  if (context === undefined) {
    throw new Error("useAuth must be used within an AuthProvider");
  }
  return context;
}
