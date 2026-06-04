package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strings"

	_ "github.com/lib/pq"
)

type Product struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Price       int64           `json:"price"`
	Description *string         `json:"description,omitempty"`
	Details     json.RawMessage `json:"details,omitempty"`
}

type Entry struct {
	Product   Product
	Embedding []float64
}

var store []Entry

func embed(ctx context.Context, text string) ([]float64, error) {
	body, _ := json.Marshal(map[string]string{
		"model":  os.Getenv("EMBED_MODEL"), // nomic-embed-text
		"prompt": text,
	})
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, os.Getenv("OLLAMA_URL")+"/api/embeddings", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Embedding []float64 `json:"embedding"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result.Embedding, nil
}

var htmlTags = regexp.MustCompile(`<[^>]+>`)

func stripHTML(s string) string {
	s = htmlTags.ReplaceAllString(s, "")
	s = strings.ReplaceAll(s, "\n", " ")
	return strings.TrimSpace(s)
}

func seed(ctx context.Context, db *sql.DB) error {
	rows, err := db.QueryContext(ctx, `
		SELECT id::text, name, description, price, details
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
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.Price, &p.Details); err != nil {
			return fmt.Errorf("scan product: %w", err)
		}

		desc := ""
		if p.Description != nil {
			desc = stripHTML(*p.Description)
			p.Description = &desc
		}

		details := ""
		if p.Details != nil {
			var attrs map[string]interface{}
			json.Unmarshal(p.Details, &attrs)
			var parts []string
			for k, v := range attrs {
				parts = append(parts, fmt.Sprintf("%s: %v", k, v))
			}
			details = strings.Join(parts, ", ")
		}

		vec, err := embed(ctx, fmt.Sprintf("%s. %s. Price: $%.2f %s", p.Name, desc, float64(p.Price)/100, details))
		if err != nil {
			return fmt.Errorf("embed product %s: %w", p.ID, err)
		}

		store = append(store, Entry{Product: p, Embedding: vec})
		count++
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterating products: %w", err)
	}

	slog.Info("Seeding complete", "products", count)
	return nil
}

// cosineSimilarity measures how similar two vectors are by computing the cosine
// of the angle between them.
//
// The dot product of two vectors tells you how much they point in the same
// direction, but it's affected by the length of the vectors — two long vectors
// will have a bigger dot product than two short ones even if they point the
// same way. So we divide by both lengths to cancel that out. What's left is
// purely the angle between them, which is all we care about.
//
//	cos(0°)  = 1.0  → same direction, identical meaning
//	cos(90°) = 0.0  → perpendicular, unrelated
//	cos(180°) = -1.0 → opposite directions
//
// We never actually compute the angle itself — just its cosine, which is
// enough to rank results by similarity.
func cosineSimilarity(a, b []float64) float64 {
	var dot, normA, normB float64
	for i := range a {
		dot += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}

func topMatches(queryVec []float64, k int) []Product {
	type scored struct {
		product Product
		score   float64
	}
	scores := make([]scored, len(store))
	for i, e := range store {
		scores[i] = scored{e.Product, cosineSimilarity(queryVec, e.Embedding)}
	}

	// Return top k results
	sort.Slice(scores, func(i, j int) bool { return scores[i].score > scores[j].score })
	if k > len(scores) {
		k = len(scores)
	}
	results := make([]Product, k)
	for i := range results {
		results[i] = scores[i].product
	}
	return results
}

func handleQuery(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Query string `json:"query"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Query == "" {
		http.Error(w, "missing query", http.StatusBadRequest)
		return
	}

	queryVec, err := embed(r.Context(), req.Query)
	if err != nil {
		slog.Error("Failed to embed query", "error", err)
		http.Error(w, "embedding failed", http.StatusInternalServerError)
		return
	}

	matches := topMatches(queryVec, 5)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(matches)
}

func debug(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	type summary struct {
		ID     string `json:"id"`
		Name   string `json:"name"`
		VecLen int    `json:"vec_len"`
	}
	summaries := make([]summary, len(store))
	for i, e := range store {
		summaries[i] = summary{e.Product.ID, e.Product.Name, len(e.Embedding)}
	}
	json.NewEncoder(w).Encode(summaries)
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

	http.HandleFunc("/debug", debug)
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
