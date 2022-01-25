package main

import (
	"context"

	"github.com/0x6377/hindsight"
)

func run(c *hindsight.Config) error {
	storage, err := hindsight.NewSQLiteStorage(c.DatabasePath)
	if err != nil {
		return err
	}
	ctx := context.Background()

	return hindsight.ListenForIngestion(ctx, c, storage)
}
