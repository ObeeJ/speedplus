package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/speedplus/api/internal/card"
	"github.com/speedplus/api/internal/config"
	"github.com/speedplus/api/internal/model"
	"github.com/speedplus/api/internal/payment"
	"github.com/speedplus/api/internal/repo"
)

// OnboardingService runs after a user is created:
//  1. Creates a Monnify dedicated virtual account (DVA)
//  2. Generates the static Fourdat card QR
//  3. Seeds the user_trust_tiers row at Tier 0
//
// All three steps are best-effort — a failure does NOT roll back registration.
type OnboardingService struct {
	repo     repo.OnboardingRepo
	userRepo repo.UserRepo
	cfg      *config.Config
	dva      payment.DVAProvider
}

func NewOnboardingService(r repo.OnboardingRepo, cfg *config.Config, dva payment.DVAProvider) *OnboardingService {
	return &OnboardingService{repo: r, cfg: cfg, dva: dva}
}

// InjectUserRepo wires the UserRepo dependency needed by RunByID.
// Called once from main.go after both repos are constructed.
func (s *OnboardingService) InjectUserRepo(ur repo.UserRepo) {
	s.userRepo = ur
}

// Run satisfies ports.OnboardingRunner.
func (s *OnboardingService) Run(ctx context.Context, user *model.User) error {
	var errs []string

	if err := s.createDVA(ctx, user); err != nil {
		errs = append(errs, fmt.Sprintf("dva: %v", err))
	}
	if err := s.createCard(ctx, user); err != nil {
		errs = append(errs, fmt.Sprintf("card: %v", err))
	}
	if err := s.seedTrustTier(ctx, user.ID); err != nil {
		errs = append(errs, fmt.Sprintf("trust_tier: %v", err))
	}

	if len(errs) > 0 {
		return fmt.Errorf("onboarding partial failure: %s", strings.Join(errs, "; "))
	}
	return nil
}

// RunByID satisfies the worker's onboardingRunner interface.
// Looks up the user by ID and delegates to Run.
func (s *OnboardingService) RunByID(ctx context.Context, userID string) error {
	if s.userRepo == nil {
		return fmt.Errorf("onboarding: userRepo not injected")
	}
	id, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("onboarding: invalid user id %s: %w", userID, err)
	}
	user, err := s.userRepo.FindByID(ctx, id)
	if err != nil {
		return fmt.Errorf("onboarding: user not found %s: %w", userID, err)
	}
	return s.Run(ctx, user)
}

func (s *OnboardingService) createDVA(ctx context.Context, user *model.User) error {
	if _, err := s.repo.FindVirtualAccount(ctx, user.ID); err == nil {
		return nil // already exists — idempotent
	}

	email := ""
	if user.Email != nil {
		email = *user.Email
	}

	resp, err := s.dva.CreateReservedAccount(ctx, payment.DVARequest{
		UserID:   user.ID.String(),
		FullName: user.FirstName + " " + user.LastName,
		Email:    email,
	})
	if err != nil {
		return err
	}

	return s.repo.CreateVirtualAccount(ctx, &model.VirtualAccount{
		ID:            uuid.New(),
		UserID:        user.ID,
		AccountNumber: resp.AccountNumber,
		BankName:      resp.BankName,
		BankCode:      resp.BankCode,
		Provider:      "monnify",
		ProviderRef:   resp.ProviderRef,
	})
}

func (s *OnboardingService) createCard(ctx context.Context, user *model.User) error {
	if _, err := s.repo.FindUserCard(ctx, user.ID); err == nil {
		return nil // already exists — idempotent
	}

	return s.repo.CreateUserCard(ctx, &model.UserCard{
		ID:      uuid.New(),
		UserID:  user.ID,
		Payload: card.BuildPayload(user.ID, s.cfg.PaycodeSecret),
	})
}

func (s *OnboardingService) seedTrustTier(ctx context.Context, userID uuid.UUID) error {
	_, err := s.repo.FindOrCreateTrustTier(ctx, userID)
	return err
}
