package main

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
)

func main() {
	if err := initDB(); err != nil {
		panic(err)
	}

	app := fiber.New()

	// LIVENESS: app hidup (nggak cek DB)
	app.Get("/health/live", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	// READINESS: siap nerima traffic (cek DB)
	app.Get("/health/ready", func(c *fiber.Ctx) error {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		if err := db.Ping(ctx); err != nil {
			return c.Status(503).SendString("db not ready")
		}
		return c.SendString("ready")
	})

	// transaksi sederhana tapi real: insert + commit
	app.Post("/transactions", func(c *fiber.Ctx) error {
		type Req struct {
			Amount int `json:"amount"`
		}
		var req Req
		if err := c.BodyParser(&req); err != nil {
			return c.Status(400).SendString("invalid body")
		}
		if req.Amount <= 0 {
			return c.Status(400).SendString("amount must be > 0")
		}

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		tx, err := db.Begin(ctx)
		if err != nil {
			return c.Status(500).SendString("tx begin error")
		}
		defer tx.Rollback(ctx)

		// contoh transaction: insert row
		_, err = tx.Exec(ctx, `INSERT INTO transactions(amount) VALUES ($1)`, req.Amount)
		if err != nil {
			return c.Status(500).SendString("insert error")
		}

		if err := tx.Commit(ctx); err != nil {
			return c.Status(500).SendString("commit error")
		}

		return c.JSON(fiber.Map{"status": "success", "amount": req.Amount})
	})

	app.Listen(":3000")
}
