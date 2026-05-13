package knowledge

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/mcabezas/archlang/graph"
)

// MCPServer exposes the architecture knowledge graph via the Model Context Protocol.
type MCPServer struct {
	storage Storage
	server  *server.MCPServer
}

const agentInstructions = `You are an Architecture Solution Assistant powered by ArchLang — a compiled, deterministic knowledge graph of the system architecture.

## Core Principle

Every answer you give MUST come from the compiled architecture graph. You NEVER guess, assume, or hallucinate architectural facts. If the graph doesn't contain the answer, say so explicitly.

## How to Answer Questions

### "What depends on X?" or "What calls X?"
1. Call get_component with the component name
2. Look at the "Upstream" section — those are the callers
3. Present the list with descriptions and features

### "What does X depend on?" or "What does X call?"
1. Call get_component with the component name
2. Look at the "Downstream" section — those are the dependencies
3. Present the list with descriptions and features

### "What would break if we change X?"
1. Call analyze_impact with the component name
2. Present affected features, flows, upstream dependents, and testing recommendations
3. Be specific about what integrations are at risk

### "How does feature Y work?" or "Trace feature Y"
1. Call trace_feature with the feature name
2. Walk through the flows and steps in order
3. Explain the end-to-end journey across services

### "How does flow Z work?" or "Trace flow Z"
1. Call trace_flow with the flow name
2. Present steps in order with their collaborations
3. Explain the sequence of operations

### "What features exist?" or "What does the system do?"
1. Call list_features to get all business capabilities
2. Summarize each feature and what it represents

### "What services exist?" or "Show me the architecture"
1. Call list_components to get all components
2. Group by organization and present with their kinds and visibility

### "Can service A talk to service B?"
1. Call get_component for both services
2. Check if there is a direct collaboration between them
3. If not, check if there is a transitive path through shared dependencies

## Response Guidelines

- Lead with the answer, then provide supporting detail from the graph
- When listing collaborations, include: source → target, description, feature, flow, step, and cardinality
- When discussing impact, always mention affected features — those represent business capabilities at risk
- Use step order to explain sequences chronologically
- If a component has both upstream and downstream dependencies, present both — the full picture matters
- When asked about organizational boundaries, note which components are public vs internal
- Never say "I think" or "probably" — either the graph says it or it doesn't
- If a component is not found, suggest using list_components to find the correct name`

// NewMCPServer creates a new MCP server wrapping the architecture knowledge graph.
func NewMCPServer(graphs []*graph.Graph) *MCPServer {
	s := &MCPServer{
		storage: New(graphs),
		server:  server.NewMCPServer("archlang", "0.0.3", server.WithInstructions(agentInstructions)),
	}
	s.registerTools()
	return s
}

// ServeStdio starts the MCP server on stdin/stdout.
func (s *MCPServer) ServeStdio() error {
	return server.ServeStdio(s.server)
}

// ServeSSE starts the MCP server as a remote SSE (Server-Sent Events) server.
// Multiple clients can connect simultaneously over HTTP.
// Handles graceful shutdown on SIGINT/SIGTERM.
func (s *MCPServer) ServeSSE(addr string) error {
	sse := server.NewSSEServer(s.server)
	return s.serveHTTP(sse, addr)
}

// ServeStreamableHTTP starts the MCP server as a remote Streamable HTTP server.
// This is the newer MCP HTTP transport. Multiple clients can connect simultaneously.
// Handles graceful shutdown on SIGINT/SIGTERM.
func (s *MCPServer) ServeStreamableHTTP(addr string) error {
	sh := server.NewStreamableHTTPServer(s.server)
	return s.serveHTTP(sh, addr)
}

type startShutdowner interface {
	Start(addr string) error
	Shutdown(ctx context.Context) error
}

func (s *MCPServer) serveHTTP(srv startShutdowner, addr string) error {
	errCh := make(chan error, 1)
	go func() {
		fmt.Printf("mcp server listening on %s\n", addr)
		errCh <- srv.Start(addr)
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-quit:
		fmt.Printf("\nreceived %s, shutting down...\n", sig)
	case err := <-errCh:
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		return fmt.Errorf("shutdown error: %w", err)
	}
	fmt.Println("mcp server stopped")
	return nil
}

func (s *MCPServer) registerTools() {
	s.server.AddTool(
		mcp.NewTool("list_components",
			mcp.WithDescription("List all components in the architecture graph. Returns name, kind (service/component/infra), organization, domain, and visibility for each component."),
		),
		s.handleListComponents,
	)

	s.server.AddTool(
		mcp.NewTool("get_component",
			mcp.WithDescription("Get detailed information about a specific component including all its upstream and downstream collaborations. Use the component name (e.g. 'payment-service') or qualified name (e.g. 'orgs/myteam.payment-service')."),
			mcp.WithString("name", mcp.Required(), mcp.Description("Component name or qualified name")),
		),
		s.handleGetComponent,
	)

	s.server.AddTool(
		mcp.NewTool("list_features",
			mcp.WithDescription("List all declared features in the architecture. Each feature represents a business capability that spans across services."),
		),
		s.handleListFeatures,
	)

	s.server.AddTool(
		mcp.NewTool("trace_feature",
			mcp.WithDescription("Trace a feature across the entire architecture. Returns all collaborations involved in a feature, organized by flow and step. Use this to understand how a business capability flows through the system."),
			mcp.WithString("name", mcp.Required(), mcp.Description("Feature name to trace")),
		),
		s.handleTraceFeature,
	)

	s.server.AddTool(
		mcp.NewTool("list_flows",
			mcp.WithDescription("List all declared flows in the architecture. Flows group collaborations into named sequences within a feature."),
		),
		s.handleListFlows,
	)

	s.server.AddTool(
		mcp.NewTool("trace_flow",
			mcp.WithDescription("Trace a flow across the architecture. Returns all collaborations in the flow organized by step order. Use this to understand a specific sequence of operations."),
			mcp.WithString("name", mcp.Required(), mcp.Description("Flow name to trace")),
		),
		s.handleTraceFlow,
	)

	s.server.AddTool(
		mcp.NewTool("analyze_impact",
			mcp.WithDescription("Analyze the impact of changing a component. Returns all features, flows, and collaborations that involve this component — both as source and target. Use this to assess what would break, what needs testing, and what downstream effects a change would have."),
			mcp.WithString("name", mcp.Required(), mcp.Description("Component name to analyze")),
		),
		s.handleAnalyzeImpact,
	)
}

func (s *MCPServer) handleListComponents(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	components, err := s.storage.ListAll()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	var sb strings.Builder
	sb.WriteString("# Architecture Components\n\n")
	for _, c := range components {
		fmt.Fprintf(&sb, "- **%s** (%s) — org: %s, domain: %s, visibility: %s\n",
			c.Name(), kindOf(c), c.Org(), c.Domain(), c.Visibility())
	}
	return mcp.NewToolResultText(sb.String()), nil
}

func (s *MCPServer) handleGetComponent(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name := req.GetString("name", "")
	if name == "" {
		return mcp.NewToolResultError("name is required"), nil
	}

	c, err := s.storage.FindByName(name, WithNestedLevels(1), WithUpperLevels(1))
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("component %q not found", name)), nil
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "# Component: %s\n\n", c.Name())
	fmt.Fprintf(&sb, "- **Kind:** %s\n", kindOf(c))
	fmt.Fprintf(&sb, "- **Organization:** %s\n", c.Org())
	fmt.Fprintf(&sb, "- **Domain:** %s\n", c.Domain())
	fmt.Fprintf(&sb, "- **Visibility:** %s\n\n", c.Visibility())

	downstream, upstream := splitCollaborations(c)

	if len(downstream) > 0 {
		sb.WriteString("## Downstream (this component calls)\n\n")
		for _, col := range downstream {
			writeCollaboration(&sb, col.Target.Name(), col)
		}
	}

	if len(upstream) > 0 {
		sb.WriteString("## Upstream (calls this component)\n\n")
		for _, col := range upstream {
			writeCollaboration(&sb, col.Source.Name(), col)
		}
	}

	if len(downstream) == 0 && len(upstream) == 0 {
		sb.WriteString("_No collaborations found._\n")
	}

	return mcp.NewToolResultText(sb.String()), nil
}

func (s *MCPServer) handleListFeatures(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	features, err := s.storage.ListFeatures()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	var sb strings.Builder
	sb.WriteString("# Features\n\n")
	for _, f := range features {
		fmt.Fprintf(&sb, "- **%s** — %s\n", f.Name, f.Description)
	}
	if len(features) == 0 {
		sb.WriteString("_No features declared._\n")
	}
	return mcp.NewToolResultText(sb.String()), nil
}

func (s *MCPServer) handleTraceFeature(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name := req.GetString("name", "")
	if name == "" {
		return mcp.NewToolResultError("name is required"), nil
	}

	components, err := s.storage.FindByFeature(name)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("feature %q not found", name)), nil
	}

	// Collect all collaborations for this feature
	var collabs []graph.Collaboration
	for _, c := range components {
		for _, col := range c.Collaborations() {
			if col.Feature.Name == name {
				collabs = append(collabs, col)
			}
		}
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "# Feature: %s\n\n", name)
	if len(collabs) > 0 && collabs[0].Feature.Description != "" {
		fmt.Fprintf(&sb, "__%s__\n\n", collabs[0].Feature.Description)
	}

	// Group by flow then step
	type flowGroup struct {
		name        string
		description string
		collabs     []graph.Collaboration
	}
	var flows []flowGroup
	flowIdx := make(map[string]int)

	for _, c := range collabs {
		fn := c.Flow.Name
		if idx, ok := flowIdx[fn]; ok {
			flows[idx].collabs = append(flows[idx].collabs, c)
		} else {
			flowIdx[fn] = len(flows)
			flows = append(flows, flowGroup{name: fn, description: c.Flow.Description, collabs: []graph.Collaboration{c}})
		}
	}

	// Involved components
	seen := make(map[string]bool)
	var involved []string
	for _, c := range collabs {
		if !seen[c.Source.Name()] {
			seen[c.Source.Name()] = true
			involved = append(involved, c.Source.Name())
		}
		if !seen[c.Target.Name()] {
			seen[c.Target.Name()] = true
			involved = append(involved, c.Target.Name())
		}
	}
	fmt.Fprintf(&sb, "## Involved Components (%d)\n\n", len(involved))
	for _, name := range involved {
		fmt.Fprintf(&sb, "- %s\n", name)
	}
	sb.WriteString("\n")

	for _, flow := range flows {
		if flow.name != "" {
			fmt.Fprintf(&sb, "## Flow: %s\n\n", flow.name)
			if flow.description != "" {
				fmt.Fprintf(&sb, "_%s_\n\n", flow.description)
			}
		}
		for _, c := range flow.collabs {
			step := ""
			if c.Step != "" {
				step = fmt.Sprintf(" [step %d: %s]", c.StepOrder, c.Step)
			}
			fmt.Fprintf(&sb, "- **%s** → **%s**%s", c.Source.Name(), c.Target.Name(), step)
			if c.Description != "" {
				fmt.Fprintf(&sb, " — %s", c.Description)
			}
			if c.Cardinality != "" && c.Cardinality != "1:1" {
				fmt.Fprintf(&sb, " (%s", c.Cardinality)
				if c.CardinalityBy != "" {
					fmt.Fprintf(&sb, " by %s", c.CardinalityBy)
				}
				sb.WriteString(")")
			}
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	}

	return mcp.NewToolResultText(sb.String()), nil
}

func (s *MCPServer) handleListFlows(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	flows, err := s.storage.ListFlows()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	var sb strings.Builder
	sb.WriteString("# Flows\n\n")
	for _, f := range flows {
		fmt.Fprintf(&sb, "- **%s**", f.Name)
		if f.Description != "" {
			fmt.Fprintf(&sb, " — %s", f.Description)
		}
		sb.WriteString("\n")
	}
	if len(flows) == 0 {
		sb.WriteString("_No flows declared._\n")
	}
	return mcp.NewToolResultText(sb.String()), nil
}

func (s *MCPServer) handleTraceFlow(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name := req.GetString("name", "")
	if name == "" {
		return mcp.NewToolResultError("name is required"), nil
	}

	collabs, err := s.storage.FindByFlow(name)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("flow %q not found", name)), nil
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "# Flow: %s\n\n", name)
	if len(collabs) > 0 && collabs[0].Flow.Description != "" {
		fmt.Fprintf(&sb, "_%s_\n\n", collabs[0].Flow.Description)
	}

	// Group by step order
	type stepGroup struct {
		name    string
		order   int
		collabs []graph.Collaboration
	}
	var steps []stepGroup
	stepIdx := make(map[string]int)

	for _, c := range collabs {
		sn := c.Step
		if idx, ok := stepIdx[sn]; ok {
			steps[idx].collabs = append(steps[idx].collabs, c)
		} else {
			stepIdx[sn] = len(steps)
			steps = append(steps, stepGroup{name: sn, order: c.StepOrder, collabs: []graph.Collaboration{c}})
		}
	}

	for _, step := range steps {
		if step.name != "" {
			fmt.Fprintf(&sb, "## Step %d: %s\n\n", step.order, step.name)
		}
		for _, c := range step.collabs {
			fmt.Fprintf(&sb, "- **%s** → **%s**", c.Source.Name(), c.Target.Name())
			if c.Description != "" {
				fmt.Fprintf(&sb, " — %s", c.Description)
			}
			if c.Cardinality != "" && c.Cardinality != "1:1" {
				fmt.Fprintf(&sb, " (%s", c.Cardinality)
				if c.CardinalityBy != "" {
					fmt.Fprintf(&sb, " by %s", c.CardinalityBy)
				}
				sb.WriteString(")")
			}
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	}

	return mcp.NewToolResultText(sb.String()), nil
}

func (s *MCPServer) handleAnalyzeImpact(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name := req.GetString("name", "")
	if name == "" {
		return mcp.NewToolResultError("name is required"), nil
	}

	c, err := s.storage.FindByName(name, WithNestedLevels(Maximum), WithUpperLevels(Maximum))
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("component %q not found", name)), nil
	}

	downstream, upstream := splitCollaborations(c)

	// Collect affected features
	featureSet := make(map[string]string) // name -> description
	flowSet := make(map[string]string)    // name -> description
	for _, col := range c.Collaborations() {
		if col.Feature.Name != "" {
			featureSet[col.Feature.Name] = col.Feature.Description
		}
		if col.Flow.Name != "" {
			flowSet[col.Flow.Name] = col.Flow.Description
		}
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "# Impact Analysis: %s\n\n", c.Name())
	fmt.Fprintf(&sb, "- **Kind:** %s\n", kindOf(c))
	fmt.Fprintf(&sb, "- **Organization:** %s\n", c.Org())
	fmt.Fprintf(&sb, "- **Visibility:** %s\n\n", c.Visibility())

	// Affected features
	fmt.Fprintf(&sb, "## Affected Features (%d)\n\n", len(featureSet))
	if len(featureSet) > 0 {
		sb.WriteString("Changes to this component may impact the following business capabilities:\n\n")
		for name, desc := range featureSet {
			fmt.Fprintf(&sb, "- **%s** — %s\n", name, desc)
		}
	} else {
		sb.WriteString("_This component is not part of any declared feature._\n")
	}
	sb.WriteString("\n")

	// Affected flows
	fmt.Fprintf(&sb, "## Affected Flows (%d)\n\n", len(flowSet))
	if len(flowSet) > 0 {
		sb.WriteString("Changes to this component may disrupt the following flows:\n\n")
		for name, desc := range flowSet {
			fmt.Fprintf(&sb, "- **%s**", name)
			if desc != "" {
				fmt.Fprintf(&sb, " — %s", desc)
			}
			sb.WriteString("\n")
		}
	} else {
		sb.WriteString("_This component is not part of any declared flow._\n")
	}
	sb.WriteString("\n")

	// Downstream
	fmt.Fprintf(&sb, "## Downstream Dependencies (%d)\n\n", len(downstream))
	if len(downstream) > 0 {
		sb.WriteString("This component calls the following services. Changes here may require updates to these integrations:\n\n")
		for _, col := range downstream {
			writeCollaboration(&sb, col.Target.Name(), col)
		}
	} else {
		sb.WriteString("_No downstream dependencies._\n\n")
	}

	// Upstream
	fmt.Fprintf(&sb, "## Upstream Dependents (%d)\n\n", len(upstream))
	if len(upstream) > 0 {
		sb.WriteString("These services depend on this component. Changes may break their integrations:\n\n")
		for _, col := range upstream {
			writeCollaboration(&sb, col.Source.Name(), col)
		}
	} else {
		sb.WriteString("_No upstream dependents._\n\n")
	}

	// Testing recommendations
	sb.WriteString("## Testing Recommendations\n\n")
	if len(featureSet) > 0 {
		sb.WriteString("Verify the following features still work end-to-end:\n\n")
		for name := range featureSet {
			fmt.Fprintf(&sb, "- [ ] Feature: %s\n", name)
		}
		sb.WriteString("\n")
	}
	if len(upstream) > 0 {
		sb.WriteString("Verify upstream integrations:\n\n")
		seen := make(map[string]bool)
		for _, col := range upstream {
			if !seen[col.Source.Name()] {
				seen[col.Source.Name()] = true
				fmt.Fprintf(&sb, "- [ ] %s can still reach %s\n", col.Source.Name(), c.Name())
			}
		}
		sb.WriteString("\n")
	}
	if len(downstream) > 0 {
		sb.WriteString("Verify downstream integrations:\n\n")
		seen := make(map[string]bool)
		for _, col := range downstream {
			if !seen[col.Target.Name()] {
				seen[col.Target.Name()] = true
				fmt.Fprintf(&sb, "- [ ] %s can still reach %s\n", c.Name(), col.Target.Name())
			}
		}
	}

	return mcp.NewToolResultText(sb.String()), nil
}

// splitCollaborations separates collaborations into downstream (component is source)
// and upstream (component is target).
func splitCollaborations(c graph.Component) (downstream, upstream []graph.Collaboration) {
	for _, col := range c.Collaborations() {
		if col.Source.Name() == c.Name() {
			downstream = append(downstream, col)
		} else {
			upstream = append(upstream, col)
		}
	}
	return
}

func writeCollaboration(sb *strings.Builder, peerName string, col graph.Collaboration) {
	fmt.Fprintf(sb, "- **%s**", peerName)
	if col.Feature.Name != "" {
		fmt.Fprintf(sb, " [feature: %s]", col.Feature.Name)
	}
	if col.Flow.Name != "" {
		fmt.Fprintf(sb, " [flow: %s]", col.Flow.Name)
	}
	if col.Description != "" {
		fmt.Fprintf(sb, " — %s", col.Description)
	}
	if col.Cardinality != "" && col.Cardinality != "1:1" {
		fmt.Fprintf(sb, " (%s", col.Cardinality)
		if col.CardinalityBy != "" {
			fmt.Fprintf(sb, " by %s", col.CardinalityBy)
		}
		sb.WriteString(")")
	}
	sb.WriteString("\n")
}

// mcpJSON is a helper to return JSON-encoded tool results.
func mcpJSON(v any) (*mcp.CallToolResult, error) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText(string(data)), nil
}
