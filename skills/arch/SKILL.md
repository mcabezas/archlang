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
   - If unsure what exists: call `mcp__archlang__list_features` or `mcp__archlang__list_components` first

3. **If the MCP server is not running**, tell the user to start it with `make mcp-up` and retry.

4. **Present the results:**
   - Summarize the feature/flow in plain language
   - Include a Mermaid sequence diagram showing the interactions
   - Group by flow and step order when tracing features
   - Include cardinality and descriptions for each collaboration

5. **If "open" is specified**, generate the Mermaid diagram and open it in the browser using mermaid.live.

6. **If invoked via /copy**, copy the Mermaid diagram markdown to the clipboard.

### Mermaid Diagram Format

When generating sequence diagrams from trace results, use this format:

```
sequenceDiagram
    participant A as service-a
    participant B as service-b
    participant C as service-c

    Note over A,C: Flow: flow-name
    A->>B: description
    B->>C: description
```

- Use `participant` aliases for readability
- Add `Note over` to separate flows
- Use `->>` for synchronous calls
- Include step descriptions as arrow labels
- For one-to-many cardinality, annotate with `loop for each <cardinality_by>`

### Rules

- NEVER invent or assume architectural data — only use what the MCP tools return
- If a feature or component is not found, suggest using list_features or list_components to find the correct name
- Always mention which features and flows are involved
- When discussing impact, always mention affected business capabilities
