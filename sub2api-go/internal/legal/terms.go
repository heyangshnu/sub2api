package legal

import "os"

// CurrentTermsVersion is the only version accepted at registration (override via TERMS_VERSION env).
func CurrentTermsVersion() string {
	if v := os.Getenv("TERMS_VERSION"); v != "" {
		return v
	}
	return "2026-05-22"
}
