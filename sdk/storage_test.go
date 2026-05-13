package knowledge

import (
	"errors"
	"testing"

	"github.com/mcabezas/archlang/graph"
)

func TestFindByName(t *testing.T) {
	// Seeds
	seed := graph.NewGraph()

	orders := graph.Domain("orders")
	orderMgmt := graph.NewService(graph.WithName("order-management"), graph.WithDomain(orders))
	ordersDB := graph.NewInfra(graph.WithName("orders-db"), graph.WithDomain(orders))
	payments := graph.Domain("payments")
	paymentProc := graph.NewService(graph.WithName("payment-processing"), graph.WithDomain(payments))

	seed.Register("orders.order-management", orderMgmt)
	seed.Register("orders.orders-db", ordersDB)
	seed.Register("payments.payment-processing", paymentProc)

	seed.AddDownstream(orderMgmt, ordersDB)
	seed.AddDownstream(orderMgmt, paymentProc)

	// test cases
	testCases := map[string]struct {
		graphs      []*graph.Graph
		name        string
		expectedErr error
	}{
		"not found with no graphs": {
			graphs:      []*graph.Graph{},
			name:        "orders.order-management",
			expectedErr: ErrNotFound,
		},
		"not found with graphs": {
			graphs:      []*graph.Graph{seed},
			name:        "nonexistent",
			expectedErr: ErrNotFound,
		},
		"found service": {
			graphs: []*graph.Graph{seed},
			name:   "orders.order-management",
		},
		"found infra": {
			graphs: []*graph.Graph{seed},
			name:   "orders.orders-db",
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			storage := New(tc.graphs)
			node, err := storage.FindByName(tc.name)

			if tc.expectedErr != nil {
				if !errors.Is(err, tc.expectedErr) {
					t.Fatalf("expected error %v, got %v", tc.expectedErr, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if node == nil {
				t.Fatal("expected node, got nil")
			}
		})
	}
}
