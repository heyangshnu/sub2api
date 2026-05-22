"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { useAuth } from "@/lib/auth-context";
import { apiClient } from "@/lib/api";
import { TERMS_VERSION } from "@/lib/terms";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Eye, EyeOff, Lock, Mail, TrendingUp } from "lucide-react";
import { cn } from "@/lib/utils";

type AuthMode = "login" | "register" | "forgot";

export type LoginFormProps = {
  embedded?: boolean;
  initialMode?: "login" | "register";
  onSuccess?: () => void;
  onSwitchToRegister?: () => void;
  onSwitchToLogin?: () => void;
};

const inputLight =
  "h-10 w-full rounded-lg border border-slate-200 bg-white px-3 text-sm text-slate-900 shadow-sm placeholder:text-slate-400 focus-visible:border-slate-400 focus-visible:ring-2 focus-visible:ring-slate-200 md:text-sm";

const inputCompact =
  "h-9 w-full rounded-lg border border-slate-200 bg-white px-3 text-sm text-slate-900 placeholder:text-slate-400 focus-visible:border-slate-400 focus-visible:ring-2 focus-visible:ring-slate-200";

export function LoginForm({
  embedded = false,
  initialMode = "login",
  onSuccess,
  onSwitchToRegister,
  onSwitchToLogin,
}: LoginFormProps = {}) {
  const { loginWithEmail, register } = useAuth();
  const envEmailVerifyHint =
    process.env.NEXT_PUBLIC_EMAIL_VERIFY_ENABLED === "true";
  const [emailVerifyEnabled, setEmailVerifyEnabled] = useState(
    envEmailVerifyHint
  );
  const [termsVersion, setTermsVersion] = useState(TERMS_VERSION);
  const [termsAccepted, setTermsAccepted] = useState(false);
  const [mode, setMode] = useState<AuthMode>(initialMode);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState("");
  const [successMessage, setSuccessMessage] = useState("");

  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [name, setName] = useState("");
  const [verificationCode, setVerificationCode] = useState("");
  const [sendCooldown, setSendCooldown] = useState(0);
  const [showPassword, setShowPassword] = useState(false);
  const [showRegisterPassword, setShowRegisterPassword] = useState(false);
  /** After successful register: show on login card before user signs in */
  const [postRegisterNotice, setPostRegisterNotice] = useState("");

  const [forgotEmail, setForgotEmail] = useState("");
  const [resetVerificationCode, setResetVerificationCode] = useState("");
  const [resetNewPassword, setResetNewPassword] = useState("");
  const [resetConfirmPassword, setResetConfirmPassword] = useState("");
  const [resetSendCooldown, setResetSendCooldown] = useState(0);
  const [showResetNewPassword, setShowResetNewPassword] = useState(false);
  const [showResetConfirmPassword, setShowResetConfirmPassword] = useState(false);

  useEffect(() => {
    if (sendCooldown <= 0) return;
    const t = setInterval(() => {
      setSendCooldown((s) => (s <= 1 ? 0 : s - 1));
    }, 1000);
    return () => clearInterval(t);
  }, [sendCooldown]);

  useEffect(() => {
    if (resetSendCooldown <= 0) return;
    const t = setInterval(() => {
      setResetSendCooldown((s) => (s <= 1 ? 0 : s - 1));
    }, 1000);
    return () => clearInterval(t);
  }, [resetSendCooldown]);

  useEffect(() => {
    let cancelled = false;
    apiClient
      .getAuthConfig()
      .then((cfg) => {
        if (!cancelled) {
          setEmailVerifyEnabled(!!cfg.email_verify_enabled);
          if (cfg.terms_version) {
            setTermsVersion(cfg.terms_version);
          }
        }
      })
      .catch(() => {
        /* 旧后端无此接口或网络失败时沿用 NEXT_PUBLIC_* 提示 */
      });
    return () => {
      cancelled = true;
    };
  }, []);

  const switchMode = (next: AuthMode) => {
    if (embedded && next === "register" && onSwitchToRegister) {
      onSwitchToRegister();
      setMode("register");
      return;
    }
    if (embedded && next === "login" && onSwitchToLogin) {
      onSwitchToLogin();
      setMode("login");
      return;
    }
    setMode(next);
    setError("");
    setSuccessMessage("");
    if (next !== "login") {
      setPostRegisterNotice("");
    }
    if (next !== "forgot") {
      setResetVerificationCode("");
      setResetNewPassword("");
      setResetConfirmPassword("");
      setResetSendCooldown(0);
    }
    if (next === "forgot") {
      setForgotEmail(email.trim());
    }
  };

  const handleSendResetCode = async () => {
    setError("");
    setSuccessMessage("");
    const em = forgotEmail.trim();
    if (!em) {
      setError("Please enter your email");
      return;
    }
    try {
      await apiClient.sendResetPasswordCode(em);
      setSuccessMessage("Verification code sent. Check your inbox.");
      setResetSendCooldown(60);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Send failed");
    }
  };

  const handleResetPassword = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsLoading(true);
    setError("");
    setSuccessMessage("");

    if (resetNewPassword !== resetConfirmPassword) {
      setError("Passwords do not match");
      setIsLoading(false);
      return;
    }
    if (resetNewPassword.length < 6) {
      setError("Password must be at least 6 characters");
      setIsLoading(false);
      return;
    }
    const em = forgotEmail.trim();
    if (!em) {
      setError("Please enter your email");
      setIsLoading(false);
      return;
    }
    const code = resetVerificationCode.trim();
    if (!/^\d{6}$/.test(code)) {
      setError("Enter the 6-digit code from your email");
      setIsLoading(false);
      return;
    }

    try {
      await apiClient.resetPassword(em, code, resetNewPassword);
      setEmail(em);
      setPassword("");
      switchMode("login");
      setSuccessMessage("Password reset. Sign in with your new password.");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Reset failed");
    }
    setIsLoading(false);
  };

  const handleEmailLogin = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsLoading(true);
    setError("");

    const result = await loginWithEmail(email, password);
    if (!result.success) {
      setError(result.error || "Sign in failed");
    } else {
      onSuccess?.();
    }
    setIsLoading(false);
  };

  const handleRegister = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsLoading(true);
    setError("");
    setSuccessMessage("");

    if (password !== confirmPassword) {
      setError("Passwords do not match");
      setIsLoading(false);
      return;
    }

    if (password.length < 6) {
      setError("Password must be at least 6 characters");
      setIsLoading(false);
      return;
    }

    if (emailVerifyEnabled && !verificationCode.trim()) {
      setError("Enter the 6-digit verification code");
      setIsLoading(false);
      return;
    }

    if (!termsAccepted) {
      setError("You must accept the User Agreement");
      setIsLoading(false);
      return;
    }

    const result = await register(email, password, {
      name: name || undefined,
      verificationCode: emailVerifyEnabled ? verificationCode.trim() : undefined,
      termsAccepted: true,
      termsVersion,
    });
    if (result.success) {
      setPostRegisterNotice(
        `Account created. Sign in with this email and password. Top up before creating API keys.`
      );
      setSuccessMessage("");
      setError("");
      setVerificationCode("");
      setConfirmPassword("");
      setPassword("");
      setTermsAccepted(false);
      setMode("login");
    } else {
      setError(result.error || "Registration failed");
    }
    setIsLoading(false);
  };

  const handleSendRegisterCode = async () => {
    setError("");
    setSuccessMessage("");
    const em = email.trim();
    if (!em) {
      setError("Please enter your email");
      return;
    }
    if (!termsAccepted) {
      setError("Please accept the User Agreement first");
      return;
    }
    if (sendCooldown > 0) return;
    setSendCooldown(60);
    try {
      await apiClient.sendRegisterCode(em);
      setSuccessMessage("Verification code sent. Check your inbox.");
    } catch (e) {
      setSendCooldown(0);
      setError(e instanceof Error ? e.message : "Send failed");
    }
  };

  const isRegister = mode === "register";

  if (embedded) {
    return (
      <div className="px-4 py-4" data-auth-shell>
        <div className="w-full">{renderCard()}</div>
      </div>
    );
  }

  return (
    <div
      className={cn(
        "relative flex min-h-screen flex-col overflow-hidden px-4",
        isRegister ? "items-center justify-center py-4" : "overflow-x-hidden py-10 sm:py-16"
      )}
      data-auth-shell
    >
      <div
        className={cn(
          "relative z-10 mx-auto flex w-full max-w-md flex-col items-center",
          isRegister ? "gap-4" : "gap-8 sm:gap-10"
        )}
      >
        <header className="flex flex-col items-center text-center">
          <div
            className={cn(
              "flex items-center justify-center rounded-2xl bg-slate-900",
              isRegister ? "mb-2 h-10 w-10" : "mb-5 h-14 w-14"
            )}
            aria-hidden
          >
            <TrendingUp
              className={cn("text-white", isRegister ? "h-5 w-5" : "h-8 w-8")}
              strokeWidth={2.5}
            />
          </div>
          <h1
            className={cn(
              "font-semibold tracking-tight text-slate-900",
              isRegister ? "text-2xl" : "text-3xl sm:text-[1.75rem]"
            )}
          >
            Sub2API
          </h1>
          {!isRegister && (
            <p className="mt-2 max-w-xs text-sm leading-relaxed text-slate-600 sm:max-w-sm">
              OpenAI-compatible gateway · API keys, usage & billing
            </p>
          )}
        </header>

        {renderCard()}
      </div>
    </div>
  );

  function renderCard() {
    return (
      <div
        className={cn(
          "w-full rounded-2xl border border-slate-200/80 bg-white/90 shadow-xl shadow-slate-200/40 backdrop-blur-xl ring-1 ring-slate-200/60",
          embedded ? "border-0 shadow-none ring-0" : isRegister ? "p-5 sm:p-6" : "p-8 sm:p-9",
          embedded && "p-4"
        )}
      >
          {mode === "login" && (
            <>
              <h2 className="mb-6 text-lg font-semibold text-slate-900">Sign in</h2>
              {postRegisterNotice && (
                <div className="mb-5 space-y-3 rounded-xl border border-emerald-200 bg-emerald-50/95 p-4 ring-1 ring-emerald-100">
                  <p className="text-sm leading-relaxed text-emerald-900">
                    {postRegisterNotice}
                  </p>
                  <button
                    type="button"
                    className="text-sm font-medium text-emerald-700 hover:text-emerald-800 hover:underline"
                    onClick={() => setPostRegisterNotice("")}
                  >
                    Got it
                  </button>
                </div>
              )}
              <form onSubmit={handleEmailLogin} className="space-y-5">
                <div className="space-y-2">
                  <Label htmlFor="login-email" className="text-xs text-slate-600">
                    Email
                  </Label>
                  <Input
                    id="login-email"
                    type="email"
                    placeholder="your@email.com"
                    value={email}
                    onChange={(e) => setEmail(e.target.value)}
                    className={inputLight}
                    required
                    autoComplete="email"
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="login-password" className="text-xs text-slate-600">
                    Password
                  </Label>
                  <div className="relative">
                    <Input
                      id="login-password"
                      type={showPassword ? "text" : "password"}
                      placeholder="Enter password"
                      value={password}
                      onChange={(e) => setPassword(e.target.value)}
                      className={cn(inputLight, "pr-11")}
                      required
                      autoComplete="current-password"
                    />
                    <button
                      type="button"
                      tabIndex={-1}
                      className="absolute right-2.5 top-1/2 -translate-y-1/2 rounded-md p-1.5 text-slate-500 transition-colors hover:bg-slate-100 hover:text-slate-800"
                      onClick={() => setShowPassword((v) => !v)}
                      aria-label={showPassword ? "Hide password" : "Show password"}
                    >
                      {showPassword ? (
                        <EyeOff className="size-4" />
                      ) : (
                        <Eye className="size-4" />
                      )}
                    </button>
                  </div>
                </div>
                {emailVerifyEnabled && (
                  <div className="flex justify-end">
                    <button
                      type="button"
                      className="text-xs font-medium text-emerald-600 hover:text-emerald-700 hover:underline"
                      onClick={() => switchMode("forgot")}
                    >
                      Forgot password?
                    </button>
                  </div>
                )}
                {error && (
                  <p
                    role="alert"
                    className="rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-800"
                  >
                    {error}
                  </p>
                )}
                {successMessage && (
                  <div className="rounded-lg border border-emerald-200 bg-emerald-50 px-3 py-2.5 text-sm text-emerald-900">
                    {successMessage}
                  </div>
                )}
                <button
                  type="submit"
                  disabled={isLoading}
                  className="mt-2 flex h-11 w-full items-center justify-center rounded-lg bg-gradient-to-r from-emerald-500 to-cyan-500 text-sm font-semibold text-white shadow-lg shadow-emerald-500/20 transition-[filter,transform] hover:brightness-105 active:scale-[0.99] disabled:pointer-events-none disabled:opacity-50"
                >
                  {isLoading ? "Signing in…" : "Sign in"}
                </button>
              </form>
              <p className="mt-6 text-center text-sm text-slate-600">
                No account yet?{" "}
                <button
                  type="button"
                  className="font-medium text-emerald-600 transition-colors hover:text-emerald-700 hover:underline"
                  onClick={() => switchMode("register")}
                >
                  Sign up
                </button>
              </p>
            </>
          )}

          {mode === "forgot" && (
            <>
              <h2 className="mb-2 text-lg font-semibold text-slate-900">Reset password</h2>
              <p className="mb-6 text-sm text-slate-600">
                We will email a 6-digit code. Enter it below to set a new password.
              </p>
              <form onSubmit={handleResetPassword} className="space-y-5">
                <div className="space-y-2">
                  <Label htmlFor="forgot-email" className="text-xs text-slate-600">
                    Email
                  </Label>
                  <div className="relative">
                    <Mail className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-slate-500" />
                    <Input
                      id="forgot-email"
                      type="email"
                      placeholder="your@email.com"
                      value={forgotEmail}
                      onChange={(e) => setForgotEmail(e.target.value)}
                      className={cn(inputLight, "pl-10")}
                      required
                      autoComplete="email"
                    />
                  </div>
                </div>
                <div className="space-y-2 rounded-xl border border-slate-200 bg-slate-50/90 p-3">
                  <Label htmlFor="reset-code" className="text-xs text-slate-600">
                    EmailVerification code
                  </Label>
                  <div className="flex gap-2">
                    <Input
                      id="reset-code"
                      type="text"
                      inputMode="numeric"
                      autoComplete="one-time-code"
                      placeholder="6 digits"
                      maxLength={6}
                      value={resetVerificationCode}
                      onChange={(e) =>
                        setResetVerificationCode(e.target.value.replace(/\D/g, ""))
                      }
                      className={cn(
                        inputLight,
                        "flex-1 text-center font-mono tracking-[0.25em]"
                      )}
                    />
                    <Button
                      type="button"
                      variant="outline"
                      className="h-11 shrink-0 border-slate-200 bg-white text-slate-800 hover:bg-slate-50 hover:text-slate-900"
                      disabled={resetSendCooldown > 0}
                      onClick={handleSendResetCode}
                    >
                      {resetSendCooldown > 0 ? `${resetSendCooldown}s` : "Send code"}
                    </Button>
                  </div>
                  <p className="text-[11px] leading-relaxed text-slate-500">
                    Enter your email, send the code, then enter the 6 digits from the email.
                  </p>
                </div>
                <div className="space-y-2">
                  <Label htmlFor="reset-new-pw" className="text-xs text-slate-600">
                    New password
                  </Label>
                  <div className="relative">
                    <Lock className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-slate-500" />
                    <Input
                      id="reset-new-pw"
                      type={showResetNewPassword ? "text" : "password"}
                      placeholder="At least 6 characters"
                      value={resetNewPassword}
                      onChange={(e) => setResetNewPassword(e.target.value)}
                      className={cn(inputLight, "pl-10 pr-11")}
                      required
                      autoComplete="new-password"
                      minLength={6}
                    />
                    <button
                      type="button"
                      tabIndex={-1}
                      className="absolute right-2.5 top-1/2 -translate-y-1/2 rounded-md p-1.5 text-slate-500 hover:bg-slate-100 hover:text-slate-800"
                      onClick={() => setShowResetNewPassword((v) => !v)}
                      aria-label={showResetNewPassword ? "Hide password" : "Show password"}
                    >
                      {showResetNewPassword ? (
                        <EyeOff className="size-4" />
                      ) : (
                        <Eye className="size-4" />
                      )}
                    </button>
                  </div>
                </div>
                <div className="space-y-2">
                  <Label htmlFor="reset-confirm-pw" className="text-xs text-slate-600">
                    Confirm new password
                  </Label>
                  <div className="relative">
                    <Lock className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-slate-500" />
                    <Input
                      id="reset-confirm-pw"
                      type={showResetConfirmPassword ? "text" : "password"}
                      placeholder="Re-enter"
                      value={resetConfirmPassword}
                      onChange={(e) => setResetConfirmPassword(e.target.value)}
                      className={cn(inputLight, "pl-10 pr-11")}
                      required
                      autoComplete="new-password"
                      minLength={6}
                    />
                    <button
                      type="button"
                      tabIndex={-1}
                      className="absolute right-2.5 top-1/2 -translate-y-1/2 rounded-md p-1.5 text-slate-500 hover:bg-slate-100 hover:text-slate-800"
                      onClick={() => setShowResetConfirmPassword((v) => !v)}
                      aria-label={showResetConfirmPassword ? "Hide password" : "Show password"}
                    >
                      {showResetConfirmPassword ? (
                        <EyeOff className="size-4" />
                      ) : (
                        <Eye className="size-4" />
                      )}
                    </button>
                  </div>
                </div>
                {error && (
                  <p
                    role="alert"
                    className="rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-800"
                  >
                    {error}
                  </p>
                )}
                {successMessage && (
                  <div className="rounded-lg border border-emerald-200 bg-emerald-50 px-3 py-2.5 text-xs leading-relaxed text-emerald-900">
                    {successMessage}
                  </div>
                )}
                <button
                  type="submit"
                  disabled={isLoading}
                  className="flex h-11 w-full items-center justify-center rounded-lg bg-gradient-to-r from-emerald-500 to-cyan-500 text-sm font-semibold text-white shadow-lg shadow-emerald-500/20 transition-[filter,transform] hover:brightness-105 active:scale-[0.99] disabled:pointer-events-none disabled:opacity-50"
                >
                  {isLoading ? "Submitting…" : "Reset password"}
                </button>
              </form>
              <p className="mt-6 text-center text-sm text-slate-600">
                <button
                  type="button"
                  className="font-medium text-emerald-600 hover:text-emerald-700 hover:underline"
                  onClick={() => switchMode("login")}
                >
                  Back to sign in
                </button>
              </p>
            </>
          )}

          {mode === "register" && (
            <>
              <h2 className="mb-4 text-center text-lg font-semibold tracking-tight text-slate-900">
                Create account
              </h2>
              <form onSubmit={handleRegister} className="space-y-3">
                <div className="grid grid-cols-2 gap-3">
                  <div className="col-span-2 space-y-1">
                    <Label htmlFor="reg-email" className="text-[11px] font-medium text-slate-500">
                      Email
                    </Label>
                    <div className="relative">
                      <Mail className="pointer-events-none absolute left-2.5 top-1/2 size-3.5 -translate-y-1/2 text-slate-400" />
                      <Input
                        id="reg-email"
                        type="email"
                        placeholder="your@email.com"
                        value={email}
                        onChange={(e) => setEmail(e.target.value)}
                        className={cn(inputCompact, "pl-9")}
                        required
                        autoComplete="email"
                      />
                    </div>
                  </div>
                  <div className="space-y-1">
                    <Label htmlFor="reg-password" className="text-[11px] font-medium text-slate-500">
                      Password
                    </Label>
                    <div className="relative">
                      <Input
                        id="reg-password"
                        type={showRegisterPassword ? "text" : "password"}
                        placeholder="At least 6 characters"
                        value={password}
                        onChange={(e) => setPassword(e.target.value)}
                        className={cn(inputCompact, "pr-9")}
                        required
                        autoComplete="new-password"
                      />
                      <button
                        type="button"
                        tabIndex={-1}
                        className="absolute right-2 top-1/2 -translate-y-1/2 text-slate-400 hover:text-slate-700"
                        onClick={() => setShowRegisterPassword((v) => !v)}
                        aria-label={showRegisterPassword ? "Hide password" : "Show password"}
                      >
                        {showRegisterPassword ? (
                          <EyeOff className="size-3.5" />
                        ) : (
                          <Eye className="size-3.5" />
                        )}
                      </button>
                    </div>
                  </div>
                  <div className="space-y-1">
                    <Label htmlFor="reg-confirm" className="text-[11px] font-medium text-slate-500">
                      Confirm password
                    </Label>
                    <Input
                      id="reg-confirm"
                      type="password"
                      placeholder="Re-enter"
                      value={confirmPassword}
                      onChange={(e) => setConfirmPassword(e.target.value)}
                      className={inputCompact}
                      required
                      autoComplete="new-password"
                    />
                  </div>
                  {emailVerifyEnabled && (
                    <div className="col-span-2 flex gap-2">
                      <Input
                        id="reg-code"
                        type="text"
                        inputMode="numeric"
                        autoComplete="one-time-code"
                        placeholder="Verification code"
                        maxLength={6}
                        value={verificationCode}
                        onChange={(e) =>
                          setVerificationCode(e.target.value.replace(/\D/g, ""))
                        }
                        className={cn(inputCompact, "flex-1 text-center font-mono tracking-widest")}
                      />
                      <Button
                        type="button"
                        variant="outline"
                        className="h-9 shrink-0 border-slate-200 px-3 text-xs"
                        disabled={sendCooldown > 0 || isLoading}
                        onClick={handleSendRegisterCode}
                      >
                        {sendCooldown > 0 ? `Resend in ${sendCooldown}s` : "Send code"}
                      </Button>
                    </div>
                  )}
                </div>
                {error && (
                  <p role="alert" className="rounded-lg bg-red-50 px-3 py-1.5 text-xs text-red-800">
                    {error}
                  </p>
                )}
                {successMessage && (
                  <p className="rounded-lg bg-emerald-50 px-3 py-1.5 text-xs text-emerald-900">
                    {successMessage}
                  </p>
                )}
                <label className="flex cursor-pointer items-start gap-2 rounded-lg border border-slate-200 bg-slate-50/80 px-3 py-2.5 text-xs leading-relaxed text-slate-700">
                  <input
                    type="checkbox"
                    className="mt-0.5 size-3.5 shrink-0 rounded border-slate-300"
                    checked={termsAccepted}
                    onChange={(e) => setTermsAccepted(e.target.checked)}
                  />
                  <span>
                    I have read and agree to the {" "}
                    <Link
                      href="/terms"
                      target="_blank"
                      rel="noopener noreferrer"
                      className="font-medium text-emerald-700 underline hover:text-emerald-800"
                      onClick={(e) => e.stopPropagation()}
                    >
                      User Agreement & Privacy Notice
                    </Link>
                    (version {termsVersion})
                  </span>
                </label>
                <button
                  type="submit"
                  disabled={isLoading || !termsAccepted}
                  className="flex h-10 w-full items-center justify-center rounded-full bg-slate-900 text-sm font-medium text-white transition-colors hover:bg-slate-800 disabled:opacity-50"
                >
                  {isLoading ? "Creating…" : "Sign up"}
                </button>
              </form>
              <p className="mt-4 text-center text-xs text-slate-500">
                Already have an account?{" "}
                <button
                  type="button"
                  className="font-medium text-slate-900 hover:underline"
                  onClick={() => switchMode("login")}
                >
                  Sign in
                </button>
              </p>
            </>
          )}

      </div>
    );
  }
}
