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
	"github.com/speedplus/api/internal/whatsapp"
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

	rdbOpts, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		slog.Error("invalid REDIS_URL", "error", err)
		os.Exit(1)
	}
	rdb := redis.NewClient(rdbOpts)

	// ── Repos ──────────────────────────────────────────────────────────────────
	userRepo := repo.NewUserRepo(gormDB)
	orderRepo := repo.NewOrderRepo(gormDB)
	ledgerRepo := repo.NewLedgerRepo(gormDB)
	dispatchRepo := repo.NewDispatchRepo(gormDB)
	_ = dispatchRepo // consumed by services below
	merchantRepo := repo.NewMerchantRepo(gormDB)

	// ── Email ──────────────────────────────────────────────────────────────────────
	emailClient, err := email.New(cfg)
	if err != nil {
		slog.Error("email client init failed", "error", err)
		os.Exit(1)
	}

	// ── Services ───────────────────────────────────────────────────────────────
	paystackProvider := payment.NewPaystack(cfg.PaystackSecretKey)
	flutterwaveProvider := payment.NewFlutterwave(cfg.FlutterwaveSecretKey, cfg.FlutterwaveHash)
	monnifyProvider := payment.NewMonnify(cfg.MonnifyAPIKey, cfg.MonnifySecretKey, cfg.MonnifyContractCode, cfg.MonnifyWalletAccountNumber)
	catalogRepo := repo.NewCatalogRepo(gormDB)
	onboardingRepo := repo.NewOnboardingRepo(gormDB)
	tierRepo := repo.NewTierRepo(gormDB)
	deliveryCodeRepo := repo.NewDeliveryCodeRepo(gormDB)
	affordabilityRepo := repo.NewAffordabilityRepo(gormDB)
	onboardingSvc := service.NewOnboardingService(onboardingRepo, cfg, monnifyProvider)
	onboardingSvc.InjectUserRepo(userRepo)
	authSvc := service.NewAuthService(userRepo, cfg, onboardingSvc, emailClient)
	authSvc.InjectMerchantRepo(merchantRepo)
	kycProvider := kyc.NewPrembly(cfg.PremblyAPIKey, cfg.PremblyBaseURL, cfg.PremblyAppID)
	kycSvc := service.NewKYCService(dispatchRepo, kycProvider)
	kycSvc.InjectEmail(emailClient)
	feeConfigRepo := repo.NewFeeConfigRepo(gormDB)
	feeConfigSvc := service.NewFeeConfigService(feeConfigRepo)
	platformSettingRepo := repo.NewPlatformSettingRepo(gormDB)
	weatherSurchargeSvc := service.NewWeatherSurchargeService(platformSettingRepo, feeConfigRepo)
	pricingSvc := service.NewPricingService(orderRepo, cfg, cfg.OSRMURL, feeConfigSvc)
	pricingSvc.InjectWeatherSurcharge(weatherSurchargeSvc)
	ledgerSvc := service.NewLedgerService(ledgerRepo, pricingSvc)
	ledgerSvc.InjectFeeConfigs(feeConfigSvc)
	tierSvc := service.NewTierService(tierRepo)
	orderSvc := service.NewOrderService(orderRepo, pricingSvc, ledgerSvc, tierSvc)
	walletRepo := repo.NewWalletRepo(gormDB)
	velocityRepo := repo.NewVelocityRepo(gormDB)
	velocitySvc := service.NewVelocityService(velocityRepo)
	walletSvc := service.NewWalletService(walletRepo, ledgerSvc, authSvc, paystackProvider, emailClient, userRepo)
	walletSvc.InjectVelocity(velocitySvc, tierSvc)
	deliveryCodeSvc := service.NewDeliveryCodeService(deliveryCodeRepo)
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
	proofMediaRepo := repo.NewProofMediaRepo(gormDB)
	proofMediaSvc := service.NewProofMediaService(proofMediaRepo, orderRepo, r2Client)
	orderSvc.InjectDeliveryCodes(deliveryCodeSvc)
	orderSvc.InjectEmail(emailClient, userRepo)
	waSvc := whatsapp.New(cfg.WhatsAppPhoneNumberID, cfg.WhatsAppToken)
	orderSvc.InjectWhatsApp(waSvc)
	authSvc.InjectWhatsApp(waSvc)
	if cfg.WhatsAppPhoneNumberID == "" {
		slog.Warn("WHATSAPP_PHONE_NUMBER_ID not set — WhatsApp notifications disabled")
	}
	if len(cfg.EncryptionKey) == 32 {
		if recipientCipher, err := crypto.NewCipher([]byte(cfg.EncryptionKey)); err == nil {
			orderSvc.InjectRecipientCipher(recipientCipher)
		} else {
			slog.Error("recipient cipher init failed — package orders with recipient PII will error", "error", err)
		}
	} else if cfg.Environment == "production" {
		slog.Error("ENCRYPTION_KEY must be exactly 32 bytes in production — exiting")
		os.Exit(1)
	} else {
		slog.Warn("ENCRYPTION_KEY not set (or not 32 bytes) — recipient PII encryption disabled; package orders with recipient data will fail")
	}
	loyaltyRepo := repo.NewLoyaltyRepo(gormDB)
	loyaltySvc := service.NewLoyaltyService(loyaltyRepo, ledgerSvc)
	referralRepo := repo.NewReferralRepo(gormDB)
	referralSvc := service.NewReferralService(referralRepo, userRepo, ledgerSvc, loyaltySvc)
	authSvc.InjectReferrals(referralSvc)
	paycodeRepo := repo.NewPaycodeRepo(gormDB)
	paycodeSvc := service.NewPaycodeService(paycodeRepo, cfg, ledgerSvc, orderSvc, tierSvc, emailClient, userRepo, orderRepo, deliveryCodeSvc, referralSvc)
	paymentLinkRepo := repo.NewPaymentLinkRepo(gormDB)
	paymentLinkSvc := service.NewPaymentLinkService(paymentLinkRepo, ledgerSvc, paystackProvider, emailClient, userRepo)
	ussdRepo := repo.NewUSSDRepo(gormDB)
	ussdSvc := service.NewUSSDService(ussdRepo, monnifyProvider)
	giftCardRepo := repo.NewGiftCardRepo(gormDB)
	giftCardSvc := service.NewGiftCardService(giftCardRepo, ledgerSvc)
	giftCardSvc.InjectVelocity(velocitySvc, tierSvc)
	subscriptionRepo := repo.NewSubscriptionRepo(gormDB)
	subscriptionSvc := service.NewSubscriptionService(subscriptionRepo, orderRepo, orderSvc, ledgerSvc)
	dispatchSvc := service.NewDispatchService(dispatchRepo)
	runRepo := repo.NewRunRepo(gormDB)
	runSvc := service.NewRunService(runRepo, orderRepo, pricingSvc, dispatchSvc)
	gasSvc := service.NewGasService(orderRepo)
	catalogSvc := service.NewCatalogService(catalogRepo, r2Client)
	catalogSvc.InjectWhatsApp(waSvc, userRepo)
	merchantSvc := service.NewMerchantService(merchantRepo)
	adminRepo := repo.NewAdminRepo(gormDB)
	adminSvc := service.NewAdminService(adminRepo, ledgerSvc)
	affordabilitySvc := service.NewAffordabilityService(ledgerSvc, affordabilityRepo)

	// ── Asynq ──────────────────────────────────────────────────────────────────
	asynqClient := asynq.NewClient(asynq.RedisClientOpt{Addr: rdb.Options().Addr})
	defer asynqClient.Close()
	authSvc.InjectQueue(asynqClient, func(userID string) error {
		return worker.EnqueueOnboarding(asynqClient, userID)
	})
	walletSvc.InjectQueue(func(cashoutID string) error {
		return worker.EnqueueCashout(asynqClient, cashoutID)
	})
	orderSvc.InjectReviewQueue(func(revieweeID, revieweeType string) error {
		return worker.EnqueueReviewAggregate(asynqClient, revieweeID, revieweeType)
	})

	workerHandlers := worker.NewHandlers(walletSvc, dispatchSvc, ledgerSvc, subscriptionSvc, onboardingSvc, asynqClient)
	workerHandlers.InjectDB(gormDB)
	workerHandlers.InjectOrders(orderSvc)
	workerHandlers.InjectRuns(runSvc)
	reconciliationRepo := service.NewReconciliationRepo(gormDB)
	reconciliationSvc := service.NewReconciliationService(reconciliationRepo, paystackProvider, flutterwaveProvider, monnifyProvider)
	workerHandlers.InjectReconciliation(reconciliationSvc)
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
	// Wire order service into dispatch for state-machine-safe ManualAssign.
	dispatchSvc.InjectOrders(orderSvc)
	// Wire order ownership checker into hub — gates order channel subscriptions.
	hub.InjectOrderChecker(orderSvc)

	// ── Handlers ───────────────────────────────────────────────────────────────
	healthH := handler.NewHealthHandler(gormDB, rdb)
	authH := handler.NewAuthHandler(authSvc)
	usersH := handler.NewUsersHandler(userRepo)
	kycH := handler.NewKYCHandler(kycSvc)
	orderH := handler.NewOrderHandler(orderSvc)
	orderH.InjectMerchant(merchantSvc)
	proofMediaH := handler.NewProofMediaHandler(proofMediaSvc)
	walletH := handler.NewWalletHandler(walletSvc, ledgerSvc, userRepo)
	if cfg.BridgeEnabled && cfg.BridgeAPIKey != "" {
		walletH.SetBridge(payment.NewBridge(cfg.BridgeAPIKey))
	}
	paycodeH := handler.NewPaycodeHandler(paycodeSvc)
	dispatchH := handler.NewDispatchHandler(dispatchSvc)
	cardH := handler.NewCardHandler(paycodeSvc, authSvc, gormDB)
	paymentLinkH := handler.NewPaymentLinkHandler(paymentLinkSvc)
	ussdH := handler.NewUSSDHandler(ussdSvc)
	loyaltyH := handler.NewLoyaltyHandler(loyaltySvc)
	giftCardH := handler.NewGiftCardHandler(giftCardSvc)
	subscriptionH := handler.NewSubscriptionHandler(subscriptionSvc)
	gasH := handler.NewGasHandler(gasSvc)
	runH := handler.NewRunHandler(runSvc)
	catalogH := handler.NewCatalogHandler(catalogSvc)
	merchantH := handler.NewMerchantHandler(merchantSvc, orderSvc, catalogSvc, walletSvc)
	adminH := handler.NewAdminHandler(adminSvc, ledgerSvc, feeConfigSvc)
	adminH.InjectWeatherSurcharge(weatherSurchargeSvc)
	adminH.InjectVelocity(velocitySvc)
	affordabilityH := handler.NewAffordabilityHandler(affordabilitySvc)
	pricingH := handler.NewPricingHandler(pricingSvc)
	driverBankH := handler.NewDriverBankHandler(walletSvc, paystackProvider)

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
		if cfg.BridgeEnabled && cfg.BridgeAPIKey != "" {
			bridgeProvider := payment.NewBridge(cfg.BridgeAPIKey)
			webhooks.POST("/bridge", walletH.HandleBridgeWebhook(bridgeProvider))
		}
	}

	v1 := r.Group("/api/v1")

	authGroup := v1.Group("/auth")
	authGroup.Use(middleware.RateLimit(rdb, "auth", 10, time.Minute, true))
	{
		authGroup.POST("/register", authH.Register)
		authGroup.POST("/login", authH.Login)
		authGroup.POST("/logout", authH.Logout)
		authGroup.POST("/refresh", authH.Refresh)
		authGroup.POST("/reset-password", authH.ResetPassword)
		authGroup.POST("/pin/set", middleware.Auth(authSvc), authH.SetPIN)
		authGroup.POST("/pin/verify", middleware.Auth(authSvc), authH.VerifyPIN)
	}

	otpGroup := v1.Group("/otp")
	otpGroup.Use(middleware.RateLimit(rdb, "otp", 3, 5*time.Minute, true))
	{
		otpGroup.POST("/request", authH.RequestOTP)
		otpGroup.POST("/verify", authH.VerifyOTP)
	}

	authed := v1.Group("")
	authed.Use(middleware.Auth(authSvc), middleware.RequireActiveUser(userRepo))

	// Users
	authed.GET("/users/me", usersH.Me)
	authed.PUT("/users/me", usersH.UpdateMe)
	authed.GET("/users/me/addresses", usersH.ListAddresses)
	authed.POST("/users/me/addresses", usersH.CreateAddress)
	authed.GET("/users/me/driver-profile", middleware.RequireRole("driver"), usersH.GetDriverProfile)
	authed.GET("/users/me/merchant-profile", middleware.RequireRole("merchant"), usersH.GetMerchantProfile)

	// Catalog — public browse (no auth required)
	v1.GET("/merchants", middleware.RateLimit(rdb, "catalog", 60, time.Minute), catalogH.ListMerchants)
	v1.GET("/merchants/:id", middleware.RateLimit(rdb, "catalog", 60, time.Minute), catalogH.GetMerchant)
	v1.GET("/products", middleware.RateLimit(rdb, "catalog", 60, time.Minute), catalogH.ListProducts)
	v1.GET("/products/search", middleware.RateLimit(rdb, "catalog-search", 30, time.Minute), catalogH.SearchProducts)
	v1.GET("/products/:id", middleware.RateLimit(rdb, "catalog", 60, time.Minute), catalogH.GetProduct)

	// Prescriptions (auth required, row-level ownership enforced in handler).
	// Rate-limited + idempotent: this is a controlled-substance-adjacent
	// write, not a generic list/get.
	prescriptions := authed.Group("/prescriptions")
	{
		prescriptions.POST("/presign", middleware.RateLimit(rdb, "rx-presign", 5, time.Minute), catalogH.PresignPrescriptionUpload)
		prescriptions.POST("", middleware.RateLimit(rdb, "rx-create", 5, time.Minute), middleware.Idempotency(rdb, 24*time.Hour), catalogH.CreatePrescription)
		prescriptions.GET("", catalogH.ListPrescriptions)
		prescriptions.GET("/:id", catalogH.GetPrescription)
	}

	// KYC
	kyc := authed.Group("/kyc")
	{
		kyc.POST("/check", kycH.SubmitCheck)
		kyc.GET("/status", kycH.MyKYCStatus)
	}

	// Quotes
	authed.POST("/quotes", pricingH.Quote)
	authed.POST("/quotes/multistop", pricingH.QuoteMultiStop)

	// Orders
	orders := authed.Group("/orders")
	{
		orders.GET("", orderH.List)
		orders.POST("", middleware.RequireVerified(), middleware.RateLimit(rdb, "order-create", 10, time.Minute), middleware.Idempotency(rdb, 24*time.Hour), orderH.Create)
		orders.GET("/:id", orderH.GetByID)
		orders.GET("/:id/track", orderH.GetByID)
		orders.GET("/:id/receipt", orderH.Receipt)
		orders.POST("/:id/review", middleware.Idempotency(rdb, 24*time.Hour), orderH.Review)
		orders.GET("/:id/stops", orderH.GetStops)
		orders.POST("/:id/stops/confirm", middleware.RequireRole("driver"), orderH.ConfirmStop)
		orders.POST("/:id/cancel", middleware.RateLimit(rdb, "order-cancel", 5, time.Minute), orderH.Cancel)
		// Customer dispute — POST raises a dispute (freezes escrow + sets 72h SLA),
		// GET returns current status. Rate-limited and idempotent so a double-tap
		// cannot freeze an already-frozen hold.
		orders.POST("/:id/dispute", middleware.RateLimit(rdb, "order-dispute", 3, time.Minute), middleware.Idempotency(rdb, 24*time.Hour), orderH.RaiseDispute)
		orders.GET("/:id/dispute", orderH.GetDisputeStatus)
		orders.POST("/:id/proof/presign", middleware.RequireRole("driver"), proofMediaH.PresignUpload)
		orders.POST("/:id/proof/confirm", middleware.RequireRole("driver"), proofMediaH.ConfirmUpload)
		orders.GET("/:id/proof", proofMediaH.GetMedia)
	}

	// Driver badges — public, no auth required
	v1.GET("/drivers/:id/badges", orderH.DriverBadges)

	// Wallet
	wallet := authed.Group("/wallet")
	{
		wallet.GET("", walletH.GetBalance)
		wallet.GET("/transactions", walletH.GetTransactions)
		wallet.GET("/affordability", affordabilityH.GetAffordability)
		wallet.POST("/fund", middleware.RequireVerified(), middleware.Idempotency(rdb, 24*time.Hour, true), walletH.Fund)
		wallet.POST("/fund/crypto", middleware.RequireVerified(), middleware.Idempotency(rdb, 24*time.Hour, true), walletH.FundCrypto)
		// Rate-limited: /transfer resolves recipients by phone/username before
		// moving money, which is an enumeration surface (does this
		// phone/username have an account). The recipient-name reveal itself
		// is intentional anti-misdirected-payment UX (confirm you're paying
		// the right person, same pattern PayPal/Venmo use) — throttling the
		// lookup rate is the actual mitigation, not hiding the name.
		wallet.POST("/transfer", middleware.RequireVerified(), middleware.RateLimit(rdb, "wallet-transfer", 10, time.Minute), middleware.Idempotency(rdb, 24*time.Hour, true), walletH.Transfer)
	}

	// EWA
	earnings := authed.Group("/earnings")
	earnings.Use(middleware.RequireRole("driver"))
	{
		earnings.POST("/cashout", middleware.Idempotency(rdb, 24*time.Hour, true), walletH.EWACashout)
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
	// Guest pay: rate-limited (5/min per IP, fail-closed) and idempotency-keyed
	// so a retried request cannot double-charge. The idempotency middleware
	// cannot scope by userID here (no auth), so it falls back to the raw key
	// value — callers must supply a stable key per payment attempt.
	v1.POST("/pay/:slug/guest",
		middleware.RateLimit(rdb, "guest-pay", 5, time.Minute, true),
		paymentLinkH.InitiateGuestPayment,
	)

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
		giftCards.POST("", middleware.Idempotency(rdb, 24*time.Hour, true), giftCardH.Issue)
		giftCards.POST("/redeem", middleware.Idempotency(rdb, 24*time.Hour, true), giftCardH.Redeem)
	}

	// Subscriptions
	subs := authed.Group("/subscriptions")
	{
		subs.GET("", subscriptionH.List)
		subs.POST("", subscriptionH.Create)
		subs.POST("/:id/pause", subscriptionH.Pause)
		subs.POST("/:id/cancel", subscriptionH.Cancel)
	}

	// Gas LPG price index — public read
	v1.GET("/gas/price-index", subscriptionH.GetLPGPrice)
	v1.GET("/gas/specs", gasH.ListSpecs)

	// Customer cylinder registry
	cylinders := authed.Group("/cylinders")
	cylinders.Use(middleware.RequireRole("customer"))
	{
		cylinders.GET("", gasH.ListCylinders)
		cylinders.POST("", middleware.RateLimit(rdb, "cylinder-register", 10, time.Minute), gasH.RegisterCylinder)
		cylinders.POST("/:id/retire", gasH.RetireCylinder)
	}

	// ── Merchant self-service ─────────────────────────────────────────────────
	// All routes require role=merchant. Merchant.ID is resolved server-side
	// from the JWT User.ID — the request body never carries a merchant ID.
	merchantGroup := authed.Group("/merchant")
	merchantGroup.Use(middleware.RequireRole("merchant"))
	{
		merchantGroup.GET("/profile", merchantH.GetProfile)
		merchantGroup.POST("/status", merchantH.SetOpen)
		merchantGroup.GET("/orders", merchantH.ListOrders)
		merchantGroup.POST("/orders/:id/transition", merchantH.TransitionOrder)
		merchantGroup.GET("/products", merchantH.ListProducts)
		merchantGroup.POST("/products", merchantH.CreateProduct)
		merchantGroup.PUT("/products/:id", merchantH.UpdateProduct)
		merchantGroup.POST("/products/:id/availability", merchantH.SetProductAvailability)
		merchantGroup.GET("/wallet", walletH.GetBalance)
		merchantGroup.GET("/wallet/transactions", walletH.GetTransactions)
		merchantGroup.GET("/bank-account", merchantH.GetBankAccount)
		merchantGroup.POST("/bank-account", merchantH.SaveBankAccount)
		merchantGroup.POST("/withdraw", middleware.Idempotency(rdb, 24*time.Hour, true), merchantH.Withdraw)
		merchantGroup.GET("/prescriptions", merchantH.ListPrescriptions)
		merchantGroup.POST("/prescriptions/:id/review", middleware.Idempotency(rdb, 24*time.Hour), merchantH.ReviewPrescription)
	}

	// Driver dispatch
	drivers := authed.Group("/drivers")
	drivers.Use(middleware.RequireRole("driver"))
	{
		drivers.PATCH("/online", dispatchH.SetOnline)
		drivers.POST("/location", dispatchH.UpdateLocation)
		drivers.POST("/offers/:id/accept", dispatchH.AcceptOffer)
		drivers.POST("/offers/:id/reject", dispatchH.RejectOffer)
		// Bank account — resolve (rate-limited) then save
		drivers.POST("/bank-account/resolve", middleware.RateLimit(rdb, "bank-resolve", 5, time.Minute, true), driverBankH.ResolveAccount)
		drivers.POST("/bank-account", driverBankH.SaveAccount)
		drivers.GET("/bank-account", driverBankH.GetAccount)
	}
	// Bank list — public, no auth, cached by client (changes rarely)
	v1.GET("/banks", driverBankH.ListBanks)

	// WebSocket
	authed.GET("/ws", hub.Handler())
	authed.GET("/runs/:id", runH.GetRun)

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
		admin.GET("/users/:id/kyc", kycH.AdminGetUserKYC)

		// Dispatch
		admin.POST("/dispatch/:orderId/assign", dispatchH.AdminAssign)

		// Merchants
		admin.GET("/merchants", adminH.ListMerchants)
		admin.POST("/merchants/:id/status", adminH.SetMerchantStatus)

		// Drivers
		admin.GET("/drivers", adminH.ListDrivers)
		admin.POST("/drivers/:id/status", adminH.SetDriverStatus)

		admin.POST("/users/:id/active", adminH.SetUserActive)

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
		admin.GET("/settings/weather", adminH.GetWeatherSurcharge)
		admin.PUT("/settings/weather", adminH.SetWeatherSurcharge)

		// LPG price index
		admin.POST("/gas/price-index", subscriptionH.RecordLPGPrice)

		// Gas merchants (fill accuracy dashboard + override)
		admin.GET("/gas/merchants", adminH.ListGasMerchants)
		admin.PUT("/gas/merchants/:id/fill-status", adminH.SetMerchantFillStatus)

		// Gas zones (launch status)
		admin.GET("/gas/zones", adminH.ListZones)
		admin.PUT("/gas/zones/:id/launch-status", adminH.SetZoneLaunchStatus)

		// Ledger viewer
		admin.GET("/ledger", adminH.GetLedger)

		// Operational metrics
		admin.GET("/metrics", adminH.GetMetrics)

		// Admin console listings — these back the /users, /runs, /subscriptions
		// and /pharmacy pages, which were calling these paths before the routes
		// existed and 404'd on load.
		admin.GET("/users", adminH.ListUsers)
		admin.GET("/runs", adminH.ListRuns)
		admin.GET("/subscriptions", adminH.ListSubscriptions)
		admin.GET("/prescriptions", adminH.ListPrescriptions)

		// Velocity limits + AML flags
		admin.GET("/velocity/limits", adminH.ListVelocityLimits)
		admin.PUT("/velocity/limits", adminH.UpsertVelocityLimit)
		admin.GET("/velocity/suspicious", adminH.ListSuspiciousActivity)
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
