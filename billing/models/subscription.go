package models

import (
	"time"

	"github.com/uptrace/bun"
)

type SubscriptionStatus string

const (
	SubscriptionActive   SubscriptionStatus = "active"
	SubscriptionCanceled SubscriptionStatus = "canceled"
	SubscriptionPastDue  SubscriptionStatus = "past_due"
)

type Subscription struct {
	bun.BaseModel `bun:"table:subscriptions,alias:s"`

	ID          string             `bun:",pk,default:uuidv7()"`
	CustomerID  string             `bun:",notnull"`
	PlanID      string             `bun:",notnull"`
	ExternalID  string             `bun:",nullzero"`
	Status      SubscriptionStatus `bun:",notnull,default:'active'"`
	PeriodStart time.Time          `bun:",notnull"`
	PeriodEnd   time.Time          `bun:",notnull"`
	CreatedAt   time.Time          `bun:",nullzero,notnull,default:current_timestamp"`
	UpdatedAt   time.Time          `bun:",nullzero,notnull,default:current_timestamp"`

	Plan *Plan `bun:"rel:belongs-to,join:plan_id=id"`
}
