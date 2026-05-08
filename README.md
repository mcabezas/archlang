# The ArchLang Programming Language

ArchLang is a programming language for defining solution architectures that
compiles into a deterministic API, serving architecture knowledge to humans
and AI agents through multiple communication protocols.

<p align="center">
  <img src="doc/archie.jpg" alt="Archie — the ArchLang mascot" width="250">
</p>

Unless otherwise noted, the ArchLang source files are distributed under the
MIT license found in the LICENSE file.

## About

Architecture knowledge is often scattered across wikis, diagrams, and tribal
knowledge. When AI agents need to understand your system to make implementation
decisions, they need deterministic, structured facts — not documents open to
interpretation.

ArchLang compiles `.arch` definition files into an Architecture Knowledge
Engine that returns exact, repeatable responses. No AI in the data path.
No hallucinations. Same question, same answer, every time.

## The Language

```
component payments
component users
component ledger
component checkout

collaboration checkout -> payments
collaboration payments -> users
collaboration payments -> ledger
```

The compiler validates all definitions at build time. Referencing an
undeclared component is a compile error, not a runtime surprise.

## Download and Install

### Binary Distributions

Official binary distributions will be available at a future date.

### Install From Source

```bash
git clone https://github.com/mcabezas/archlang.git
cd archlang
go build -o archlang .
```

## Usage

```bash
# Compile .arch files and start the API server
archlang .

# Compile to a standalone binary
archlang build .
```

### API Endpoints

```
GET /components                    → List all components
GET /components/{name}             → Component details
GET /components/{name}/upstreams   → Who depends on this component
GET /components/{name}/downstreams → What this component depends on
```

## Agents

The [`agents/`](agents/) directory contains official agent implementations
that wrap the Architecture Knowledge Engine. These agents provide natural
language interfaces while guaranteeing that all data comes from the compiled
architecture definition.

Planned agents:

- **MCP Server** — Model Context Protocol server for AI coding assistants
- **CLI Agent** — Natural language queries from the terminal
- **Slack Bot** — Architecture queries in Slack channels

## Contributing

ArchLang is under active development. Contributions are welcome.

To contribute, please open an issue or pull request on this repository.

## Reporting Issues

The issue tracker is for bug reports and feature proposals only.
For questions about ArchLang, please use the Discussions tab.
