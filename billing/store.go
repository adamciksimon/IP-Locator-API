package billing

import (
	"context"
	"database/sql"
	"errors"
	"time"

	billingmodels "github.com/adamciksimon/public-api/billing/models"
	"github.com/uptrace/bun"
)

var ErrInvalidKey = errors.New("invalid api key")
var ErrNotFound = errors.New("not found")

type PgStore struct {
	db *bun.DB
}

func NewStore(db *bun.DB) *PgStore {
	return &PgStore{db: db}
}

func (s *PgStore) ValidateKey(key string) error {
	var ak billingmodels.APIKey
	err := s.db.NewSelect().
		Model(&ak).
		Where("ak.key = ? AND ak.active = true", key).
		Scan(context.Background())

	if errors.Is(err, sql.ErrNoRows) {
		return ErrInvalidKey
	}
	return err
}

func (s *PgStore) GetPlanByUID(ctx context.Context, uid string) (*billingmodels.Plan, error) {
	var plan billingmodels.Plan
	err := s.db.NewSelect().Model(&plan).Where("p.uid = ?", uid).Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &plan, err
}

func (s *PgStore) FindOrCreateCustomer(ctx context.Context, email string) (*billingmodels.Customer, error) {
	customer := &billingmodels.Customer{Email: email}
	_, err := s.db.NewInsert().
		Model(customer).
		On("CONFLICT (email) DO UPDATE").
		Set("updated_at = current_timestamp").
		Returning("*").
		Exec(ctx)
	return customer, err
}

func (s *PgStore) UpsertBillingProfile(ctx context.Context, customerID, stripeCustomerID string) error {
	profile := &billingmodels.BillingProfile{
		CustomerID: customerID,
		ExternalID: stripeCustomerID,
	}
	_, err := s.db.NewInsert().
		Model(profile).
		On("CONFLICT (customer_id) DO UPDATE").
		Set("external_id = EXCLUDED.external_id").
		Set("updated_at = current_timestamp").
		Exec(ctx)
	return err
}

func (s *PgStore) CreateSubscription(ctx context.Context, sub *billingmodels.Subscription) error {
	_, err := s.db.NewInsert().Model(sub).Exec(ctx)
	return err
}

func (s *PgStore) UpdateSubscriptionByExternalID(ctx context.Context, externalID string, status billingmodels.SubscriptionStatus, periodStart, periodEnd time.Time) error {
	_, err := s.db.NewUpdate().
		TableExpr("subscriptions").
		Set("status = ?", status).
		Set("period_start = ?", periodStart).
		Set("period_end = ?", periodEnd).
		Set("updated_at = current_timestamp").
		Where("external_id = ?", externalID).
		Exec(ctx)
	return err
}

func (s *PgStore) GetAllPlans(ctx context.Context) ([]*billingmodels.Plan, error) {
	var plans []*billingmodels.Plan
	err := s.db.NewSelect().Model(&plans).Scan(ctx)
	return plans, err
}

func (s *PgStore) UpdatePlanExternalID(ctx context.Context, id, externalID string) error {
	_, err := s.db.NewUpdate().
		Model((*billingmodels.Plan)(nil)).
		Set("external_id = ?", externalID).
		Where("id = ?", id).
		Exec(ctx)
	return err
}

func (s *PgStore) GetBillingProfileByCustomerID(ctx context.Context, customerID string) (*billingmodels.BillingProfile, error) {
	var profile billingmodels.BillingProfile
	err := s.db.NewSelect().Model(&profile).Where("bp.customer_id = ?", customerID).Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &profile, err
}

func (s *PgStore) GetSubscriptionByCustomerID(ctx context.Context, customerID string) (*billingmodels.Subscription, error) {
	var sub billingmodels.Subscription
	err := s.db.NewSelect().Model(&sub).
		Where("s.customer_id = ?", customerID).
		Where("s.status IN (?)", bun.In([]string{
			string(billingmodels.SubscriptionActive),
			string(billingmodels.SubscriptionPastDue),
		})).
		OrderExpr("s.created_at DESC").
		Limit(1).
		Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &sub, err
}

func (s *PgStore) GetSubscriptionByExternalID(ctx context.Context, externalID string) (*billingmodels.Subscription, error) {
	var sub billingmodels.Subscription
	err := s.db.NewSelect().Model(&sub).Where("s.external_id = ?", externalID).Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &sub, err
}

func (s *PgStore) UpsertSubscriptionByCustomerID(ctx context.Context, sub *billingmodels.Subscription) error {
	var existing billingmodels.Subscription
	err := s.db.NewSelect().Model(&existing).
		Where("s.customer_id = ?", sub.CustomerID).
		OrderExpr("s.created_at DESC").
		Limit(1).
		Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		_, err = s.db.NewInsert().Model(sub).Exec(ctx)
		return err
	}
	if err != nil {
		return err
	}
	_, err = s.db.NewUpdate().
		TableExpr("subscriptions").
		Set("plan_id = ?", sub.PlanID).
		Set("external_id = ?", sub.ExternalID).
		Set("status = ?", sub.Status).
		Set("period_start = ?", sub.PeriodStart).
		Set("period_end = ?", sub.PeriodEnd).
		Set("updated_at = current_timestamp").
		Where("id = ?", existing.ID).
		Exec(ctx)
	return err
}

func (s *PgStore) UpdateSubscription(ctx context.Context, sub *billingmodels.Subscription) error {
	_, err := s.db.NewUpdate().
		TableExpr("subscriptions").
		Set("plan_id = ?", sub.PlanID).
		Set("period_start = ?", sub.PeriodStart).
		Set("period_end = ?", sub.PeriodEnd).
		Set("updated_at = current_timestamp").
		Where("id = ?", sub.ID).
		Exec(ctx)
	return err
}

func (s *PgStore) UpdateSubscriptionStatus(ctx context.Context, id string, status billingmodels.SubscriptionStatus) error {
	_, err := s.db.NewUpdate().
		TableExpr("subscriptions").
		Set("status = ?", status).
		Set("updated_at = current_timestamp").
		Where("id = ?", id).
		Exec(ctx)
	return err
}

func (s *PgStore) GetBillingProfileByEmail(ctx context.Context, email string) (*billingmodels.BillingProfile, error) {
	var profile billingmodels.BillingProfile
	err := s.db.NewSelect().
		Model(&profile).
		Relation("Customer").
		Where("c.email = ?", email).
		Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &profile, err
}
