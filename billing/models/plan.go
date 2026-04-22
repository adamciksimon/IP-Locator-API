package models

import (
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

type Plan struct {
	bun.BaseModel `bun:"table:plans,alias:p"`

	ID            string          `bun:",pk,default:uuidv7()" json:"id"`
	Uid           string          `bun:",unique,notnull" json:"uid"`
	Name          string          `bun:",unique,notnull" json:"name"`
	RequestLimit  int64           `bun:",notnull,default:0" json:"requestLimit"`
	RateLimit     int64           `bun:",notnull,default:0" json:"rateLimit"`
	PricePerMonth decimal.Decimal `bun:",notnull,type:numeric(10,2),default:0" json:"pricePerMonth"`
	ExternalID    string          `bun:",nullzero" json:"-"`
}
