const TYPE_KEYS: Record<string, string> = {
  topup: "tx.topup",
  admin_topup: "tx.admin_topup",
  monthly_grant: "tx.monthly_grant",
  subscription_grant: "tx.subscription_grant",
  admin_adjust: "tx.admin_adjust",
  chat_consume: "tx.chat_consume",
  api_consume: "tx.api_consume",
  consume: "tx.consume",
  refund: "tx.refund",
};

export function transactionTypeLabel(
  type: string,
  t?: (key: string) => string
): string {
  const key = TYPE_KEYS[type];
  if (t && key) {
    const label = t(key);
    if (label !== key) return label;
  }
  return type;
}

export function isTopupType(type: string): boolean {
  return ["topup", "admin_topup", "monthly_grant", "subscription_grant", "admin_adjust"].includes(
    type
  );
}

export function isConsumeType(type: string): boolean {
  return ["chat_consume", "api_consume", "consume"].includes(type);
}
