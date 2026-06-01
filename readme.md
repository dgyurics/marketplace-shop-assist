<p align="center">
  <img src="https://github.com/dgyurics/marketplace/blob/main/logo.webp?raw=true" alt="marketplace">
</p>

Proof of concept. A self-hosted AI sidecar service that lets customers find products using natural language instead of keyword search. Ask it _"a lamp for my office, under $500"_ and it returns the most relevant products.

Built as a companion to [marketplace](https://github.com/dgyurics/marketplace).

## How it works

When a query is received from the marketplace backend:

1. Query is embedded using a local model (Ollama)
2. Vector similarity search finds candidate products
3. Top-K products are passed to the LLM
4. LLM returns a ranked + natural language response
5. Response is returned to the marketplace API

## Stack

- **Go** — single HTTP endpoint (`POST /query`)
- **Ollama** — runs the LLM and embedding model locally, no API keys needed
- **In-memory vector store** — cosine similarity search, no database required
- **Static seed data** — hardcoded products mirroring the marketplace schema

## Architecture

This service runs as an optional sidecar alongside the main marketplace backend.

Marketplace UI
    ↓
Marketplace API (Go)
    ↓
gRPC (or HTTP fallback)
    ↓
AI Sidecar (this service)
    ↓
Vector search + LLM (Ollama)

## Future

Swap the in-memory store for pgvector and point it at the marketplace's existing Postgres instance.