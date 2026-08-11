package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/speedplus/api/internal/middleware"
	"github.com/speedplus/api/internal/payment"
	"github.com/speedplus/api/internal/service"
)

// DriverBankHandler handles driver bank account management.
// All routes require role=driver.
type DriverBankHandler struct {
	wallet   *service.WalletService
	paystack *payment.PaystackProvider
}

func NewDriverBankHandler(wallet *service.WalletService, paystack *payment.PaystackProvider) *DriverBankHandler {
	return &DriverBankHandler{wallet: wallet, paystack: paystack}
}

// ListBanks — GET /drivers/banks (no auth required — public list)
func (h *DriverBankHandler) ListBanks(c *gin.Context) {
	banks, err := h.paystack.ListBanks(c.Request.Context())
	if err != nil {
		internalError(c, err)
		return
	}
	type bankRow struct {
		Code string `json:"code"`
		Name string `json:"name"`
	}
	rows := make([]bankRow, len(banks))
	for i, b := range banks {
		rows[i] = bankRow{Code: b.Code, Name: b.Name}
	}
	c.JSON(http.StatusOK, successResp(gin.H{"banks": rows}))
}

// ResolveAccount — POST /drivers/bank-account/resolve
func (h *DriverBankHandler) ResolveAccount(c *gin.Context) {
	var req struct {
		BankCode      string `json:"bankCode"      binding:"required"`
		AccountNumber string `json:"accountNumber" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		validationError(c, err)
		return
	}
	name, err := h.wallet.ResolveDriverBankAccount(c.Request.Context(), req.BankCode, req.AccountNumber)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, errResp("VALIDATION_ERROR", "Could not verify account — check the account number and bank code", "accountNumber"))
		return
	}
	c.JSON(http.StatusOK, successResp(gin.H{"accountName": name}))
}

// SaveAccount — POST /drivers/bank-account
// Always resolves server-side — client-supplied account names are ignored.
func (h *DriverBankHandler) SaveAccount(c *gin.Context) {
	var req struct {
		BankCode      string `json:"bankCode"      binding:"required"`
		BankName      string `json:"bankName"      binding:"required"`
		AccountNumber string `json:"accountNumber" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		validationError(c, err)
		return
	}
	driverID, _ := uuid.Parse(c.GetString(middleware.CtxUserID))

	resolvedName, err := h.wallet.ResolveDriverBankAccount(c.Request.Context(), req.BankCode, req.AccountNumber)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, errResp("VALIDATION_ERROR", "Could not verify account — check the account number and bank code", "accountNumber"))
		return
	}
	acct, err := h.wallet.SaveDriverBankAccount(c.Request.Context(), driverID, req.BankCode, req.BankName, req.AccountNumber, resolvedName)
	if err != nil {
		internalError(c, err)
		return
	}
	c.JSON(http.StatusOK, successResp(gin.H{
		"bankCode":      acct.BankCode,
		"bankName":      acct.BankName,
		"accountNumber": maskAccount(acct.AccountNumber),
		"accountName":   acct.AccountName,
	}))
}

// GetAccount — GET /drivers/bank-account
func (h *DriverBankHandler) GetAccount(c *gin.Context) {
	driverID, _ := uuid.Parse(c.GetString(middleware.CtxUserID))
	acct, err := h.wallet.GetDriverBankAccount(c.Request.Context(), driverID)
	if err != nil {
		internalError(c, err)
		return
	}
	if acct == nil {
		c.JSON(http.StatusOK, successResp(nil))
		return
	}
	c.JSON(http.StatusOK, successResp(gin.H{
		"bankCode":      acct.BankCode,
		"bankName":      acct.BankName,
		"accountNumber": maskAccount(acct.AccountNumber),
		"accountName":   acct.AccountName,
	}))
}

// maskAccount returns ****XXXX where XXXX is the last 4 digits.
func maskAccount(n string) string {
	if len(n) <= 4 {
		return "****"
	}
	return "****" + n[len(n)-4:]
}
