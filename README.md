# ArchLang

A programming language for defining solution architectures. Compile `.arch` files into a deterministic REST API that serves architecture knowledge to humans and AI agents alike.

## Why ArchLang?

Architecture knowledge is often scattered across wikis, diagrams, and tribal knowledge. When AI agents need to understand your system to make implementation decisions, they need **deterministic, structured facts** — not documents that can be misinterpreted.

ArchLang solves this with two layers:

1. **Architecture Knowledge Engine** — A compiled REST API that returns exact, repeatable responses. No AI, no interpretation, no hallucinations.
2. **Architecture Knowledge Agent** — An optional AI layer that wraps the engine, translating natural language questions into API calls. The agent may paraphrase, but the data is always exact.

## Language

Define your architecture in `.arch` files:

```
component payments
component users
component ledger
component checkout

collaboration checkout -> payments
collaboration payments -> users
collaboration payments -> ledger
```

## Usage

```bash
# Compile and start the API server
archlang .

# Compile to a standalone binary
archlang build .
```

## API

```
GET /components                          # List all components
GET /components/{name}                   # Component details
GET /components/{name}/upstreams         # Who depends on this component
GET /components/{name}/downstreams       # What this component depends on
```

Every endpoint returns deterministic JSON. Same input, same output, every time.

## Agents

The `agents/` directory contains official agent implementations that wrap the ArchLang engine. These agents provide natural language interfaces while guaranteeing that all data comes from the compiled architecture definition.

## Status

Under active development.

## License

MIT
