package main

import (
	"context"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

var db *pgxpool.Pool

func initDB() error {
	dsn := os.Getenv("DATABASE_URL")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return err
	}

	// test connection
	if err := pool.Ping(ctx); err != nil {
		return err
	}

	db = pool
	return nil
}

type missingEnvError struct{ name string }

func (e *missingEnvError) Error() string { return "missing env: " + e.name }
