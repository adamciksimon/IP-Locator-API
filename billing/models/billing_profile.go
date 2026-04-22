package models

import (
	"time"

	"github.com/uptrace/bun"
)

type BillingProfile struct {
	bun.BaseModel `bun:"table:billing_profiles,alias:bp"`

	ID         string    `bun:",pk,default:uuidv7()"`
	CustomerID string    `bun:",unique,notnull"`
	ExternalID string    `bun:",nullzero"`
	CreatedAt  time.Time `bun:",nullzero,notnull,default:current_timestamp"`
	UpdatedAt  time.Time `bun:",nullzero,notnull,default:current_timestamp"`

	Customer *Customer `bun:"rel:belongs-to,join:customer_id=id"`
}
