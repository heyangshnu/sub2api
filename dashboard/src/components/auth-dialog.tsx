"use client";

import { useAuth } from "@/lib/auth-context";
import { LoginForm } from "@/components/login-form";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";

export function AuthDialog() {
  const { authDialogOpen, authDialogTab, closeAuthDialog, onAuthSuccess, openAuthDialog } =
    useAuth();

  return (
    <Dialog open={authDialogOpen} onOpenChange={(open) => !open && closeAuthDialog()}>
      <DialogContent className="max-h-[90vh] max-w-md overflow-y-auto border-slate-200 bg-white p-0 sm:max-w-md">
        <DialogHeader className="sr-only">
          <DialogTitle>{authDialogTab === "register" ? "注册" : "登录"}</DialogTitle>
          <DialogDescription>登录或注册 Sub2API 控制台账户</DialogDescription>
        </DialogHeader>
        <LoginForm
          key={authDialogTab}
          embedded
          initialMode={authDialogTab}
          onSuccess={onAuthSuccess}
          onSwitchToRegister={() => openAuthDialog("register")}
          onSwitchToLogin={() => openAuthDialog("login")}
        />
      </DialogContent>
    </Dialog>
  );
}
