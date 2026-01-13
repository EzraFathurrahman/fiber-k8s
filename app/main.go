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

	// GET list semua transaksi
	app.Get("/transactions", func(c *fiber.Ctx) error {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		rows, err := db.Query(ctx, `SELECT id, amount, created_at FROM transactions ORDER BY created_at DESC`)
		if err != nil {
			return c.Status(500).SendString("query error")
		}
		defer rows.Close()

		type Transaction struct {
			ID        int    `json:"id"`
			Amount    int    `json:"amount"`
			CreatedAt string `json:"created_at"`
		}

		var transactions []Transaction
		for rows.Next() {
			var t Transaction
			var createdAt time.Time
			if err := rows.Scan(&t.ID, &t.Amount, &createdAt); err != nil {
				return c.Status(500).SendString("scan error")
			}
			t.CreatedAt = createdAt.Format(time.RFC3339)
			transactions = append(transactions, t)
		}

		if err := rows.Err(); err != nil {
			return c.Status(500).SendString("rows error")
		}

		return c.JSON(fiber.Map{"status": "success", "data": transactions})
	})

	// UPDATE transaksi berdasarkan ID
	app.Put("/transactions/:id", func(c *fiber.Ctx) error {
		id, err := c.ParamsInt("id")
		if err != nil {
			return c.Status(400).SendString("invalid id")
		}

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

		// update transaksi
		result, err := tx.Exec(ctx, `UPDATE transactions SET amount = $1 WHERE id = $2`, req.Amount, id)
		if err != nil {
			return c.Status(500).SendString("update error")
		}

		if result.RowsAffected() == 0 {
			return c.Status(404).SendString("transaction not found")
		}

		if err := tx.Commit(ctx); err != nil {
			return c.Status(500).SendString("commit error")
		}

		return c.JSON(fiber.Map{"status": "success", "id": id, "amount": req.Amount})
	})

	app.Listen(":3000")
}
