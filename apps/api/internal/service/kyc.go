package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/speedplus/api/internal/kyc"
	"github.com/speedplus/api/internal/model"
	"github.com/speedplus/api/internal/observability"
	"gorm.io/gorm"
)

type KYCService struct {
	db       *gorm.DB
	provider kyc.Provider
}

func NewKYCService(db *gorm.DB, provider kyc.Provider) *KYCService {
	return &KYCService{db: db, provider: provider}
}

// PresignUploadURL returns a presigned R2 URL for the client to PUT a KYC doc.
// The actual R2 signing is handled by the storage service; this records the intent.
func (s *KYCService) RecordDocumentUpload(ctx context.Context, userID uuid.UUID, docType model.KYCDocType, r2Key string) error {
	doc := model.KYCDocument{
		ID:         uuid.New(),
		UserID:     userID,
		DocType:    docType,
		R2Key:      r2Key,
		UploadedAt: time.Now(),
	}
	return s.db.WithContext(ctx).Create(&doc).Error
}

// SubmitCheck triggers a Prembly verification and records the result.
func (s *KYCService) SubmitCheck(ctx context.Context, userID uuid.UUID, docType model.KYCDocType, params map[string]string) error {
	var result *kyc.Result
	var err error

	switch docType {
	case model.DocNIN:
		result, err = s.provider.VerifyNIN(ctx, params["number"], params["first_name"], params["last_name"], params["dob"])
	case model.DocBVN:
		result, err = s.provider.VerifyBVN(ctx, params["number"], params["first_name"], params["last_name"], params["dob"])
	case model.DocDriversLicence:
		result, err = s.provider.VerifyDriversLicence(ctx, params["number"], params["first_name"], params["last_name"], params["dob"])
	case model.DocLiveness:
		result, err = s.provider.Liveness(ctx, params["image"])
	default:
		return fmt.Errorf("unsupported doc type: %s", docType)
	}
	// A provider-level failure (no credit, bad input, outage) still returns a
	// non-nil result carrying the raw payload/message — record it as a failed
	// check rather than silently dropping the attempt, so admins can see WHY
	// (as opposed to an identity-mismatch, which is a legitimate "not verified").
	if err != nil {
		observability.CaptureError(ctx, err, "kyc: prembly verify failed",
			"user_id", userID.String(), "doc_type", string(docType),
			"provider_message", result.Message)

		if result != nil {
			raw := string(result.RawPayload)
			note := "Provider request failed: " + result.Message
			failedCheck := model.KYCCheck{
				ID:              uuid.New(),
				UserID:          userID,
				DocType:         docType,
				Status:          model.KYCPending,
				ProviderPayload: &raw,
				ReviewNote:      &note,
			}
			s.db.WithContext(ctx).Create(&failedCheck)
		}
		return fmt.Errorf("prembly verify: %w", err)
	}

	status := model.KYCPending
	if result.Verified {
		status = model.KYCReview // auto-verified still needs admin sign-off
	}

	raw := string(result.RawPayload)
	check := model.KYCCheck{
		ID:              uuid.New(),
		UserID:          userID,
		DocType:         docType,
		Status:          status,
		ProviderRef:     &result.ProviderRef,
		ProviderPayload: &raw,
	}
	return s.db.WithContext(ctx).Create(&check).Error
}

// AdminApprove marks a KYC check approved and, if all required checks pass,
// activates the driver/merchant profile.
func (s *KYCService) AdminApprove(ctx context.Context, checkID, adminID uuid.UUID, note string) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var check model.KYCCheck
		if err := tx.First(&check, checkID).Error; err != nil {
			return err
		}
		check.Status = model.KYCApproved
		check.ReviewedBy = &adminID
		check.ReviewNote = &note
		if err := tx.Save(&check).Error; err != nil {
			return err
		}

		// Activate profile if all required checks for this user are approved
		return s.tryActivateProfile(ctx, tx, check.UserID)
	})
}

func (s *KYCService) AdminReject(ctx context.Context, checkID, adminID uuid.UUID, note string) error {
	var check model.KYCCheck
	if err := s.db.WithContext(ctx).First(&check, checkID).Error; err != nil {
		return err
	}
	check.Status = model.KYCRejected
	check.ReviewedBy = &adminID
	check.ReviewNote = &note
	return s.db.WithContext(ctx).Save(&check).Error
}

func (s *KYCService) tryActivateProfile(ctx context.Context, tx *gorm.DB, userID uuid.UUID) error {
	var user model.User
	if err := tx.First(&user, userID).Error; err != nil {
		return err
	}

	switch user.Role {
	case model.RoleDriver:
		var dp model.DriverProfile
		if err := tx.Where("user_id = ?", userID).First(&dp).Error; err != nil {
			return nil
		}
		dp.Status = model.DriverApproved
		return tx.Save(&dp).Error
	case model.RoleMerchant:
		var mp model.MerchantProfile
		if err := tx.Where("user_id = ?", userID).First(&mp).Error; err != nil {
			return nil
		}
		mp.Status = model.MerchantActive
		return tx.Save(&mp).Error
	}
	return nil
}

// QueueForAdmin returns all KYC checks awaiting admin review.
func (s *KYCService) QueueForAdmin(ctx context.Context, page, pageSize int) ([]model.KYCCheck, error) {
	var checks []model.KYCCheck
	err := s.db.WithContext(ctx).
		Where("status IN ?", []model.KYCStatus{model.KYCPending, model.KYCReview}).
		Order("created_at ASC").
		Limit(pageSize).Offset(page * pageSize).
		Find(&checks).Error
	return checks, err
}
