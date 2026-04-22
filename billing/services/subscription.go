package services

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"time"

	"github.com/adamciksimon/public-api/billing"
	billingmodels "github.com/adamciksimon/public-api/billing/models"
	"github.com/shopspring/decimal"
	stripe "github.com/stripe/stripe-go/v82"
	stripeSession "github.com/stripe/stripe-go/v82/checkout/session"
	stripeCustomer "github.com/stripe/stripe-go/v82/customer"
	stripeProduct "github.com/stripe/stripe-go/v82/product"
	stripeSub "github.com/stripe/stripe-go/v82/subscription"
	stripePortal "github.com/stripe/stripe-go/v82/billingportal/session"
)

var stripeStatusMap = map[stripe.SubscriptionStatus]billingmodels.SubscriptionStatus{
	stripe.SubscriptionStatusActive:   billingmodels.SubscriptionActive,
	stripe.SubscriptionStatusPastDue:  billingmodels.SubscriptionPastDue,
	stripe.SubscriptionStatusCanceled: billingmodels.SubscriptionCanceled,
	stripe.SubscriptionStatusTrialing: billingmodels.SubscriptionActive,
}

type stripeSubEvent struct {
	ID                 string                    `json:"id"`
	Status             stripe.SubscriptionStatus `json:"status"`
	CancelAt           int64                     `json:"cancel_at"`
	CurrentPeriodStart int64                     `json:"current_period_start"`
	CurrentPeriodEnd   int64                     `json:"current_period_end"`
	Metadata           map[string]string         `json:"metadata"`
	Items              struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	} `json:"items"`
}

type SubscriptionService struct {
	Store      *billing.PgStore
	SuccessURL string
	CancelURL  string
	ReturnURL  string
}

func (s *SubscriptionService) Checkout(ctx context.Context, customerEmail, planUID string) (map[string]any, error) {
	plan, err := s.Store.GetPlanByUID(ctx, planUID)
	if err != nil {
		return nil, err
	}
	if plan.ExternalID == "" {
		return nil, errors.New("plan not available for purchase")
	}

	customer, err := s.Store.FindOrCreateCustomer(ctx, customerEmail)
	if err != nil {
		return nil, err
	}

	existing, err := s.Store.GetSubscriptionByCustomerID(ctx, customer.ID)
	if err != nil && !errors.Is(err, billing.ErrNotFound) {
		return nil, err
	}
	if existing != nil {
		if err := s.update(ctx, existing, plan); err != nil {
			return nil, err
		}
		return map[string]any{"status": true}, nil
	}

	stripeCustomerID, err := s.resolveBillingProfile(ctx, customer, customerEmail)
	if err != nil {
		return nil, err
	}

	product, err := stripeProduct.Get(plan.ExternalID, nil)
	if err != nil {
		return nil, err
	}
	if product.DefaultPrice == nil || product.DefaultPrice.ID == "" {
		return nil, errors.New("plan has no price configured")
	}

	session, err := stripeSession.New(&stripe.CheckoutSessionParams{
		Mode:     stripe.String(string(stripe.CheckoutSessionModeSubscription)),
		Customer: stripe.String(stripeCustomerID),
		SubscriptionData: &stripe.CheckoutSessionSubscriptionDataParams{
			Metadata: map[string]string{
				"planUid":       plan.Uid,
				"customerEmail": customerEmail,
			},
		},
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Price:    stripe.String(product.DefaultPrice.ID),
				Quantity: stripe.Int64(1),
			},
		},
		Metadata: map[string]string{
			"planUid":       plan.Uid,
			"customerEmail": customerEmail,
		},
		SuccessURL: stripe.String(s.SuccessURL),
		CancelURL:  stripe.String(s.CancelURL),
	})
	if err != nil {
		return nil, err
	}

	return map[string]any{"url": session.URL}, nil
}

func (s *SubscriptionService) update(ctx context.Context, sub *billingmodels.Subscription, plan *billingmodels.Plan) error {
	product, err := stripeProduct.Get(plan.ExternalID, nil)
	if err != nil {
		return err
	}
	if product.DefaultPrice == nil || product.DefaultPrice.ID == "" {
		return errors.New("plan has no price configured")
	}

	existing, err := stripeSub.Get(sub.ExternalID, nil)
	if err != nil {
		return err
	}
	if len(existing.Items.Data) == 0 {
		return errors.New("subscription has no items")
	}

	result, err := stripeSub.Update(sub.ExternalID, &stripe.SubscriptionParams{
		ProrationBehavior: stripe.String("always_invoice"),
		Items: []*stripe.SubscriptionItemsParams{
			{
				ID:    stripe.String(existing.Items.Data[0].ID),
				Price: stripe.String(product.DefaultPrice.ID),
			},
		},
		Metadata: map[string]string{
			"planUid": plan.Uid,
		},
	})
	if err != nil {
		return err
	}

	sub.PlanID = plan.ID
	if len(result.Items.Data) > 0 {
		sub.PeriodStart = time.Unix(result.Items.Data[0].CurrentPeriodStart, 0)
		sub.PeriodEnd = time.Unix(result.Items.Data[0].CurrentPeriodEnd, 0)
	}
	return s.Store.UpdateSubscription(ctx, sub)
}

func (s *SubscriptionService) Cancel(ctx context.Context, customerEmail string) error {
	customer, err := s.Store.FindOrCreateCustomer(ctx, customerEmail)
	if err != nil {
		return err
	}

	sub, err := s.Store.GetSubscriptionByCustomerID(ctx, customer.ID)
	if err != nil {
		return err
	}

	if _, err := stripeSub.Update(sub.ExternalID, &stripe.SubscriptionParams{
		CancelAtPeriodEnd: stripe.Bool(true),
	}); err != nil { //nolint
		return err
	}

	return s.Store.UpdateSubscriptionStatus(ctx, sub.ID, billingmodels.SubscriptionCanceled)
}

func (s *SubscriptionService) Portal(ctx context.Context, customerEmail string) (string, error) {
	customer, err := s.Store.FindOrCreateCustomer(ctx, customerEmail)
	if err != nil {
		return "", err
	}

	stripeCustomerID, err := s.resolveBillingProfile(ctx, customer, customerEmail)
	if err != nil {
		return "", err
	}

	session, err := stripePortal.New(&stripe.BillingPortalSessionParams{
		Customer:  stripe.String(stripeCustomerID),
		ReturnURL: stripe.String(s.ReturnURL),
	})
	if err != nil {
		return "", err
	}

	return session.URL, nil
}

func (s *SubscriptionService) HandleEvent(ctx context.Context, event stripe.Event) error {
	switch event.Type {
	case "customer.subscription.created":
		return s.handleSubscriptionCreated(ctx, event.Data.Raw)
	case "invoice.paid":
		return s.handleInvoicePaid(ctx, event.Data.Raw)
	case "customer.subscription.updated", "customer.subscription.deleted":
		return s.handleSubscriptionUpdated(ctx, event.Data.Raw)
	}
	return nil
}

func (s *SubscriptionService) handleSubscriptionCreated(ctx context.Context, raw json.RawMessage) error {
	var data stripeSubEvent
	if err := json.Unmarshal(raw, &data); err != nil {
		return err
	}

	planUID := data.Metadata["planUid"]
	customerEmail := data.Metadata["customerEmail"]
	if planUID == "" || customerEmail == "" {
		log.Printf("activate: missing metadata planUid=%q customerEmail=%q", planUID, customerEmail)
		return nil
	}

	plan, err := s.Store.GetPlanByUID(ctx, planUID)
	if err != nil {
		return err
	}

	customer, err := s.Store.FindOrCreateCustomer(ctx, customerEmail)
	if err != nil {
		return err
	}

	sub := &billingmodels.Subscription{
		CustomerID:  customer.ID,
		PlanID:      plan.ID,
		ExternalID:  data.ID,
		Status:      billingmodels.SubscriptionActive,
		PeriodStart: time.Unix(data.CurrentPeriodStart, 0),
		PeriodEnd:   time.Unix(data.CurrentPeriodEnd, 0),
	}

	if err := s.Store.UpsertSubscriptionByCustomerID(ctx, sub); err != nil {
		return err
	}

	log.Printf("activate: customer=%s plan=%s subscription=%s", customerEmail, plan.Name, data.ID)
	return nil
}

func (s *SubscriptionService) handleInvoicePaid(ctx context.Context, raw json.RawMessage) error {
	var inv struct {
		BillingReason string `json:"billing_reason"`
		Subscription  string `json:"subscription"`
	}
	if err := json.Unmarshal(raw, &inv); err != nil {
		return err
	}
	if inv.BillingReason == "subscription_create" || inv.Subscription == "" {
		return nil
	}

	sub, err := s.Store.GetSubscriptionByExternalID(ctx, inv.Subscription)
	if errors.Is(err, billing.ErrNotFound) {
		log.Printf("renew: subscription %s not found", inv.Subscription)
		return nil
	}
	if err != nil {
		return err
	}

	log.Printf("renew: subscription %s renewed", sub.ExternalID)
	return nil
}

func (s *SubscriptionService) handleSubscriptionUpdated(ctx context.Context, raw json.RawMessage) error {
	var data stripeSubEvent
	if err := json.Unmarshal(raw, &data); err != nil {
		return err
	}

	sub, err := s.Store.GetSubscriptionByExternalID(ctx, data.ID)
	if errors.Is(err, billing.ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}

	var status billingmodels.SubscriptionStatus
	if data.Status == stripe.SubscriptionStatusCanceled || data.CancelAt != 0 {
		status = billingmodels.SubscriptionCanceled
	} else if mapped, ok := stripeStatusMap[data.Status]; ok {
		status = mapped
	} else {
		return nil
	}

	if err := s.Store.UpdateSubscriptionStatus(ctx, sub.ID, status); err != nil {
		return err
	}

	log.Printf("subscription %s → %s", data.ID, status)
	return nil
}

func (s *SubscriptionService) resolveBillingProfile(ctx context.Context, customer *billingmodels.Customer, email string) (string, error) {
	profile, err := s.Store.GetBillingProfileByCustomerID(ctx, customer.ID)
	if err != nil && !errors.Is(err, billing.ErrNotFound) {
		return "", err
	}
	if err == nil && profile.ExternalID != "" {
		return profile.ExternalID, nil
	}

	sc, err := stripeCustomer.New(&stripe.CustomerParams{
		Email: stripe.String(email),
	})
	if err != nil {
		return "", err
	}

	if err := s.Store.UpsertBillingProfile(ctx, customer.ID, sc.ID); err != nil {
		return "", err
	}

	return sc.ID, nil
}

func (s *SubscriptionService) SyncPlans(ctx context.Context) error {
	plans, err := s.Store.GetAllPlans(ctx)
	if err != nil {
		return err
	}

	for _, plan := range plans {
		cents := plan.PricePerMonth.Mul(decimal.NewFromInt(100)).IntPart()

		if plan.ExternalID != "" {
			if _, err := stripeProduct.Update(plan.ExternalID, &stripe.ProductParams{
				Name: stripe.String(plan.Name),
			}); err != nil {
				log.Printf("sync: update stripe product for plan %s: %v", plan.Name, err)
				return err
			}
			log.Printf("sync: updated stripe product %s for plan %s", plan.ExternalID, plan.Name)
			continue
		}

		product, err := stripeProduct.New(&stripe.ProductParams{
			Name: stripe.String(plan.Name),
			DefaultPriceData: &stripe.ProductDefaultPriceDataParams{
				Currency:   stripe.String("usd"),
				UnitAmount: stripe.Int64(cents),
				Recurring: &stripe.ProductDefaultPriceDataRecurringParams{
					Interval: stripe.String("month"),
				},
			},
		})
		if err != nil {
			log.Printf("sync: create stripe product for plan %s: %v", plan.Name, err)
			return err
		}

		if err := s.Store.UpdatePlanExternalID(ctx, plan.ID, product.ID); err != nil {
			log.Printf("sync: update plan %s external_id: %v", plan.Name, err)
			return err
		}

		log.Printf("sync: plan %s → stripe product %s", plan.Name, product.ID)
	}

	return nil
}
