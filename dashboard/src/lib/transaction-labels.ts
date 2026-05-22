const TYPE_LABELS: Record<string, string> = {
  topup: "Top-up",
  admin_topup: "Admin top-up",
  monthly_grant: "Monthly grant",
  subscription_grant: "Subscription credit",
  admin_adjust: "Adjustment",
  chat_consume: "Chat usage",
  api_consume: "API usage",
  consume: "Usage",
  refund: "Refund",
};

export function transactionTypeLabel(type: string): string {
  return TYPE_LABELS[type] || type;
}

export function isTopupType(type: string): boolean {
  return ["topup", "admin_topup", "monthly_grant", "subscription_grant", "admin_adjust"].includes(
    type
  );
}

export function isConsumeType(type: string): boolean {
  return ["chat_consume", "api_consume", "consume"].includes(type);
}
