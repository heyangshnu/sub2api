"use client";

import {
  createContext,
  useCallback,
  useContext,
  useRef,
  useState,
  useEffect,
  ReactNode,
} from "react";
import { apiClient, User, UserProfile, APIKey } from "@/lib/api";
import { clearSloganSession, flagSloganAfterLogin, hasSloganPlayed } from "@/lib/brand";

export type AuthDialogTab = "login" | "register";

interface AuthContextType {
  isAuthenticated: boolean;
  isGuest: boolean;
  isLoading: boolean;
  authMode: "none" | "jwt";
  user: User | null;
  userProfile: UserProfile | null;
  apiKey: string | null;
  apiKeys: APIKey[];
  authDialogOpen: boolean;
  authDialogTab: AuthDialogTab;
  refreshProfile: () => Promise<void>;
  loginWithEmail: (email: string, password: string) => Promise<{ success: boolean; error?: string }>;
  register: (
    email: string,
    password: string,
    options: {
      name?: string;
      verificationCode?: string;
      termsAccepted: boolean;
      termsVersion: string;
    }
  ) => Promise<{ success: boolean; error?: string }>;
  logout: () => void;
  refreshKeys: () => Promise<void>;
  bindUsageApiKey: (rawKey: string) => void;
  openAuthDialog: (tab?: AuthDialogTab) => void;
  closeAuthDialog: () => void;
  requireAuth: (action: () => void | Promise<void>, tab?: AuthDialogTab) => void;
  onAuthSuccess: () => void;
  /** Bumps on each login to start slogan hero in console shell */
  sloganPlayId: number;
  sloganPinned: boolean;
  setSloganPinned: (pinned: boolean) => void;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

const JWT_TOKEN_STORAGE_KEY = "sub2api_token";
const USER_STORAGE_KEY = "sub2api_user";
const API_KEY_STORAGE_KEY = "sub2api_key";

export function AuthProvider({ children }: { children: ReactNode }) {
  const [isAuthenticated, setIsAuthenticated] = useState(false);
  const [isLoading, setIsLoading] = useState(true);
  const [authMode, setAuthMode] = useState<"none" | "jwt">("none");
  const [user, setUser] = useState<User | null>(null);
  const [userProfile, setUserProfile] = useState<UserProfile | null>(null);
  const [apiKey, setApiKey] = useState<string | null>(null);
  const [apiKeys, setApiKeys] = useState<APIKey[]>([]);
  const [authDialogOpen, setAuthDialogOpen] = useState(false);
  const [authDialogTab, setAuthDialogTab] = useState<AuthDialogTab>("login");
  const pendingActionRef = useRef<(() => void | Promise<void>) | null>(null);
  const [sloganPlayId, setSloganPlayId] = useState(0);
  const [sloganPinned, setSloganPinned] = useState(false);

  const triggerSloganOnLogin = useCallback(() => {
    flagSloganAfterLogin();
    setSloganPlayId((n) => n + 1);
  }, []);

  const refreshProfile = useCallback(async () => {
    if (!apiClient.getToken()) return;
    try {
      const profile = await apiClient.getMe();
      setUserProfile(profile);
      setUser({
        id: profile.id,
        email: profile.email,
        name: profile.name,
        status: profile.status,
        created_at: "",
      });
    } catch {
      /* ignore */
    }
  }, []);

  const refreshKeys = useCallback(async () => {
    if (!apiClient.getToken()) return;
    try {
      const { keys } = await apiClient.getMyKeys();
      setApiKeys(keys);
      const savedApiKey = localStorage.getItem(API_KEY_STORAGE_KEY);
      if (savedApiKey) {
        apiClient.setApiKey(savedApiKey);
        setApiKey(savedApiKey);
      }
    } catch {
      /* ignore */
    }
  }, []);

  useEffect(() => {
    const init = async () => {
      const savedToken = localStorage.getItem(JWT_TOKEN_STORAGE_KEY);
      const savedUser = localStorage.getItem(USER_STORAGE_KEY);

      if (savedToken && savedUser) {
        apiClient.setToken(savedToken);
        try {
          const currentUser = await apiClient.getMe();
          setUserProfile(currentUser);
          setUser({
            id: currentUser.id,
            email: currentUser.email,
            name: currentUser.name,
            status: currentUser.status,
            created_at: "",
          });
          setIsAuthenticated(true);
          setAuthMode("jwt");
          const { keys } = await apiClient.getMyKeys();
          setApiKeys(keys);
          const savedApiKey = localStorage.getItem(API_KEY_STORAGE_KEY);
          if (savedApiKey) {
            apiClient.setApiKey(savedApiKey);
            setApiKey(savedApiKey);
          }
        } catch {
          localStorage.removeItem(JWT_TOKEN_STORAGE_KEY);
          localStorage.removeItem(USER_STORAGE_KEY);
          apiClient.clearToken();
        }
      }

      setIsLoading(false);
      if (apiClient.getToken() && hasSloganPlayed()) {
        setSloganPinned(true);
      }
    };

    void init();
  }, []);

  const loginWithEmail = async (
    email: string,
    password: string
  ): Promise<{ success: boolean; error?: string }> => {
    try {
      const response = await apiClient.login(email, password);
      if (!response.token || !response.user) {
        return { success: false, error: "Login did not return a valid session" };
      }
      localStorage.setItem(JWT_TOKEN_STORAGE_KEY, response.token);
      localStorage.setItem(USER_STORAGE_KEY, JSON.stringify(response.user));
      apiClient.setToken(response.token);
      setUser(response.user);
      setIsAuthenticated(true);
      setAuthMode("jwt");
      await refreshProfile();
      await refreshKeys();
      triggerSloganOnLogin();
      return { success: true };
    } catch (err) {
      return { success: false, error: err instanceof Error ? err.message : "Login failed" };
    }
  };

  const register = async (
    email: string,
    password: string,
    options: {
      name?: string;
      verificationCode?: string;
      termsAccepted: boolean;
      termsVersion: string;
    }
  ): Promise<{ success: boolean; error?: string }> => {
    try {
      const response = await apiClient.register(email, password, options);
      if (!response.user) {
        return { success: false, error: "Registration did not return user info" };
      }
      return { success: true };
    } catch (err) {
      return { success: false, error: err instanceof Error ? err.message : "Registration failed" };
    }
  };

  const logout = () => {
    clearSloganSession();
    setSloganPinned(false);
    setSloganPlayId(0);
    localStorage.removeItem(API_KEY_STORAGE_KEY);
    localStorage.removeItem(JWT_TOKEN_STORAGE_KEY);
    localStorage.removeItem(USER_STORAGE_KEY);
    apiClient.clearAll();
    setApiKey(null);
    setUser(null);
    setUserProfile(null);
    setApiKeys([]);
    setIsAuthenticated(false);
    setAuthMode("none");
  };

  const bindUsageApiKey = (rawKey: string) => {
    localStorage.setItem(API_KEY_STORAGE_KEY, rawKey);
    apiClient.setApiKey(rawKey);
    setApiKey(rawKey);
  };

  const openAuthDialog = (tab: AuthDialogTab = "login") => {
    setAuthDialogTab(tab);
    setAuthDialogOpen(true);
  };

  const closeAuthDialog = () => {
    pendingActionRef.current = null;
    setAuthDialogOpen(false);
  };

  const onAuthSuccess = () => {
    setAuthDialogOpen(false);
    const pending = pendingActionRef.current;
    pendingActionRef.current = null;
    if (pending) void Promise.resolve(pending());
  };

  const requireAuth = (action: () => void | Promise<void>, tab: AuthDialogTab = "login") => {
    if (isAuthenticated) {
      void Promise.resolve(action());
      return;
    }
    pendingActionRef.current = action;
    openAuthDialog(tab);
  };

  return (
    <AuthContext.Provider
      value={{
        isAuthenticated,
        isGuest: !isAuthenticated,
        isLoading,
        authMode,
        user,
        userProfile,
        apiKey,
        apiKeys,
        authDialogOpen,
        authDialogTab,
        refreshProfile,
        loginWithEmail,
        register,
        logout,
        refreshKeys,
        bindUsageApiKey,
        openAuthDialog,
        closeAuthDialog,
        requireAuth,
        onAuthSuccess,
        sloganPlayId,
        sloganPinned,
        setSloganPinned,
      }}
    >
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
