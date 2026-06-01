package main

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	_ "github.com/lib/pq"
)

type Product struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Price       int64   `json:"price"`
	Description *string `json:"description,omitempty"`
}

type Entry struct {
	Product   Product
	Embedding []float64
}

var store []Entry

func embed(ctx context.Context, text string) ([]float64, error) {
	return nil, nil
}

func seed(ctx context.Context, db *sql.DB) error {
	rows, err := db.QueryContext(ctx, `
		SELECT id::text, name, description, price
		FROM products
		WHERE is_deleted = false
	`)
	if err != nil {
		return fmt.Errorf("query products: %w", err)
	}
	defer rows.Close()

	var count int
	for rows.Next() {
		var p Product
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.Price); err != nil {
			return fmt.Errorf("scan product: %w", err)
		}

		// desc := ""
		// if p.Description != nil {
		// 	desc = *p.Description
		// }

		// vec, err := embed(ctx, fmt.Sprintf("%s. %s. Price: $%.2f", p.Name, desc, float64(p.Price)/100))
		// if err != nil {
		// 	return fmt.Errorf("embed product %s: %w", p.ID, err)
		// }

		// store = append(store, Entry{Product: p, Embedding: vec})
		count++
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterating products: %w", err)
	}

	slog.Info("Seeding complete", "products", count)
	return nil
}

func handleQuery(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "not yet implemented", http.StatusNotImplemented)
}

func main() {
	ctx := context.Background()

	dataSourceName := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"),
		os.Getenv("DB_SSLMODE"),
	)

	db, err := sql.Open("postgres", dataSourceName)
	if err != nil {
		slog.Error("Failed to open database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	if err := seed(ctx, db); err != nil {
		slog.Error("Seeding failed", "error", err)
		os.Exit(1)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	http.HandleFunc("/query", handleQuery)
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{"status":"ok","products":%d}`, len(store))
	})

	slog.Info("shop-assist listening", "port", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		slog.Error("Server failed", "error", err)
		os.Exit(1)
	}
}
