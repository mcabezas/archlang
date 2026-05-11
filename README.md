# ArchLang

**Architecture documentation that never lies.**

ArchLang is a programming language for defining solution architectures. It compiles `.arch` files into a typed, queryable knowledge graph — serving architecture facts to humans and AI agents through multiple protocols.

No more outdated wikis. No more tribal knowledge. No more diagrams that rot the day after they're drawn. If it compiles, it's true.

<p align="center">
  <img src="doc/archie.jpg" alt="Archie — the ArchLang mascot" width="250">
</p>

## The Problem

Architecture knowledge lives in the worst possible places: Confluence pages nobody updates, Miro boards nobody checks, and the heads of engineers who leave.

When a new team member asks "what depends on the order service?", the answer is a 30-minute meeting. When an AI agent needs to make an implementation decision, it hallucinates one.

Your code has type safety. Your infrastructure has Terraform. **Your architecture solution knowledge has nothing.**

## Why Nothing Like This Exists Yet

Tools exist in the neighborhood. None of them solve the actual problem.

| | Compiles | Validates refs | Org boundaries | Feature tracing | Queryable API | AI-agent ready |
|---|---|---|---|---|---|---|
| **Structurizr / C4 DSL** | No | No | No | No | No | No |
| **Backstage / Port / Cortex** | No | No | No | No | Partial | No |
| **Mermaid / PlantUML** | No | No | No | No | No | No |
| **Confluence / Wikis** | No | No | No | No | No | No |
| **ArchLang** | **Yes** | **Yes** | **Yes** | **Yes** | **Yes** | **Yes** |

Every existing tool either generates static diagrams or maintains a manual catalog. None of them **compile**. None of them treat an undeclared dependency as an error. None of them enforce organizational boundaries. None of them let you trace a business feature across every service collaboration. And none of them were designed for a world where AI agents need structured, deterministic facts to make implementation decisions — not Confluence pages to hallucinate from.

ArchLang is what happens when you apply the same rigor we already use for code and infrastructure to the one thing that still lives on whiteboards: architecture.

## The Solution

ArchLang treats architecture like code:

- **Write it** — Human-readable `.arch` files define components, services, collaborations, and features
- **Compile it** — The compiler validates everything at build time. Undeclared references, cross-org visibility violations, and missing imports are compile errors — not runtime surprises
- **Query it** — The compiled graph is served through REST, gRPC, MCP, and Slack. Same question, same answer, every time
- **Trace it** — Features are first-class citizens. Trace a single feature across every collaboration in your architecture

```
import orgs/acme

feature checkout: "Process order payments at checkout"
feature notifications: "Send transactional notifications"

public service api-gateway
service order-service
service payment-service
public service notification-service

collaboration api-gateway -> order-service {
  feature checkout
}
collaboration order-service -> payment-service {
  feature checkout: "REST POST /payments with order payload and idempotency key"
  cardinality 1:1
}
collaboration order-service -> notification-service {
  feature notifications
}
collaboration notification-service -> orgs/acme.email-provider {
  feature notifications
}
```

This compiles. Every reference is validated. Cross-org targets are checked for public visibility. The `notifications` feature can be traced from `order-service` all the way to `acme`.

## Key Concepts

**Components, Services & Infra** — Define what exists in your architecture.

**Organizations** — Inferred from `orgs/` folder structure. Components that receive cross-org calls must be `public`. Enforced at compile time.

**Collaborations** — Define how components communicate. Each collaboration block carries one feature (with an optional inline description) and an optional cardinality (`1:1` or `1:N`). Duplicate collaborations between the same pair are allowed — one per feature.

**Features** — Declared with a name and description. Referenced inside collaborations. Trace a feature across the entire graph to see every service involved.

**Visibility** — `public` or `internal`. Only public components can receive calls from other organizations. A service doesn't need to be public to call external services — only the target must be public. The compiler rejects anything else.

## Collaboration Blocks

A collaboration can be plain or carry a feature and a description explaining the integration:

```
# Plain — just an edge
collaboration api-gateway -> order-service

# With a feature
collaboration order-service -> payment-service {
  feature checkout
}

# With a feature, description, and cardinality
collaboration order-service -> payment-service {
  feature refund: "Async event via message queue for refund processing"
  cardinality 1:1
}
```

Each block carries **one feature** (with an optional inline description) and an optional **cardinality** (`1:1` or `1:N`). To describe multiple features between the same pair, use separate blocks — one per feature.

## How It Works

```
.arch files → ArchLang Compiler → Knowledge Graph → API → AI Agents → Teams
```

1. Teams write `.arch` files — the single source of truth
2. The compiler generates a typed Go graph with all validations enforced
3. The Architecture Documentation Service exposes the graph via HTTP, gRPC, MCP, and Slack
4. An AI agent consumes the knowledge and serves it to engineering teams

The graph is deterministic. No AI in the data path. No hallucinations. The agent answers from compiled facts.

## Install

```bash
go install github.com/mcabezas/archlang/cmd/archlang@latest
```

### From Source

```bash
git clone https://github.com/mcabezas/archlang.git
cd archlang
go install ./cmd/archlang
```

## Usage

```bash
# Generate Go code from .arch files
archlang generate ./architecture --out ./generated --package generated
```

The generated code is a standalone Go package. Import it into your service, wire it to any transport layer, and serve.

## Agents

The [`agents/`](agents/) directory contains agent implementations that wrap the Architecture Knowledge Graph. They provide natural language interfaces while guaranteeing all answers come from the compiled architecture — not from guessing.

- **MCP Server** — Model Context Protocol for AI coding assistants
- **Slack Bot** — Architecture queries in Slack channels

## Contributing

ArchLang is under active development. Contributions are welcome — open an issue or pull request.

## License

Unless otherwise noted, the ArchLang source files are distributed under the MIT license found in the LICENSE file.
