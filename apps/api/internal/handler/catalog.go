package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/speedplus/api/internal/dto"
	"github.com/speedplus/api/internal/middleware"
	"github.com/speedplus/api/internal/service"
)

type CatalogHandler struct {
	catalog *service.CatalogService
}

func NewCatalogHandler(catalog *service.CatalogService) *CatalogHandler {
	return &CatalogHandler{catalog: catalog}
}

// ── Merchants ─────────────────────────────────────────────────────────────────

func (h *CatalogHandler) ListMerchants(c *gin.Context) {
	vertical := c.Query("vertical")
	page := queryInt(c, "page", 0)
	merchants, err := h.catalog.ListMerchants(c.Request.Context(), vertical, page, 20)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.Fail("INTERNAL_ERROR", "An unexpected error occurred", ""))
		return
	}
	c.Header("Cache-Control", "public, max-age=60")
	c.JSON(http.StatusOK, dto.OK(gin.H{"merchants": merchants}))
}

func (h *CatalogHandler) GetMerchant(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.Fail("VALIDATION_ERROR", "Invalid merchant ID", "id"))
		return
	}
	merchant, err := h.catalog.GetMerchant(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, dto.Fail("NOT_FOUND", "Merchant not found", ""))
		return
	}
	c.Header("Cache-Control", "public, max-age=60")
	c.JSON(http.StatusOK, dto.OK(merchant))
}

// ── Products ──────────────────────────────────────────────────────────────────

func (h *CatalogHandler) ListProducts(c *gin.Context) {
	merchantID, err := uuid.Parse(c.Query("merchantId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.Fail("VALIDATION_ERROR", "merchantId query param required", "merchantId"))
		return
	}
	category := c.Query("category")
	page := queryInt(c, "page", 0)
	products, err := h.catalog.ListProducts(c.Request.Context(), merchantID, category, page, 40)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.Fail("INTERNAL_ERROR", "An unexpected error occurred", ""))
		return
	}
	c.Header("Cache-Control", "public, max-age=60")
	c.JSON(http.StatusOK, dto.OK(gin.H{"products": products}))
}

func (h *CatalogHandler) GetProduct(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.Fail("VALIDATION_ERROR", "Invalid product ID", "id"))
		return
	}
	product, err := h.catalog.GetProduct(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, dto.Fail("NOT_FOUND", "Product not found", ""))
		return
	}
	c.Header("Cache-Control", "public, max-age=60")
	c.JSON(http.StatusOK, dto.OK(product))
}

func (h *CatalogHandler) SearchProducts(c *gin.Context) {
	q := c.Query("q")
	if q == "" {
		c.JSON(http.StatusBadRequest, dto.Fail("VALIDATION_ERROR", "q query param required", "q"))
		return
	}
	vertical := c.Query("vertical")
	products, err := h.catalog.SearchProducts(c.Request.Context(), q, vertical)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.Fail("INTERNAL_ERROR", "An unexpected error occurred", ""))
		return
	}
	c.JSON(http.StatusOK, dto.OK(gin.H{"products": products}))
}

// ── Prescriptions ─────────────────────────────────────────────────────────────

// CreatePrescription records a prescription after the frontend has uploaded
// the image directly to R2 and obtained the object key.
func (h *CatalogHandler) CreatePrescription(c *gin.Context) {
	var req struct {
		R2Key      string  `json:"r2Key"      binding:"required"`
		MerchantID *string `json:"merchantId"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.Fail("VALIDATION_ERROR", err.Error(), ""))
		return
	}

	customerID, _ := uuid.Parse(c.GetString(middleware.CtxUserID))

	var merchantID *uuid.UUID
	if req.MerchantID != nil {
		id, err := uuid.Parse(*req.MerchantID)
		if err != nil {
			c.JSON(http.StatusBadRequest, dto.Fail("VALIDATION_ERROR", "Invalid merchantId", "merchantId"))
			return
		}
		merchantID = &id
	}

	prescription, err := h.catalog.CreatePrescription(c.Request.Context(), customerID, req.R2Key, merchantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.Fail("INTERNAL_ERROR", "An unexpected error occurred", ""))
		return
	}
	c.JSON(http.StatusCreated, dto.OK(prescription))
}

func (h *CatalogHandler) GetPrescription(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.Fail("VALIDATION_ERROR", "Invalid prescription ID", "id"))
		return
	}
	customerID, _ := uuid.Parse(c.GetString(middleware.CtxUserID))
	prescription, err := h.catalog.GetPrescription(c.Request.Context(), id, customerID)
	if err != nil {
		c.JSON(http.StatusNotFound, dto.Fail("NOT_FOUND", "Prescription not found", ""))
		return
	}
	c.JSON(http.StatusOK, dto.OK(prescription))
}

func (h *CatalogHandler) ListPrescriptions(c *gin.Context) {
	customerID, _ := uuid.Parse(c.GetString(middleware.CtxUserID))
	prescriptions, err := h.catalog.ListPrescriptions(c.Request.Context(), customerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.Fail("INTERNAL_ERROR", "An unexpected error occurred", ""))
		return
	}
	c.JSON(http.StatusOK, dto.OK(gin.H{"prescriptions": prescriptions}))
}


