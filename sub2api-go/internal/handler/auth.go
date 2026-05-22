package handler

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"net/smtp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"sub2api-go/internal/config"
	"sub2api-go/internal/legal"
	"sub2api-go/internal/middleware"
	"sub2api-go/internal/model"
	"sub2api-go/internal/service"
	"sub2api-go/internal/store"
)

type AuthHandler struct {
	store              store.Store
	jwtSecret          []byte
	inviteCode         string
	emailVerifyEnabled bool
	smtpHost           string
	smtpPort           int
	smtpUsername       string
	smtpPassword       string
	smtpFrom           string
	cfg                *config.Config
}

func NewAuthHandler(
	s store.Store,
	cfg *config.Config,
	jwtSecret, inviteCode string,
	emailVerifyEnabled bool,
	smtpHost string,
	smtpPort int,
	smtpUsername, smtpPassword, smtpFrom string,
) *AuthHandler {
	return &AuthHandler{
		store:              s,
		cfg:                cfg,
		jwtSecret:          []byte(jwtSecret),
		inviteCode:         inviteCode,
		emailVerifyEnabled: emailVerifyEnabled,
		smtpHost:           smtpHost,
		smtpPort:           smtpPort,
		smtpUsername:       smtpUsername,
		smtpPassword:       smtpPassword,
		smtpFrom:           smtpFrom,
	}
}

// AuthConfig handles GET /auth/config — public flags for the dashboard (no auth).
func (h *AuthHandler) AuthConfig(c *gin.Context) {
	inviteRequired := strings.TrimSpace(h.inviteCode) != ""
	models := []string{"deepseek-chat"}
	if h.cfg != nil && len(h.cfg.ChatEnabledModels) > 0 {
		models = h.cfg.ChatEnabledModels
	}
	out := gin.H{
		"email_verify_enabled": h.emailVerifyEnabled,
		"invite_required":      inviteRequired,
		"terms_version":        legal.CurrentTermsVersion(),
		"terms_required":       true,
		"chat_enabled_models":  models,
		"currency":             "USD",
	}
	if h.cfg != nil {
		out["subscriptions_enabled"] = h.cfg.SubscriptionsEnabled
		out["subscription_period_days"] = h.cfg.SubscriptionPeriodDays
		if h.cfg.SubscriptionsEnabled {
			out["subscription_plans"] = h.cfg.SubscriptionPlans
		}
	}
	c.JSON(http.StatusOK, out)
}

// Register handles POST /auth/register
func (h *AuthHandler) Register(c *gin.Context) {
	var req model.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.NewAPIError("invalid_request_error", "Invalid request: "+err.Error()))
		return
	}

	if msg := validateTermsAcceptance(req.TermsAccepted, req.TermsVersion); msg != "" {
		c.JSON(http.StatusBadRequest, model.NewAPIError("invalid_request_error", msg))
		return
	}

	_, err := h.store.GetUserByEmail(c.Request.Context(), req.Email)
	if err == nil {
		c.JSON(http.StatusConflict, model.NewAPIError("conflict", "Email already registered"))
		return
	}

	if h.emailVerifyEnabled {
		code := strings.TrimSpace(req.VerificationCode)
		if code == "" {
			c.JSON(http.StatusBadRequest, model.NewAPIError("invalid_request_error", "verification_code is required"))
			return
		}
		if err := h.store.ConsumeRegisterOTP(c.Request.Context(), req.Email, code); err != nil {
			if err == store.ErrRegisterOTPInvalid {
				c.JSON(http.StatusBadRequest, model.NewAPIError("invalid_request_error", "Invalid or expired verification code"))
				return
			}
			c.JSON(http.StatusInternalServerError, model.NewAPIError("internal_error", "Failed to verify code"))
			return
		}
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.NewAPIError("internal_error", "Failed to hash password"))
		return
	}

	now := time.Now()
	termsAt := now
	userID := generateUserID(req.Email)
	user := &model.User{
		ID:                   userID,
		Email:                req.Email,
		PasswordHash:         string(passwordHash),
		Name:                 req.Name,
		Status:               "active",
		EmailVerified:        true,
		EmailVerifyTokenHash: "",
		EmailVerifyExpiresAt: nil,
		TermsAcceptedAt:      &termsAt,
		TermsVersion:         strings.TrimSpace(req.TermsVersion),
		CreatedAt:            now,
		UpdatedAt:            now,
	}

	if user.Name == "" {
		user.Name = req.Email
	}

	if err := h.store.CreateUser(c.Request.Context(), user); err != nil {
		if err == store.ErrUserExists {
			c.JSON(http.StatusConflict, model.NewAPIError("conflict", "Email already registered"))
			return
		}
		c.JSON(http.StatusInternalServerError, model.NewAPIError("internal_error", "Failed to create user"))
		return
	}

	// 不在注册时创建 API Key；用户登录并充值后在控制台自行创建
	token, err := h.generateToken(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.NewAPIError("internal_error", "Failed to generate token"))
		return
	}

	c.JSON(http.StatusCreated, model.AuthResponse{
		Token: token,
		User:  user,
	})
}

// SendRegisterCode handles POST /auth/send-register-code — sends a 6-digit code to email (registration only).
func (h *AuthHandler) SendRegisterCode(c *gin.Context) {
	if !h.emailVerifyEnabled {
		c.JSON(http.StatusBadRequest, model.NewAPIError("invalid_request_error", "Email verification is not enabled on this server"))
		return
	}

	var req model.SendRegisterCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.NewAPIError("invalid_request_error", "Invalid request: "+err.Error()))
		return
	}

	email := strings.TrimSpace(req.Email)
	if _, err := h.store.GetUserByEmail(c.Request.Context(), email); err == nil {
		c.JSON(http.StatusConflict, model.NewAPIError("conflict", "Email already registered"))
		return
	}

	code, err := generateSixDigitCode()
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.NewAPIError("internal_error", "Failed to generate code"))
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(code), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.NewAPIError("internal_error", "Failed to hash code"))
		return
	}

	now := time.Now()
	expires := now.Add(15 * time.Minute)
	if err := h.store.SaveRegisterOTP(c.Request.Context(), email, string(hash), expires, now); err != nil {
		if err == store.ErrRegisterOTPCooldown {
			c.JSON(http.StatusTooManyRequests, model.NewAPIError("rate_limit_error", "Please wait a minute before requesting another code"))
			return
		}
		c.JSON(http.StatusInternalServerError, model.NewAPIError("internal_error", "Failed to save verification code"))
		return
	}

	if err := h.sendRegisterCodeEmail(email, code); err != nil {
		log.Printf("send register code email: %v", err)
		msg := "Failed to send email"
		if strings.Contains(err.Error(), "SMTP is not fully configured") {
			msg = "SMTP is not fully configured on the server"
		} else if strings.Contains(strings.ToLower(err.Error()), "auth") {
			msg = "SMTP authentication failed (check username and app password)"
		} else if strings.Contains(strings.ToLower(err.Error()), "tls") ||
			strings.Contains(strings.ToLower(err.Error()), "connection") ||
			strings.Contains(strings.ToLower(err.Error()), "refused") {
			msg = "Could not connect to mail server (try SMTP_PORT 465 with SSL or 587 with STARTTLS)"
		}
		c.JSON(http.StatusInternalServerError, model.NewAPIError("internal_error", msg))
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Verification code sent"})
}

// SendResetPasswordCode handles POST /auth/send-reset-password-code — 6-digit code to reset password (same SMTP rules as registration).
func (h *AuthHandler) SendResetPasswordCode(c *gin.Context) {
	if !h.emailVerifyEnabled {
		c.JSON(http.StatusBadRequest, model.NewAPIError("invalid_request_error", "Email verification is not enabled on this server"))
		return
	}

	var req model.SendResetPasswordCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.NewAPIError("invalid_request_error", "Invalid request: "+err.Error()))
		return
	}

	email := strings.TrimSpace(req.Email)
	user, err := h.store.GetUserByEmail(c.Request.Context(), email)
	if err != nil || user.Status != "active" {
		// Do not reveal whether the email is registered
		c.JSON(http.StatusOK, gin.H{"message": "Verification code sent"})
		return
	}

	code, err := generateSixDigitCode()
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.NewAPIError("internal_error", "Failed to generate code"))
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(code), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.NewAPIError("internal_error", "Failed to hash code"))
		return
	}

	now := time.Now()
	expires := now.Add(15 * time.Minute)
	if err := h.store.SaveResetPasswordOTP(c.Request.Context(), email, string(hash), expires, now); err != nil {
		if err == store.ErrResetPasswordOTPCooldown {
			c.JSON(http.StatusTooManyRequests, model.NewAPIError("rate_limit_error", "Please wait a minute before requesting another code"))
			return
		}
		c.JSON(http.StatusInternalServerError, model.NewAPIError("internal_error", "Failed to save verification code"))
		return
	}

	if err := h.sendResetPasswordEmail(email, code); err != nil {
		log.Printf("send reset password code email: %v", err)
		msg := "Failed to send email"
		if strings.Contains(err.Error(), "SMTP is not fully configured") {
			msg = "SMTP is not fully configured on the server"
		} else if strings.Contains(strings.ToLower(err.Error()), "auth") {
			msg = "SMTP authentication failed (check username and app password)"
		} else if strings.Contains(strings.ToLower(err.Error()), "tls") ||
			strings.Contains(strings.ToLower(err.Error()), "connection") ||
			strings.Contains(strings.ToLower(err.Error()), "refused") {
			msg = "Could not connect to mail server (try SMTP_PORT 465 with SSL or 587 with STARTTLS)"
		}
		c.JSON(http.StatusInternalServerError, model.NewAPIError("internal_error", msg))
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Verification code sent"})
}

// ResetPassword handles POST /auth/reset-password — consumes email OTP and sets a new password.
func (h *AuthHandler) ResetPassword(c *gin.Context) {
	if !h.emailVerifyEnabled {
		c.JSON(http.StatusBadRequest, model.NewAPIError("invalid_request_error", "Email verification is not enabled on this server"))
		return
	}

	var req model.ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.NewAPIError("invalid_request_error", "Invalid request: "+err.Error()))
		return
	}

	email := strings.TrimSpace(req.Email)
	user, err := h.store.GetUserByEmail(c.Request.Context(), email)
	if err != nil || user.Status != "active" {
		c.JSON(http.StatusBadRequest, model.NewAPIError("invalid_request_error", "Invalid or expired verification code"))
		return
	}

	if err := h.store.ConsumeResetPasswordOTP(c.Request.Context(), email, req.VerificationCode); err != nil {
		if err == store.ErrResetPasswordOTPInvalid {
			c.JSON(http.StatusBadRequest, model.NewAPIError("invalid_request_error", "Invalid or expired verification code"))
			return
		}
		c.JSON(http.StatusInternalServerError, model.NewAPIError("internal_error", "Failed to verify code"))
		return
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.NewAPIError("internal_error", "Failed to hash password"))
		return
	}
	user.PasswordHash = string(passwordHash)
	user.UpdatedAt = time.Now()

	if err := h.store.UpdateUser(c.Request.Context(), user); err != nil {
		if err == store.ErrUserNotFound {
			c.JSON(http.StatusBadRequest, model.NewAPIError("invalid_request_error", "Invalid or expired verification code"))
			return
		}
		c.JSON(http.StatusInternalServerError, model.NewAPIError("internal_error", "Failed to update password"))
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Password has been reset"})
}

// Login handles POST /auth/login
func (h *AuthHandler) Login(c *gin.Context) {
	var req model.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.NewAPIError("invalid_request_error", "Invalid request: "+err.Error()))
		return
	}

	user, err := h.store.GetUserByEmail(c.Request.Context(), req.Email)
	if err != nil {
		c.JSON(http.StatusUnauthorized, model.NewAPIError("authentication_error", "Invalid email or password"))
		return
	}

	if user.Status == "pending_verification" {
		c.JSON(http.StatusForbidden, model.NewAPIError("forbidden", "Account is not active. Complete registration or contact support."))
		return
	}
	if user.Status != "active" {
		c.JSON(http.StatusForbidden, model.NewAPIError("forbidden", "Account is disabled"))
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, model.NewAPIError("authentication_error", "Invalid email or password"))
		return
	}

	token, err := h.generateToken(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.NewAPIError("internal_error", "Failed to generate token"))
		return
	}

	c.JSON(http.StatusOK, model.AuthResponse{
		Token: token,
		User:  user,
	})
}

// GetMe handles GET /dashboard/me
func (h *AuthHandler) GetMe(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, model.NewAPIError("authentication_error", "Not authenticated"))
		return
	}

	user, err := h.store.GetUserByID(c.Request.Context(), userID.(string))
	if err != nil {
		c.JSON(http.StatusNotFound, model.NewAPIError("not_found", "User not found"))
		return
	}
	grantUSD := 0.1
	if h.cfg != nil {
		grantUSD = h.cfg.AccountMonthlyGrantUSD
	}
	if grantUSD > 0 {
		_, _ = h.store.TryMonthlyGrant(c.Request.Context(), user.ID, grantUSD)
	}

	spendable, _ := h.store.GetAccountBalance(c.Request.Context(), user.ID)
	recharged, _ := h.store.GetAccountRechargedBalance(c.Request.Context(), user.ID)
	user.Balance = spendable

	canCreate := user.HasPaid
	if h.cfg != nil && !h.cfg.RequirePaymentBeforeCreateKey {
		canCreate = true
	}
	subSvc := service.NewSubscriptionService(h.store, h.cfg)
	_ = subSvc.EnsureUser(c.Request.Context(), user.ID)

	c.JSON(http.StatusOK, model.UserProfile{
		ID:               user.ID,
		Email:            user.Email,
		Name:             user.Name,
		Status:           user.Status,
		Balance:          recharged,
		SpendableBalance: spendable,
		HasPaid:          user.HasPaid,
		CanCreateKey:     canCreate,
		Currency:         "USD",
		Subscription:     subSvc.BuildView(c.Request.Context(), user.ID),
	})
}

// PatchMe handles PATCH /dashboard/me
func (h *AuthHandler) PatchMe(c *gin.Context) {
	userID, _ := c.Get("user_id")
	var req model.UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.NewAPIError("invalid_request_error", "Invalid request"))
		return
	}
	user, err := h.store.GetUserByID(c.Request.Context(), userID.(string))
	if err != nil {
		c.JSON(http.StatusNotFound, model.NewAPIError("not_found", "User not found"))
		return
	}
	if strings.TrimSpace(req.Name) != "" {
		user.Name = strings.TrimSpace(req.Name)
	}
	user.UpdatedAt = time.Now()
	if err := h.store.UpdateUser(c.Request.Context(), user); err != nil {
		c.JSON(http.StatusInternalServerError, model.NewAPIError("internal_error", "Failed to update profile"))
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "ok", "name": user.Name})
}

// ChangePassword handles POST /dashboard/change-password
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	userID, _ := c.Get("user_id")
	var req model.ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.NewAPIError("invalid_request_error", "Invalid request"))
		return
	}
	user, err := h.store.GetUserByID(c.Request.Context(), userID.(string))
	if err != nil {
		c.JSON(http.StatusNotFound, model.NewAPIError("not_found", "User not found"))
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.CurrentPassword)); err != nil {
		c.JSON(http.StatusUnauthorized, model.NewAPIError("authentication_error", "Current password is incorrect"))
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.NewAPIError("internal_error", "Failed to hash password"))
		return
	}
	user.PasswordHash = string(hash)
	user.UpdatedAt = time.Now()
	if err := h.store.UpdateUser(c.Request.Context(), user); err != nil {
		c.JSON(http.StatusInternalServerError, model.NewAPIError("internal_error", "Failed to update password"))
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Password updated"})
}

// JWTAuthMiddleware validates JWT token
func (h *AuthHandler) JWTAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, model.NewAPIError("authentication_error", "Missing authorization header"))
			c.Abort()
			return
		}

		tokenString := middleware.StripBearerPrefix(authHeader)

		parser := jwt.NewParser(jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
		token, err := parser.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			return h.jwtSecret, nil
		})

		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, model.NewAPIError("authentication_error", "Invalid token"))
			c.Abort()
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.JSON(http.StatusUnauthorized, model.NewAPIError("authentication_error", "Invalid token claims"))
			c.Abort()
			return
		}

		userID, ok := claims["user_id"].(string)
		if !ok {
			c.JSON(http.StatusUnauthorized, model.NewAPIError("authentication_error", "Invalid user ID in token"))
			c.Abort()
			return
		}

		c.Set("user_id", userID)
		c.Next()
	}
}

func (h *AuthHandler) generateToken(user *model.User) (string, error) {
	claims := jwt.MapClaims{
		"user_id": user.ID,
		"email":   user.Email,
		"exp":     time.Now().Add(7 * 24 * time.Hour).Unix(),
		"iat":     time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(h.jwtSecret)
}

func generateSixDigitCode() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(900000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", int(n.Int64())+100000), nil
}

func (h *AuthHandler) sendResetPasswordEmail(toEmail, code string) error {
	if h.smtpHost == "" || h.smtpFrom == "" || h.smtpUsername == "" || h.smtpPassword == "" || h.smtpPort == 0 {
		return fmt.Errorf("SMTP is not fully configured")
	}

	subject := "Your Sub2API password reset code"
	body := fmt.Sprintf("Your verification code is: %s\r\n\r\nIt expires in 15 minutes. If you did not request a reset, ignore this email. Do not share this code.", code)
	msg := buildPlainEmailMessage(h.smtpFrom, toEmail, subject, body)

	addr := fmt.Sprintf("%s:%d", h.smtpHost, h.smtpPort)
	auth := smtp.PlainAuth("", h.smtpUsername, h.smtpPassword, h.smtpHost)

	if h.smtpPort == 465 {
		return sendMailImplicitTLS(addr, h.smtpHost, auth, h.smtpFrom, []string{toEmail}, msg)
	}
	return sendMailSTARTTLS(addr, h.smtpHost, auth, h.smtpFrom, []string{toEmail}, msg)
}

func (h *AuthHandler) sendRegisterCodeEmail(toEmail, code string) error {
	if h.smtpHost == "" || h.smtpFrom == "" || h.smtpUsername == "" || h.smtpPassword == "" || h.smtpPort == 0 {
		return fmt.Errorf("SMTP is not fully configured")
	}

	subject := "Your Sub2API registration code"
	body := fmt.Sprintf("Your verification code is: %s\r\n\r\nIt expires in 15 minutes. Do not share this code.", code)
	msg := buildPlainEmailMessage(h.smtpFrom, toEmail, subject, body)

	addr := fmt.Sprintf("%s:%d", h.smtpHost, h.smtpPort)
	auth := smtp.PlainAuth("", h.smtpUsername, h.smtpPassword, h.smtpHost)

	// smtp.SendMail only does plain TCP + optional STARTTLS; port 465 (SMTPS) needs TLS from first byte.
	if h.smtpPort == 465 {
		return sendMailImplicitTLS(addr, h.smtpHost, auth, h.smtpFrom, []string{toEmail}, msg)
	}
	return sendMailSTARTTLS(addr, h.smtpHost, auth, h.smtpFrom, []string{toEmail}, msg)
}

func buildPlainEmailMessage(from, to, subject, body string) []byte {
	var b strings.Builder
	b.WriteString("From: ")
	b.WriteString(from)
	b.WriteString("\r\nTo: ")
	b.WriteString(to)
	b.WriteString("\r\nSubject: ")
	b.WriteString(subject)
	b.WriteString("\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n")
	b.WriteString(body)
	return []byte(b.String())
}

func sendMailImplicitTLS(addr, host string, auth smtp.Auth, from string, to []string, msg []byte) error {
	tlsCfg := &tls.Config{ServerName: host, MinVersion: tls.VersionTLS12}
	conn, err := tls.Dial("tcp", addr, tlsCfg)
	if err != nil {
		return fmt.Errorf("tls dial: %w", err)
	}
	client, err := smtp.NewClient(conn, host)
	if err != nil {
		_ = conn.Close()
		return fmt.Errorf("smtp client: %w", err)
	}
	return smtpClientSend(client, auth, from, to, msg)
}

func sendMailSTARTTLS(addr, host string, auth smtp.Auth, from string, to []string, msg []byte) error {
	client, err := smtp.Dial(addr)
	if err != nil {
		return fmt.Errorf("smtp dial: %w", err)
	}

	if ok, _ := client.Extension("STARTTLS"); ok {
		tlsCfg := &tls.Config{ServerName: host, MinVersion: tls.VersionTLS12}
		if err := client.StartTLS(tlsCfg); err != nil {
			_ = client.Close()
			return fmt.Errorf("starttls: %w", err)
		}
	}

	return smtpClientSend(client, auth, from, to, msg)
}

// smtpClientSend runs MAIL/RCPT/DATA/QUIT. On error the client is closed.
func smtpClientSend(client *smtp.Client, auth smtp.Auth, from string, to []string, msg []byte) (err error) {
	defer func() {
		if err != nil {
			_ = client.Close()
		}
	}()

	if auth != nil {
		if ok, _ := client.Extension("AUTH"); ok {
			if err = client.Auth(auth); err != nil {
				return fmt.Errorf("smtp auth: %w", err)
			}
		}
	}

	if err = client.Mail(from); err != nil {
		return fmt.Errorf("smtp mail: %w", err)
	}
	for _, rcpt := range to {
		if err = client.Rcpt(rcpt); err != nil {
			return fmt.Errorf("smtp rcpt: %w", err)
		}
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp data: %w", err)
	}
	if _, err = w.Write(msg); err != nil {
		return err
	}
	if err = w.Close(); err != nil {
		return err
	}
	if err = client.Quit(); err != nil {
		return fmt.Errorf("smtp quit: %w", err)
	}
	return nil
}

func generateUserID(email string) string {
	h := sha256.Sum256([]byte(email + time.Now().String()))
	return "user_" + hex.EncodeToString(h[:8])
}

func validateTermsAcceptance(accepted bool, version string) string {
	if !accepted {
		return "You must accept the User Agreement and Privacy Notice to register"
	}
	want := legal.CurrentTermsVersion()
	got := strings.TrimSpace(version)
	if got == "" || got != want {
		return "Terms version is outdated. Please refresh the page and accept the current agreement (version " + want + ")"
	}
	return ""
}

// validateInviteCode returns a user-facing message when invite is required but missing or wrong.
func validateInviteCode(configured, provided string) string {
	want := strings.TrimSpace(configured)
	got := strings.TrimSpace(provided)
	if want == "" {
		return ""
	}
	if got == "" {
		return "请先填写邀请码"
	}
	if got != want {
		return "邀请码输入有误"
	}
	return ""
}
