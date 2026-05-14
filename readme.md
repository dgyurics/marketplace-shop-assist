<p align="center">
  <img src="https://github.com/dgyurics/marketplace/blob/main/logo.webp?raw=true" alt="marketplace">
</p>

Proof of concept. A self-hosted AI sidecar service that lets customers find products using natural language instead of keyword search. Ask it _"a lamp for my office, under $500"_ and it returns the most relevant products.

Built as a companion to [marketplace](https://github.com/dgyurics/marketplace).

## How it works

A RAG (Retrieval-Augmented Generation) pipeline:

1. Product data is embedded into vectors and stored in memory
2. A customer query is embedded the same way and matched against the catalog via cosine similarity
3. The top matches are passed to a local LLM, which returns a natural language answer

## Stack

- **Go** — single HTTP endpoint (`POST /query`)
- **Ollama** — runs the LLM and embedding model locally, no API keys needed
- **In-memory vector store** — cosine similarity search, no database required
- **Static seed data** — hardcoded products mirroring the marketplace schema

## Future

Swap the in-memory store for pgvector and point it at the marketplace's existing Postgres instance.