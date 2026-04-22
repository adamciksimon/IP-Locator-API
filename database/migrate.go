package database

import (
	"context"

	billingmodels "github.com/adamciksimon/public-api/billing/models"
	"github.com/uptrace/bun"
)

var models = []interface{}{
	(*billingmodels.Customer)(nil),
	(*billingmodels.Plan)(nil),
	(*billingmodels.APIKey)(nil),
	(*billingmodels.Subscription)(nil),
	(*billingmodels.BillingProfile)(nil),
}

func Reset(ctx context.Context, db *bun.DB) error {
	for i := len(models) - 1; i >= 0; i-- {
		if _, err := db.NewDropTable().Model(models[i]).IfExists().Cascade().Exec(ctx); err != nil {
			return err
		}
	}
	return Migrate(ctx, db)
}

func Migrate(ctx context.Context, db *bun.DB) error {
	for _, model := range models {
		if _, err := db.NewCreateTable().Model(model).IfNotExists().Exec(ctx); err != nil {
			return err
		}
	}

	indexes := []struct {
		model  interface{}
		name   string
		column string
	}{
		{(*billingmodels.APIKey)(nil), "idx_api_keys_key", "key"},
		{(*billingmodels.APIKey)(nil), "idx_api_keys_customer_id", "customer_id"},
		{(*billingmodels.Subscription)(nil), "idx_subscriptions_customer_id", "customer_id"},
	}

	for _, idx := range indexes {
		if _, err := db.NewCreateIndex().
			Model(idx.model).
			Index(idx.name).
			Column(idx.column).
			IfNotExists().
			Exec(ctx); err != nil {
			return err
		}
	}

	return nil
}
