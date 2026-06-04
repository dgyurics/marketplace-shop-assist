<p align="center">
  <img src="https://github.com/dgyurics/marketplace/blob/main/logo.webp?raw=true" alt="marketplace">
</p>

Proof of concept. AI-powered product search for Marketplace. Users can search with natural language instead of keywords:

> "A lamp for my office under $500"

The service performs semantic search using vector embeddings, retrieves relevant products, and uses an LLM to generate ranked results.

Built as a companion to Marketplace.

## How It Works

1. Generate embeddings for product data and store them in a vector index
2. Generate an embedding for the user's query
3. Perform cosine similarity search to retrieve the most relevant products
4. Pass the top results to the LLM as contextual data (Retrieval-Augmented Generation)
5. Return a ranked, natural-language response

## Stack

- Go — HTTP API (POST /query)
- Ollama — local LLM and embedding models
- In-memory vector store — cosine similarity search
- Seed data — sample products matching the Marketplace schema

## Future Work

- Replace the in-memory vector store with pgvector
- Index products directly from Marketplace PostgreSQL
- Support incremental re-indexing and larger catalogs