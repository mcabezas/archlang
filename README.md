# ArchLang

[![Go](https://img.shields.io/badge/Go-1.24+-00ADD8?style=flat&logo=go&logoColor=white)](https://go.dev)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Report Card](https://goreportcard.com/badge/github.com/mcabezas/archlang)](https://goreportcard.com/report/github.com/mcabezas/archlang)
[![CI](https://github.com/mcabezas/archlang/actions/workflows/ci.yml/badge.svg)](https://github.com/mcabezas/archlang/actions/workflows/ci.yml)

**Architecture documentation that never lies.**

ArchLang is a declarative language for defining solution architectures as facts. It compiles `.arch` files into a typed, queryable knowledge graph — serving architecture facts through a REST API, an MCP server, and a Claude Code skill.

No more outdated wikis. No more tribal knowledge. No more diagrams that rot the day after they're drawn. If it compiles, it's true.

<p align="center">
  <img src="doc/archie.jpg" alt="Archie — the ArchLang mascot" width="250">
</p>

## The Problem

Architecture knowledge lives in the worst possible places: Confluence pages nobody updates, Miro boards nobody checks, and the heads of engineers who leave.

When a new team member asks "what depends on the cauldron service?", the answer is a 30-minute meeting. When an AI agent needs to make an implementation decision, it hallucinates one.

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

- **Write it** — Human-readable `.arch` files define services, events, collaborations, features, flows, and steps
- **Compile it** — The compiler validates everything at build time. Undeclared references, cross-org visibility violations, and orphan events are caught — not discovered in production
- **Query it** — The compiled graph is served through REST and MCP. Same question, same answer, every time
- **Trace it** — Features, flows, and steps are first-class citizens. Trace a business capability across every collaboration in your architecture

```
service cauldron "Potion mixing engine"
service grimoire "Spell recipe keeper"
service owl-post "Delivery dispatch"

event PotionBrewed "A potion has been brewed"
event SpellValidated "A spell recipe has been validated"
event DeliveryDispatched "An owl has been dispatched"

feature brew-potion: "Brew and deliver a potion" {
    collaboration grimoire -> SpellValidated

    collaboration cauldron <- SpellValidated {
        execute: mixIngredients
        publishes: PotionBrewed
    }

    collaboration owl-post <- PotionBrewed {
        execute: dispatchOwl
        publishes: DeliveryDispatched
    }
}
```

This compiles. Every reference is validated. Events are first-class graph nodes. The `execute` property links a subscription to the action that handles it. The `publishes` property traces the chain of events that follow.

## Key Concepts

**Services** — What runs in your architecture. Declared with an optional description: `service cauldron "Potion mixing engine"`.

**Events** — Facts that happened. First-class graph nodes, just like services. Declared with `event PotionBrewed "A potion has been brewed"`. Events must be declared before use.

**Collaborations** — How things communicate. Service-to-service (`->`) for direct calls. Service-to-event (`->`) for publishing. Event-to-service (`<-`) for subscribing. Each collaboration can carry a feature, description, cardinality, flow, and step.

**Publish / Subscribe** — `collaboration grimoire -> SpellValidated` means grimoire publishes the event. `collaboration cauldron <- SpellValidated { execute: mixIngredients }` means cauldron subscribes and runs `mixIngredients`. The `publishes` property declares what events the handler produces, creating traceable event chains.

**Organizations** — Inferred from `orgs/` folder structure. Components that receive cross-org calls must be `public`. Reference cross-org components with `org/name` syntax: `collaboration owl-post -> ravens/raven-tower`. No imports needed.

**Features** — Declared with a name and description. Can be standalone or wrap a block of collaborations. Trace a feature across the entire graph to see every service and event involved.

**Flows & Steps** — Group collaborations into named sequences. Steps label phases within a flow. Steps are ordered automatically by their position in the source.

**Visibility** — `public` or `internal`. Only public components can receive calls from other organizations. The compiler rejects anything else.

**Strict mode** — Pass `--strict` to surface warnings like orphan events (published but nobody subscribes).

## Syntax

### Events

Declare events and wire them with publish/subscribe collaborations:

```
event PotionBrewed "A potion has been brewed"
event PotionBottled "A potion has been bottled"

# Publish — service produces an event
collaboration cauldron -> PotionBrewed

# Subscribe — service reacts to an event
collaboration bottler <- PotionBrewed {
    execute: bottlePotion
    publishes: PotionBottled
}

# Multiple publishes from one handler
collaboration owl-post <- PotionBottled {
    execute: prepareDelivery
    publishes: [OwlAssigned, PackageSealed, DeliveryDispatched]
}
```

The `execute` property is only valid on event collaborations — using it on service-to-service is a compile error. The `publishes` property targets must be declared events.

### Collaborations

A collaboration can be plain or carry metadata:

```
# Plain — just an edge
collaboration grimoire -> cauldron

# With description and cardinality
collaboration cauldron -> owl-post {
    feature brew-potion: "Dispatch brewed potion"
    cardinality: one to many by region
}
```

### Feature Blocks

Wrap collaborations in a feature block — all collaborations inside inherit the feature automatically:

```
feature brew-potion: "Brew and deliver a potion" {
    collaboration grimoire -> cauldron {
        description: "Sends validated spell recipe"
    }
    collaboration cauldron -> owl-post {
        description: "Dispatches brewed potion for delivery"
    }
}
```

### Flow Blocks

Group collaborations into named flows with optional descriptions and steps:

```
feature brew-potion: "Brew and deliver a potion" {
    flow brewing "From recipe to bottled potion" {
        collaboration grimoire -> cauldron {
            description: "Sends validated spell recipe"
            step: validate
        }
        collaboration cauldron -> bottler {
            description: "Passes brewed potion for bottling"
            step: bottle
        }
    }
}
```

### Cross-Org References

Reference components in other organizations using `org/name` syntax:

```
collaboration owl-post -> ravens/raven-tower {
    description: "Fallback delivery via raven network"
}
```

No imports needed. The target must be declared `public` in its org.

### Dots in Names

Identifiers can contain dots for namespacing:

```
event potion.brewed "A potion has been brewed"
event potion.delivered "A potion has been delivered"

collaboration owl-post <- potion.brewed {
    execute: dispatchOwl
}
```

## How It Works

```
.arch files → ArchLang Compiler → Knowledge Graph → MCP Server → Claude Code Skill → Teams
```

1. Teams write `.arch` files — the single source of truth
2. The compiler generates a typed Go graph with all validations enforced
3. The Architecture Documentation Service exposes the graph via REST and MCP
4. The `/arch` skill in Claude Code queries the MCP server and presents results to engineers

The graph is deterministic. No AI in the data path. No hallucinations. The skill answers from compiled facts.

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

### 1. Define your architecture

Create `.arch` files inside an `architecture/` directory. Organize by org using folders:

```
architecture/
  orgs/
    hogwarts/
      services.arch
      events.arch
      potions.arch
    ravens/
      services.arch
```

Names are global — every service, event, and component must be unique. The folder structure determines the org for visibility enforcement.

### 2. Generate Go code

```bash
archlang generate ./architecture --out ./generated --package generated
```

This compiles your `.arch` files into a type-safe Go package with all validations enforced.

### 3. Serve it

Create a `main.go` that imports the generated code and starts the built-in HTTP server:

```go
package main

import (
	"log"
	"os"

	"your-module/generated"

	sdk "github.com/mcabezas/archlang/sdk"
)

func main() {
	svc := sdk.New(generated.AllGraphs, nil)

	port := os.Getenv("REST_SERVER_PORT")
	if port == "" {
		port = "8080"
	}

	server := sdk.NewHTTPServer(svc, ":"+port)
	if err := server.Start(); err != nil {
		log.Fatal(err)
	}
}
```

The server handles graceful shutdown on `SIGINT`/`SIGTERM` out of the box.

### 4. Browse

- **Architecture Overview** — `http://localhost:8080/diagram`
- **Feature Diagram** — `http://localhost:8080/diagram?feature=brew-potion`
- **Component API** — `http://localhost:8080/api/components/cauldron`

Diagrams are rendered as interactive Mermaid charts with a dark theme. Services appear as rectangles, events as stadium shapes in teal with dotted arrows. Feature diagrams include flow and step breakdowns.

### Custom transports

The generated code is a standalone Go package. You can build your own transport layer on top of the SDK:

```go
svc := sdk.New(generated.AllGraphs, nil)

// Use the SDK directly
components, _ := svc.ListAll()
features, _ := svc.ListFeatures()
component, _ := svc.FindByName("cauldron")
collabs, _ := svc.FindByFlow("brewing")
```

## MCP Server

ArchLang ships with a built-in MCP (Model Context Protocol) server that exposes the compiled architecture graph as tools. This is the backbone for the Claude Code skill and can also be used by any MCP-compatible client.

### Available Tools

| Tool | Description |
|---|---|
| `list_components` | List all components with kind, org, and visibility |
| `get_component` | Get a component's full details including upstream and downstream collaborations |
| `list_features` | List all declared business features |
| `trace_feature` | Trace a feature across every service, event, flow, and step |
| `list_flows` | List all declared flows |
| `trace_flow` | Trace a flow step by step across services |
| `analyze_impact` | Analyze what would break if a component changes — affected features, flows, and testing recommendations |

### Setup

#### 1. Create the MCP entry point

```go
// cmd/mcp/main.go
package main

import (
	"log"
	"os"

	"your-module/generated"

	sdk "github.com/mcabezas/archlang/sdk"
)

func main() {
	svc := sdk.New(generated.AllGraphs, nil)
	mcp := sdk.NewMCPServer(svc)

	port := os.Getenv("MCP_SERVER_PORT")
	if port == "" {
		port = "9090"
	}

	if err := mcp.ServeSSE(":" + port); err != nil {
		log.Fatal(err)
	}
}
```

#### 2. Build and run

```bash
go build -o archlang-mcp ./cmd/mcp
MCP_SERVER_PORT=9090 ./archlang-mcp
```

#### 3. Configure your AI assistant

Create `.mcp.json` in your project root:

```json
{
  "mcpServers": {
    "archlang": {
      "command": "/absolute/path/to/archlang-mcp"
    }
  }
}
```

## Claude Code Skill

ArchLang provides an `/arch` skill for Claude Code that uses the MCP server as its source of truth. When invoked, the skill queries the compiled architecture graph — it never searches source files or guesses.

### Install

```bash
# Symlink (recommended — stays up to date with the repo)
ln -s /path/to/archlang/skills/arch ~/.claude/skills/arch

# Or copy
cp -r /path/to/archlang/skills/arch ~/.claude/skills/arch
```

### Usage

Once installed, use it in any Claude Code session:

```
/arch brew-potion                    # Trace a feature end to end
/arch brew-potion open               # Trace and open Mermaid diagram in browser
/copy arch brew-potion               # Copy Mermaid diagram to clipboard
```

### What it does

- Queries the MCP server for architecture facts (features, flows, components, events, impact)
- Presents results in plain language with Mermaid sequence diagrams
- Refuses to guess — if the MCP server doesn't have it, it says so
- Suggests `make mcp-up` if the server isn't running

### Example questions via the skill

- *"What services exist in the system?"*
- *"What depends on the cauldron?"*
- *"Trace the brew-potion feature end to end"*
- *"What events does the owl-post service publish?"*
- *"What would break if we change the grimoire?"*

Every answer comes from the compiled graph — deterministic, accurate, always up to date.

## Contributing

ArchLang is under active development. Contributions are welcome — open an issue or pull request.

## License

Unless otherwise noted, the ArchLang source files are distributed under the MIT license found in the LICENSE file.
