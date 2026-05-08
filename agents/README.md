# ArchLang Agents

This directory contains official agent implementations that wrap the ArchLang Architecture Knowledge Engine.

## Purpose

Agents provide natural language interfaces to the deterministic REST API. They translate human or AI questions into API calls and format the responses.

**Key guarantee:** Agents may paraphrase or summarize, but they never invent data. Every fact in an agent's response comes directly from the engine's API.

## Planned Agents

- **MCP Server** — Model Context Protocol server for AI coding assistants (Claude Code, Cursor, etc.)
- **CLI Agent** — Natural language queries from the terminal
- **Slack Bot** — Architecture queries in Slack channels
