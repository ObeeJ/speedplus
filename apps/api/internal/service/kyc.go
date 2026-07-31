package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/speedplus/api/internal/kyc"
	"github.com/speedplus/api/internal/model"
	"github.com/speedplus/api/internal/observability"
	"github.com/speedplus/api/internal/repo"
	"gorm.io/gorm"
)

type KYCService struct {
	repo     repo.DispatchRepo
	provider kyc.Provider
}

func NewKYCService(r repo.DispatchRepo, provider kyc.Provider) *KYCService {
	return &KYCService{repo: r, provider: provider}
}

func (s *KYCService) RecordDocumentUpload(ctx context.Context, userID uuid.UUID, docType model.KYCDocType, r2Key string) error {
	doc := model.KYCDocument{
		ID:         uuid.New(),
		UserID:     userID,
		DocType:    docType,
		R2Key:      r2Key,
		UploadedAt: time.Now(),
	}
	return s.repo.CreateKYCDocument(ctx, &doc)
}

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
			s.repo.CreateKYCCheck(ctx, &failedCheck)
		}
		return fmt.Errorf("prembly verify: %w", err)
	}

	status := model.KYCPending
	if result.Verified {
		status = model.KYCReview
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
	return s.repo.CreateKYCCheck(ctx, &check)
}

func (s *KYCService) AdminApprove(ctx context.Context, checkID, adminID uuid.UUID, note string) error {
	return s.repo.Transaction(ctx, func(tx *gorm.DB) error {
		check, err := s.repo.FindKYCCheckTx(ctx, tx, checkID)
		if err != nil {
			return err
		}
		check.Status = model.KYCApproved
		check.ReviewedBy = &adminID
		check.ReviewNote = &note
		if err := s.repo.SaveKYCCheckTx(ctx, tx, check); err != nil {
			return err
		}
		return s.tryActivateProfile(ctx, tx, check.UserID)
	})
}

func (s *KYCService) AdminReject(ctx context.Context, checkID, adminID uuid.UUID, note string) error {
	check, err := s.repo.FindKYCCheck(ctx, checkID)
	if err != nil {
		return err
	}
	check.Status = model.KYCRejected
	check.ReviewedBy = &adminID
	check.ReviewNote = &note
	return s.repo.SaveKYCCheck(ctx, check)
}

func (s *KYCService) tryActivateProfile(ctx context.Context, tx *gorm.DB, userID uuid.UUID) error {
	user, err := s.repo.FindUserTx(ctx, tx, userID)
	if err != nil {
		return err
	}

	switch user.Role {
	case model.RoleDriver:
		dp, err := s.repo.FindDriverProfileTx(ctx, tx, userID)
		if err != nil {
			return nil
		}
		dp.Status = model.DriverApproved
		return s.repo.SaveDriverProfileTx(ctx, tx, dp)
	case model.RoleMerchant:
		mp, err := s.repo.FindMerchantProfileTx(ctx, tx, userID)
		if err != nil {
			return nil
		}
		mp.Status = model.MerchantActive
		return s.repo.SaveMerchantProfileTx(ctx, tx, mp)
	}
	return nil
}

func (s *KYCService) QueueForAdmin(ctx context.Context, page, pageSize int) ([]model.KYCCheck, error) {
	return s.repo.ListPendingKYCChecks(ctx, page*pageSize, pageSize)
}
