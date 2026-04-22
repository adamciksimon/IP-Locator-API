package billing

import (
	"context"

	billingmodels "github.com/adamciksimon/public-api/billing/models"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

var Plans = []billingmodels.Plan{
	{
		Name:          "Free",
		Uid:           "free",
		RequestLimit:  1000,
		RateLimit:     10,
		PricePerMonth: decimal.Zero,
	},
	{
		Name:          "Mini",
		Uid:           "mini",
		RequestLimit:  50000,
		RateLimit:     100,
		PricePerMonth: decimal.NewFromFloat(19.00),
	},
	{
		Name:          "Standard",
		Uid:           "standard",
		RequestLimit:  500000,
		RateLimit:     1000,
		PricePerMonth: decimal.NewFromFloat(79.00),
	},
	{
		Name:          "Pro",
		Uid:           "pro",
		RequestLimit:  0,
		RateLimit:     0,
		PricePerMonth: decimal.NewFromFloat(299.00),
	},
}

func Seed(ctx context.Context, db *bun.DB) error {
	_, err := db.NewInsert().
		Model(&Plans).
		On("CONFLICT (name) DO UPDATE").
		Set("request_limit = EXCLUDED.request_limit").
		Set("rate_limit = EXCLUDED.rate_limit").
		Set("price_per_month = EXCLUDED.price_per_month").
		Exec(ctx)
	return err
}
