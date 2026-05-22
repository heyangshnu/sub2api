const TYPE_LABELS: Record<string, string> = {
  topup: "充值",
  admin_topup: "管理员充值",
  monthly_grant: "月赠",
  subscription_grant: "订阅赠送",
  admin_adjust: "调账",
  chat_consume: "对话消费",
  api_consume: "API 消费",
  consume: "消费",
  refund: "退款",
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
