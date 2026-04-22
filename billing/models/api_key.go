package models

import (
	"time"

	"github.com/uptrace/bun"
)

type APIKey struct {
	bun.BaseModel `bun:"table:api_keys,alias:ak"`

	ID         string    `bun:",pk,default:uuidv7()"`
	CustomerID string    `bun:",notnull"`
	Key        string    `bun:",unique,notnull"`
	Active     bool      `bun:",notnull,default:true"`
	LastUsedAt time.Time `bun:",nullzero"`
	CreatedAt  time.Time `bun:",nullzero,notnull,default:current_timestamp"`

	Customer *Customer `bun:"rel:belongs-to,join:customer_id=id"`
}
