package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/gin-gonic/gin"
	"github.com/hibiken/asynq"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
	"github.com/speedplus/api/internal/config"
	"github.com/speedplus/api/internal/crypto"
	"github.com/speedplus/api/internal/db"
	"github.com/speedplus/api/internal/email"
	"github.com/speedplus/api/internal/handler"
	"github.com/speedplus/api/internal/kyc"
	"github.com/speedplus/api/internal/middleware"
	"github.com/speedplus/api/internal/payment"
	"github.com/speedplus/api/internal/repo"
	"github.com/speedplus/api/internal/service"
	"github.com/speedplus/api/internal/storage"
	"github.com/speedplus/api/internal/worker"
	"github.com/speedplus/api/internal/ws"
)

func main() {
	godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		slog.Error("config load failed", "error", err)
		os.Exit(1)
	}

	if cfg.SentryDSN != "" {
		sentry.Init(sentry.ClientOptions{
			Dsn:              cfg.SentryDSN,
			Environment:      cfg.Environment,
			TracesSampleRate: 0.1,
		})
		defer sentry.Flush(2 * time.Second)
	}

	if err := runMigrations(cfg.DatabaseURL); err != nil {
		slog.Error("migrations failed", "error", err)
		os.Exit(1)
	}

	gormDB, err := db.Connect(cfg.DatabaseURL, cfg.Environment == "development")
	if err != nil {
		slog.Error("db connect failed", "error", err)
		os.Exit(1)
	}

	rdbOpts, _ := redis.ParseURL(cfg.RedisURL)
	rdb := redis.NewClient(rdbOpts)

	// ── Repos ──────────────────────────────────────────────────────────────────
	userRepo := repo.NewUserRepo(gormDB)
	orderRepo := repo.NewOrderRepo(gormDB)
	ledgerRepo := repo.NewLedgerRepo(gormDB)
	dispatchRepo := repo.NewDispatchRepo(gormDB)
	_ = dispatchRepo // consumed by services below

	// ── Email ──────────────────────────────────────────────────────────────────────
	emailClient, err := email.New(cfg)
	if err != nil {
		slog.Error("email client init failed", "error", err)
		os.Exit(1)
	}

	// ── Services ───────────────────────────────────────────────────────────────
	paystackProvider := payment.NewPaystack(cfg.PaystackSecretKey)
	flutterwaveProvider := payment.NewFlutterwave(cfg.FlutterwaveSecretKey, cfg.FlutterwaveHash)
	monnifyProvider := payment.NewMonnify(cfg.MonnifyAPIKey, cfg.MonnifySecretKey, cfg.MonnifyContractCode)
	catalogRepo := repo.NewCatalogRepo(gormDB)
	onboardingRepo := repo.NewOnboardingRepo(gormDB)
	tierRepo := repo.NewTierRepo(gormDB)
	deliveryCodeRepo := repo.NewDeliveryCodeRepo(gormDB)
	affordabilityRepo := repo.NewAffordabilityRepo(gormDB)
	onboardingSvc := service.NewOnboardingService(onboardingRepo, cfg, monnifyProvider)
	onboardingSvc.InjectUserRepo(userRepo)
	authSvc := service.NewAuthService(userRepo, cfg, onboardingSvc, emailClient)
	kycProvider := kyc.NewPrembly(cfg.PremblyAPIKey, cfg.PremblyBaseURL, cfg.PremblyAppID)
	kycSvc := service.NewKYCService(gormDB, kycProvider)
	feeConfigSvc := service.NewFeeConfigService(gormDB)
	pricingSvc := service.NewPricingService(gormDB, cfg, os.Getenv("OSRM_URL"), feeConfigSvc)
	ledgerSvc := service.NewLedgerService(gormDB, ledgerRepo, pricingSvc)
	ledgerSvc.InjectFeeConfigs(feeConfigSvc)
	tierSvc := service.NewTierService(gormDB, tierRepo)
	orderSvc := service.NewOrderService(gormDB, pricingSvc, ledgerSvc, tierSvc)
	walletSvc := service.NewWalletService(gormDB, ledgerSvc, authSvc, paystackProvider, emailClient, userRepo)
	deliveryCodeSvc := service.NewDeliveryCodeService(gormDB, deliveryCodeRepo)
	// R2 is optional at boot — proof-media endpoints fail closed (clear error,
	// not silent no-op) rather than block the whole API from starting when
	// media storage isn't configured yet (e.g. early dev).
	var r2Client *storage.R2Client
	if r2c, err := storage.NewR2Client(context.Background(), storage.R2Config{
		AccountID: cfg.R2AccountID, AccessKeyID: cfg.R2AccessKeyID,
		SecretAccessKey: cfg.R2SecretAccessKey, BucketName: cfg.R2BucketName,
	}); err == nil {
		r2Client = r2c
	} else {
		slog.Warn("R2 storage not configured — proof-of-delivery media endpoints will error", "error", err)
	}
	proofMediaSvc := service.NewProofMediaService(gormDB, r2Client)
	orderSvc.InjectDeliveryCodes(deliveryCodeSvc)
	if len(cfg.EncryptionKey) == 32 {
		if recipientCipher, err := crypto.NewCipher([]byte(cfg.EncryptionKey)); err == nil {
			orderSvc.InjectRecipientCipher(recipientCipher)
		} else {
			slog.Error("recipient cipher init failed — package orders with recipient PII will error", "error", err)
		}
	} else if cfg.Environment != "production" {
		slog.Warn("ENCRYPTION_KEY not set (or not 32 bytes) — recipient PII encryption disabled; package orders with recipient data will fail")
	}
	loyaltySvc := service.NewLoyaltyService(gormDB)
	referralSvc := service.NewReferralService(gormDB, ledgerSvc, loyaltySvc)
	authSvc.InjectReferrals(referralSvc)
	paycodeSvc := service.NewPaycodeService(gormDB, cfg, ledgerSvc, orderSvc, tierSvc, emailClient, userRepo, orderRepo, deliveryCodeSvc, referralSvc)
	dispatchSvc := service.NewDispatchService(gormDB)
	paymentLinkSvc := service.NewPaymentLinkService(gormDB, ledgerSvc, paystackProvider, emailClient, userRepo)
	ussdSvc := service.NewUSSDService(gormDB, monnifyProvider)
	giftCardSvc := service.NewGiftCardService(gormDB, ledgerSvc)
	subscriptionSvc := service.NewSubscriptionService(gormDB, orderSvc, ledgerSvc)
	catalogSvc := service.NewCatalogService(catalogRepo)
	adminSvc := service.NewAdminService(gormDB, ledgerSvc)
	affordabilitySvc := service.NewAffordabilityService(ledgerSvc, affordabilityRepo)

	// ── Asynq ──────────────────────────────────────────────────────────────────
	asynqClient := asynq.NewClient(asynq.RedisClientOpt{Addr: rdb.Options().Addr})
	defer asynqClient.Close()
	authSvc.InjectQueue(asynqClient, func(userID string) error {
		return worker.EnqueueOnboarding(asynqClient, userID)
	})

	workerHandlers := worker.NewHandlers(walletSvc, dispatchSvc, ledgerSvc, subscriptionSvc, onboardingSvc)
	asynqServer := worker.NewServer(cfg.RedisURL)
	asynqMux := asynq.NewServeMux()
	workerHandlers.Register(asynqMux)
	go func() {
		if err := asynqServer.Run(asynqMux); err != nil {
			slog.Error("asynq server error", "error", err)
		}
	}()

	scheduler := worker.NewScheduler(cfg.RedisURL)
	go func() {
		if err := scheduler.Run(); err != nil {
			slog.Error("asynq scheduler error", "error", err)
		}
	}()

	// ── WS hub ─────────────────────────────────────────────────────────────────
	hub := ws.NewHub(rdb)
	hub.Start(context.Background())

	// Wire dispatch + hub into order service now that both are constructed.
	orderSvc.InjectDispatch(dispatchSvc, hub)

	// ── Handlers ───────────────────────────────────────────────────────────────
	healthH := handler.NewHealthHandler(gormDB, rdb)
	authH := handler.NewAuthHandler(authSvc)
	usersH := handler.NewUsersHandler(userRepo)
	kycH := handler.NewKYCHandler(kycSvc)
	orderH := handler.NewOrderHandler(orderSvc)
	proofMediaH := handler.NewProofMediaHandler(proofMediaSvc)
	walletH := handler.NewWalletHandler(walletSvc, ledgerSvc, userRepo)
	paycodeH := handler.NewPaycodeHandler(paycodeSvc)
	dispatchH := handler.NewDispatchHandler(dispatchSvc)
	cardH := handler.NewCardHandler(paycodeSvc, authSvc, gormDB)
	paymentLinkH := handler.NewPaymentLinkHandler(paymentLinkSvc)
	ussdH := handler.NewUSSDHandler(ussdSvc)
	loyaltyH := handler.NewLoyaltyHandler(loyaltySvc)
	giftCardH := handler.NewGiftCardHandler(giftCardSvc)
	subscriptionH := handler.NewSubscriptionHandler(subscriptionSvc)
	catalogH := handler.NewCatalogHandler(catalogSvc)
	adminH := handler.NewAdminHandler(adminSvc, ledgerSvc, feeConfigSvc)
	affordabilityH := handler.NewAffordabilityHandler(affordabilitySvc)
	pricingH := handler.NewPricingHandler(pricingSvc)

	// ── Router ─────────────────────────────────────────────────────────────────
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()

	// SetTrustedProxies MUST be called explicitly. Gin's default (no call)
	// trusts every proxy, meaning ClientIP() honors a client-supplied
	// X-Forwarded-For header verbatim — trivially spoofable, which would let
	// any caller rotate the header per-request to bypass the per-IP rate
	// limiters in middleware.RateLimit. TRUSTED_PROXIES is a comma-separated
	// CIDR list (set to your load balancer/ingress); unset means trust none,
	// so ClientIP() falls back to the raw TCP peer address.
	if proxies := os.Getenv("TRUSTED_PROXIES"); proxies != "" {
		r.SetTrustedProxies(strings.Split(proxies, ","))
	} else {
		r.SetTrustedProxies(nil)
	}

	r.Use(
		middleware.Recovery(),
		middleware.RequestID(),
		middleware.Logger(),
		middleware.CORS(middleware.AllowedOrigins(os.Getenv("ALLOWED_ORIGINS"))),
	)

	r.GET("/healthz", healthH.Healthz)
	r.GET("/readyz", healthH.Readyz)

	webhooks := r.Group("/webhooks")
	{
		webhooks.POST("/paystack", walletH.HandlePaystackWebhook(paystackProvider))
		webhooks.POST("/flutterwave", walletH.HandleFlutterwaveWebhook(flutterwaveProvider))
		webhooks.POST("/monnify", walletH.HandleMonnifyWebhook(monnifyProvider))
	}

	v1 := r.Group("/api/v1")

	authGroup := v1.Group("/auth")
	authGroup.Use(middleware.RateLimit(rdb, "auth", 10, time.Minute))
	{
		authGroup.POST("/register", authH.Register)
		authGroup.POST("/login", authH.Login)
		authGroup.POST("/logout", authH.Logout)
		authGroup.POST("/refresh", authH.Refresh)
		authGroup.POST("/pin/set", middleware.Auth(authSvc), authH.SetPIN)
		authGroup.POST("/pin/verify", middleware.Auth(authSvc), authH.VerifyPIN)
	}

	otpGroup := v1.Group("/otp")
	otpGroup.Use(middleware.RateLimit(rdb, "otp", 3, 5*time.Minute))
	{
		otpGroup.POST("/request", authH.RequestOTP)
		otpGroup.POST("/verify", authH.VerifyOTP)
	}

	authed := v1.Group("")
	authed.Use(middleware.Auth(authSvc))

	// Users
	authed.GET("/users/me", usersH.Me)
	authed.PUT("/users/me", usersH.UpdateMe)
	authed.GET("/users/me/addresses", usersH.ListAddresses)
	authed.POST("/users/me/addresses", usersH.CreateAddress)
	authed.GET("/users/me/driver-profile", middleware.RequireRole("driver"), usersH.GetDriverProfile)
	authed.GET("/users/me/merchant-profile", middleware.RequireRole("merchant"), usersH.GetMerchantProfile)

	// Catalog — public browse (no auth required)
	v1.GET("/merchants", catalogH.ListMerchants)
	v1.GET("/merchants/:id", catalogH.GetMerchant)
	v1.GET("/products", catalogH.ListProducts)
	v1.GET("/products/search", catalogH.SearchProducts)
	v1.GET("/products/:id", catalogH.GetProduct)

	// Prescriptions (auth required, row-level ownership enforced in handler)
	prescriptions := authed.Group("/prescriptions")
	{
		prescriptions.POST("", catalogH.CreatePrescription)
		prescriptions.GET("", catalogH.ListPrescriptions)
		prescriptions.GET("/:id", catalogH.GetPrescription)
	}

	// KYC
	kyc := authed.Group("/kyc")
	{
		kyc.POST("/check", kycH.SubmitCheck)
	}

	// Quotes
	authed.POST("/quotes", pricingH.Quote)
	authed.POST("/quotes/multistop", pricingH.QuoteMultiStop)

	// Orders
	orders := authed.Group("/orders")
	{
		orders.POST("", middleware.Idempotency(rdb, 24*time.Hour), orderH.Create)
		orders.GET("/:id", orderH.GetByID)
		orders.GET("/:id/track", orderH.GetByID)
		orders.GET("/:id/stops", orderH.GetStops)
		orders.POST("/:id/stops/confirm", middleware.RequireRole("driver"), orderH.ConfirmStop)
		orders.POST("/:id/cancel", orderH.Cancel)

		// Proof-of-delivery media (chain of custody). Presign/confirm are
		// driver-only (enforced again in the service via assigned-driver
		// check); viewing is limited to the order's customer or an admin.
		orders.POST("/:id/proof/presign", middleware.RequireRole("driver"), proofMediaH.PresignUpload)
		orders.POST("/:id/proof/confirm", middleware.RequireRole("driver"), proofMediaH.ConfirmUpload)
		orders.GET("/:id/proof", proofMediaH.GetMedia)
	}

	// Wallet
	wallet := authed.Group("/wallet")
	{
		wallet.GET("", walletH.GetBalance)
		wallet.GET("/transactions", walletH.GetTransactions)
		wallet.GET("/affordability", affordabilityH.GetAffordability)
		wallet.POST("/fund", middleware.Idempotency(rdb, 24*time.Hour), walletH.Fund)
		// Rate-limited: /transfer resolves recipients by phone/username before
		// moving money, which is an enumeration surface (does this
		// phone/username have an account). The recipient-name reveal itself
		// is intentional anti-misdirected-payment UX (confirm you're paying
		// the right person, same pattern PayPal/Venmo use) — throttling the
		// lookup rate is the actual mitigation, not hiding the name.
		wallet.POST("/transfer", middleware.RateLimit(rdb, "wallet-transfer", 10, time.Minute), middleware.Idempotency(rdb, 24*time.Hour), walletH.Transfer)
	}

	// EWA
	earnings := authed.Group("/earnings")
	earnings.Use(middleware.RequireRole("driver"))
	{
		earnings.POST("/cashout", middleware.Idempotency(rdb, 24*time.Hour), walletH.EWACashout)
	}

	// Paycodes
	paycodes := authed.Group("/paycodes")
	paycodes.Use(middleware.RateLimit(rdb, "paycodes", 20, time.Minute))
	{
		paycodes.POST("/generate", paycodeH.Generate)
		paycodes.POST("/resolve", middleware.RequireRole("driver"), paycodeH.Resolve)
		paycodes.POST("/confirm-code", middleware.RequireRole("driver"), paycodeH.ConfirmByCode)
		paycodes.POST("/:id/confirm", middleware.RequireRole("driver"), paycodeH.Confirm)
		paycodes.POST("/scan-card", middleware.RequireRole("driver"), cardH.ScanCard)
	}

	// User wallet + virtual account + trust tier + card
	authed.GET("/users/me/virtual-account", cardH.GetVirtualAccount)
	authed.GET("/users/me/trust-tier", cardH.GetTrustTier)
	authed.GET("/users/me/card", cardH.GetCard)

	// Payment links
	paymentLinks := authed.Group("/payment-links")
	{
		paymentLinks.POST("", paymentLinkH.Create)
		paymentLinks.POST("/:slug/pay", middleware.Idempotency(rdb, 24*time.Hour), paymentLinkH.PayByWallet)
	}
	// Public payment link endpoints (no auth)
	v1.GET("/pay/:slug", paymentLinkH.GetBySlug)
	v1.POST("/pay/:slug/guest", paymentLinkH.InitiateGuestPayment)

	// USSD wallet funding
	ussd := authed.Group("/wallet/ussd")
	{
		ussd.GET("/banks", ussdH.Banks)
		ussd.POST("/initiate", middleware.Idempotency(rdb, 30*time.Minute), ussdH.Initiate)
		ussd.GET("/intents/:id", ussdH.Status)
	}

		// Loyalty
	loyalty := authed.Group("/loyalty")
	{
		loyalty.GET("", loyaltyH.GetBalance)
		loyalty.GET("/history", loyaltyH.GetHistory)
	}

	// Gift cards
	giftCards := authed.Group("/gift-cards")
	{
		giftCards.POST("", giftCardH.Issue)
		giftCards.POST("/redeem", giftCardH.Redeem)
	}

	// Subscriptions
	subs := authed.Group("/subscriptions")
	{
		subs.POST("", subscriptionH.Create)
		subs.POST("/:id/pause", subscriptionH.Pause)
		subs.POST("/:id/cancel", subscriptionH.Cancel)
	}

	// Driver dispatch
	drivers := authed.Group("/drivers")
	drivers.Use(middleware.RequireRole("driver"))
	{
		drivers.POST("/location", dispatchH.UpdateLocation)
		drivers.POST("/offers/:id/accept", dispatchH.AcceptOffer)
		drivers.POST("/offers/:id/reject", dispatchH.RejectOffer)
	}

	// WebSocket
	authed.GET("/ws", hub.Handler())

	// Admin
	admin := authed.Group("/admin")
	// Rate-limited in addition to RequireRole: an admin token is a
	// high-value target (escrow release, driver/merchant suspension). A
	// compromised or leaked admin token should not be able to hammer
	// destructive endpoints at unlimited rate — this bounds the blast
	// radius even after auth is bypassed.
	admin.Use(middleware.RequireRole("admin"), middleware.RateLimit(rdb, "admin", 60, time.Minute))
	{
		// KYC
		admin.GET("/kyc/queue", kycH.AdminQueue)
		admin.POST("/kyc/:id/approve", kycH.Approve)
		admin.POST("/kyc/:id/reject", kycH.Reject)

		// Dispatch
		admin.POST("/dispatch/:orderId/assign", dispatchH.AdminAssign)

		// Merchants
		admin.GET("/merchants", adminH.ListMerchants)
		admin.POST("/merchants/:id/status", adminH.SetMerchantStatus)

		// Drivers
		admin.GET("/drivers", adminH.ListDrivers)
		admin.POST("/drivers/:id/status", adminH.SetDriverStatus)

		// Orders
		admin.GET("/orders", adminH.SearchOrders)
		admin.GET("/orders/:id", adminH.GetOrderDetail)

		// Disputes
		admin.POST("/disputes/:orderId/freeze", adminH.FreezeEscrow)
		admin.POST("/disputes/:orderId/release", adminH.ReleaseEscrow)

		// Cancellation rules
		admin.GET("/settings/cancellation-rules", adminH.ListCancellationRules)
		admin.PUT("/settings/cancellation-rules", adminH.UpsertCancellationRule)
		admin.DELETE("/settings/cancellation-rules/:id", adminH.DeleteCancellationRule)

		// Fee configs (pricing engine) — append-only, no DELETE
		admin.GET("/settings/fees", adminH.ListFeeConfigs)
		admin.PUT("/settings/fees", adminH.UpsertFeeConfig)

		// Ledger viewer
		admin.GET("/ledger", adminH.GetLedger)
	}

	// ── Server ─────────────────────────────────────────────────────────────────
	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		slog.Info("server starting", "port", cfg.Port, "env", cfg.Environment)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	asynqServer.Shutdown()
	srv.Shutdown(ctx)
	slog.Info("server stopped")
}
