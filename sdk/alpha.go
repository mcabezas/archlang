package knowledge

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/mcabezas/archlang/graph"
)

type graphNodeJSON struct {
	ID       string `json:"id"`
	Kind     string `json:"kind"`
	Platform string `json:"platform,omitempty"`
	Org      string `json:"org,omitempty"`
}

type graphLinkJSON struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Label  string `json:"label,omitempty"`
	Type   string `json:"type"` // "call" | "publishes" | "listens"
}

type graphJSON struct {
	Nodes []graphNodeJSON `json:"nodes"`
	Links []graphLinkJSON `json:"links"`
}

func buildGraph(components []graph.Component) graphJSON {
	seenNodes := make(map[string]bool)
	seenLinks := make(map[string]bool)
	var nodes []graphNodeJSON
	var links []graphLinkJSON

	addNode := func(c graph.Component) {
		if seenNodes[c.Name()] {
			return
		}
		seenNodes[c.Name()] = true
		n := graphNodeJSON{ID: c.Name(), Kind: string(c.Kind()), Org: string(c.Org())}
		if svc, ok := c.(*graph.Service); ok {
			n.Platform = svc.Platform
		}
		nodes = append(nodes, n)
	}

	addLink := func(from, to, label, typ string) {
		key := from + "->" + to + ":" + label
		if seenLinks[key] {
			return
		}
		seenLinks[key] = true
		links = append(links, graphLinkJSON{Source: from, Target: to, Label: label, Type: typ})
	}

	for _, c := range components {
		for _, col := range c.Collaborations() {
			if col.Target.Kind() == graph.KindEvent {
				if ev, ok := col.Target.(*graph.Event); ok && ev.MessageBroker() != nil {
					mb := ev.MessageBroker()
					addNode(c)
					addNode(mb)
					addLink(c.Name(), mb.Name(), col.Target.Name(), "publishes")
				}
			} else if col.Source.Kind() == graph.KindEvent {
				if col.DeliveredBy != nil {
					addNode(col.DeliveredBy)
					addNode(col.Target)
					addLink(col.DeliveredBy.Name(), col.Target.Name(), col.Source.Name(), "listens")
				}
			} else {
				addNode(c)
				addNode(col.Target)
				addLink(c.Name(), col.Target.Name(), "", "call")
			}
		}
	}

	return graphJSON{Nodes: nodes, Links: links}
}

func (s *HTTPServer) handleGraph(w http.ResponseWriter, r *http.Request) {
	feature := r.URL.Query().Get("feature")
	event := r.URL.Query().Get("event")

	var components []graph.Component
	var err error

	switch {
	case feature != "":
		components, err = s.storage.FindByFeature(feature)
	case event != "":
		components, err = s.storage.FindEvent(event)
	default:
		components, err = s.storage.ListAll()
	}

	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(buildGraph(components))
}

func (s *HTTPServer) handleAlpha(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprint(w, alphaHTML)
}

const alphaHTML = `<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <title>Architecture · Alpha</title>
  <style>
    * { margin: 0; padding: 0; box-sizing: border-box; }
    body { background: #000; overflow: hidden; font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; }
    #graph { width: 100vw; height: 100vh; }

    .hud { position: fixed; top: 1.5rem; left: 1.5rem; z-index: 10; display: flex; flex-direction: column; gap: 1rem; }

    .filter-row { display: flex; align-items: center; gap: 0.75rem; }
    .filter-row label { font-size: 0.75rem; color: #38bdf8; text-transform: uppercase; letter-spacing: 0.1em; }
    .filter-row select {
      background: rgba(2, 6, 23, 0.75);
      color: #e2e8f0;
      border: 1px solid rgba(56, 189, 248, 0.35);
      border-radius: 6px;
      padding: 0.4rem 0.8rem;
      font-size: 0.875rem;
      backdrop-filter: blur(12px);
      cursor: pointer;
      outline: none;
      transition: border-color 0.2s;
    }
    .filter-row select:focus, .filter-row select:hover { border-color: #38bdf8; }
    .filter-row select:disabled { opacity: 0.3; cursor: not-allowed; }

    .legend { display: flex; flex-direction: column; gap: 0.4rem; }
    .legend-item { display: flex; align-items: center; gap: 0.5rem; font-size: 0.75rem; color: #94a3b8; }
    .legend-dot { width: 10px; height: 10px; border-radius: 50%; flex-shrink: 0; }

  </style>
</head>
<body>
  <div class="hud">
    <div class="filter-row">
      <label>View</label>
      <select id="filter-type" onchange="onTypeChange()">
        <option value="">All</option>
        <option value="feature">Feature</option>
        <option value="event">Event</option>
      </select>
      <select id="filter-value" onchange="onValueChange()" disabled>
        <option value="">— select —</option>
      </select>
    </div>
    <div class="legend">
      <div class="legend-item"><div class="legend-dot" style="background:#38bdf8;box-shadow:0 0 6px #38bdf8"></div> Service</div>
      <div class="legend-item"><div class="legend-dot" style="background:#a78bfa;box-shadow:0 0 6px #a78bfa"></div> Message Broker</div>
      <div class="legend-item" style="color:#fde047"><div class="legend-dot" style="background:#fde047;box-shadow:0 0 6px #fde047"></div> Event flow</div>
      <div class="legend-item"><div class="legend-dot" style="background:#475569"></div> Direct call</div>
    </div>
  </div>
  <div id="graph"></div>
  <script src="https://unpkg.com/three@0.137.0/build/three.min.js"></script>
  <script src="https://unpkg.com/three-spritetext@1.6.4/dist/three-spritetext.min.js"></script>
  <script src="https://unpkg.com/3d-force-graph@1.73.3/dist/3d-force-graph.min.js"></script>
  <script>
    const NODE_COLORS = {
      service: '#38bdf8',
      message_broker: '#a78bfa',
    };
    const LINK_COLORS = {
      call: '#475569',
      publishes: '#fde047',
      listens: '#fde047',
    };

    let Graph;

    function nodeColor(n) { return NODE_COLORS[n.kind] || '#64748b'; }
    function linkColor(l) { return LINK_COLORS[l.type] || '#475569'; }

    function initGraph(data) {
      if (Graph) {
        Graph.graphData(data);
        return;
      }
      Graph = ForceGraph3D()(document.getElementById('graph'))
        .backgroundColor('#000008')
        .nodeId('id')
        .nodeThreeObject(node => {
          const sprite = new SpriteText(node.id + (node.platform ? '\n' + node.platform : ''));
          sprite.color = nodeColor(node);
          sprite.textHeight = 6;
          sprite.backgroundColor = 'rgba(0,0,8,0.65)';
          sprite.padding = 2;
          sprite.borderRadius = 3;
          return sprite;
        })
        .nodeThreeObjectExtend(false)
        .onNodeClick(node => {
          Graph.cameraPosition(
            { x: node.x * 1.5, y: node.y * 1.5, z: node.z * 1.5 },
            { x: node.x, y: node.y, z: node.z },
            800
          );
        })
        .linkColor(linkColor)
        .linkWidth(l => l.type === 'call' ? 1 : 0.5)
        .linkOpacity(0.5)
        .linkDirectionalParticles(l => l.type !== 'call' ? 4 : 0)
        .linkDirectionalParticleColor(() => '#fde047')
        .linkDirectionalParticleWidth(2.5)
        .linkDirectionalParticleSpeed(0.005)
        .linkLabel(l => l.label ? '[' + l.label + ']' : '')
        .graphData(data);
    }


    fetch('/api/graph').then(r => r.json()).then(initGraph);

    async function onTypeChange() {
      const type = document.getElementById('filter-type').value;
      const sel = document.getElementById('filter-value');
      sel.innerHTML = '<option value="">— select —</option>';
      if (!type) {
        sel.disabled = true;
        fetch('/api/graph').then(r => r.json()).then(initGraph);
        return;
      }
      const res = await fetch('/api/' + type + 's');
      const data = await res.json();
      data.forEach(item => {
        const opt = document.createElement('option');
        opt.value = item.name;
        opt.textContent = item.name + (item.description ? '  —  ' + item.description : '');
        sel.appendChild(opt);
      });
      sel.disabled = false;
    }

    function onValueChange() {
      const type = document.getElementById('filter-type').value;
      const value = document.getElementById('filter-value').value;
      if (!value) return;
      fetch('/api/graph?' + type + '=' + encodeURIComponent(value))
        .then(r => r.json())
        .then(initGraph);
    }
  </script>
</body>
</html>`

