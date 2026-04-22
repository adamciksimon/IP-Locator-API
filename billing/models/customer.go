package models

import (
	"time"

	"github.com/uptrace/bun"
)

type Customer struct {
	bun.BaseModel `bun:"table:customers,alias:c"`

	ID        string    `bun:",pk,default:uuidv7()"`
	Email     string    `bun:",unique,notnull"`
	CreatedAt time.Time `bun:",nullzero,notnull,default:current_timestamp"`
	UpdatedAt time.Time `bun:",nullzero,notnull,default:current_timestamp"`
}
