package store

// Ledger types counted as API/chat consumption for analytics.
func isConsumeLedgerType(t string) bool {
	switch t {
	case "chat_consume", "api_consume", "consume":
		return true
	default:
		return false
	}
}

func isTopupLedgerType(t string) bool {
	switch t {
	case "topup", "admin_topup":
		return true
	default:
		return false
	}
}
