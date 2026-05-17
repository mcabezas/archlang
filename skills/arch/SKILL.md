# arch

Generate architecture documentation with Mermaid diagrams for a feature.

## Usage

/arch <feature-name> — trace and explain a feature
/arch <feature-name> open — trace and open the Mermaid diagram in the browser
/copy arch <feature-name> — copy the Mermaid diagram to clipboard

## Instructions

When this skill is invoked, you MUST use the archlang MCP tools to answer. NEVER search through source files, memory, or guess architectural facts. Every fact must come from the MCP tools.

### Steps

1. **Identify what the user wants.** Parse the argument to determine the feature, flow, or component name.

2. **Use the right MCP tool:**
   - For a feature overview: call `mcp__archlang__trace_feature` with the feature name
   - For a specific flow: call `mcp__archlang__trace_flow` with the flow name
   - For component details or dependencies: call `mcp__archlang__get_component` with the component name
   - For impact analysis: call `mcp__archlang__analyze_impact` with the component name
   - For an event — who publishes it, who subscribes, which broker: call `mcp__archlang__trace_event` with the event name
   - If unsure what exists: call `mcp__archlang__list_features`, `mcp__archlang__list_components`, or `mcp__archlang__list_events` first

3. **If the MCP server is not running**, tell the user to start it with `make mcp-up` and retry.

4. **Present the results:**
   - Summarize the feature/flow in plain language
   - Include a Mermaid sequence diagram showing the interactions
   - Group by flow and step order when tracing features
   - Include cardinality and descriptions for each collaboration

5. **If "open" is specified**, generate the Mermaid diagram and open it in the browser using mermaid.live.

6. **If invoked via /copy**, copy the Mermaid diagram markdown to the clipboard.

### Understanding the Architecture Model

Every component returned by the MCP tools has a `kind` field. Use it to understand what you are looking at:

#### `service` — where business logic lives
- Business logic, domain rules, and application code live here — nowhere else
- Has a `repository_url` field. **When you need to understand implementation details, navigate to this repository.** It is the source of truth for what a service actually does
- Has an optional `platform` field (e.g. `Lambda`, `EKS`, `ECS`, `Fargate`) describing the compute environment it runs on
- Collaborations between services use solid arrows (`-->`) in diagrams

#### `event` — a fact that happened
- Represents a domain fact: something that occurred in the system
- Has a `published_at` field identifying which message broker it is published to by default
- When a service subscribes to an event, the collaboration may have a `delivered_by` field overriding the default broker for that specific subscription
- If `delivered_by` is absent on a subscribe collaboration, the event's `published_at` broker is the effective delivery channel
- Events use dotted arrows (`-.->`) in diagrams to distinguish them from direct service calls

#### `message_broker` — infrastructure, not logic
- Infrastructure component. No business logic lives here
- Has a `technology` field (e.g. `RabbitMQ`, `Kafka`, `EventBridge`) and a `cloud` field (e.g. `AWS`, `GCP`, `Azure`)
- `message_broker.Kind().IsInfra()` is true — treat it as infra, not as a service to investigate for business logic
- Do NOT look for repository URLs on message brokers — they have none

#### `component` — generic
- A generic architectural component without a more specific classification

### Querying Events

Events are first-class citizens. Use these tools to understand event flows:

- `mcp__archlang__list_events` — lists all events with their broker and cloud info
- `mcp__archlang__trace_event` — given an event name, returns:
  - The broker it is published to (`published_at`)
  - All **publishers**: services that have a collaboration where `target` is this event
  - All **subscribers**: services that have a collaboration where `source` is this event

When presenting event traces:
- Show the publisher → broker arrow labeled "publishes"
- Show the broker → subscriber arrow labeled "listen"
- Mention the `delivered_by` broker on subscribe collaborations when present

### Reading Collaborations

Each collaboration has:
- `source → target`: direction of the call or event flow
- `delivered_by`: for subscribe collaborations — the message broker that delivers the event (resolved at compile time, either explicit or inherited from the event's `published_at`)
- `execute`: the handler method invoked when the event arrives
- `feature`, `flow`, `step`: tracing metadata grouping collaborations into business capabilities and sequences

### When the User Asks About Implementation

If the user asks **how something works in code**, **where logic lives**, or **wants to see the implementation**:
1. Find the relevant `service` components from the MCP tools
2. Use the `repository_url` from each service to navigate to its repository
3. Make clear that business logic lives in services — events and message brokers carry no logic

### Mermaid Diagram Format

When generating sequence diagrams from trace results, use this format:

```
sequenceDiagram
    participant A as service-a
    participant B as service-b
    participant E as OrderPlaced

    Note over A,B: Flow: flow-name
    A->>E: publishes
    E-->>B: execute: handleOrder (via EventBus)
```

- Use `participant` aliases for readability
- Add `Note over` to separate flows
- Use `->>` for synchronous service-to-service calls
- Use `-->>` (dotted) for event-driven interactions
- Include `execute` handler name and `delivered_by` broker on subscribe arrows when present
- For one-to-many cardinality, annotate with `loop for each <cardinality_by>`

### Rules

- NEVER invent or assume architectural data — only use what the MCP tools return
- If a feature or component is not found, suggest using `list_features`, `list_components`, or `list_events` to find the correct name
- Always mention which features and flows are involved
- When discussing impact, always mention affected business capabilities
- Business logic lives in **services** — if you need to understand implementation, use the service's `repository_url`
- **Never confuse events or message brokers with services** — they carry no business logic
