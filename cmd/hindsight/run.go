package main

import (
	"fmt"
	"net/http"

	"github.com/0x6377/hindsight"
)

func run(c *hindsight.Config) error {
	storage, err := hindsight.NewSQLiteStorage(c.DatabasePath)
	if err != nil {
		return err
	}
	mux := hindsight.CreateAPIHandler(c, storage)

	return http.ListenAndServe(fmt.Sprintf(":%d", c.ListenAPI), mux)
}
