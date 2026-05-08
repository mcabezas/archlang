package sdk

import (
	"testing"
)

func TestCompile(t *testing.T) {
	arch, err := Compile("../internal/examples/ecommerce")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	components := arch.Components()
	if len(components) != 9 {
		t.Fatalf("expected 9 components, got %d", len(components))
	}

	services := arch.Services()
	if len(services) != 6 {
		t.Fatalf("expected 6 services, got %d", len(services))
	}
}

func TestGetComponent(t *testing.T) {
	arch, err := Compile("../internal/examples/ecommerce")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	payments, ok := arch.GetComponent("payments")
	if !ok {
		t.Fatal("expected 'payments' to exist")
	}
	if payments.Kind != KindService {
		t.Fatalf("expected kind service, got %s", payments.Kind)
	}

	redis, ok := arch.GetComponent("redis")
	if !ok {
		t.Fatal("expected 'redis' to exist")
	}
	if redis.Kind != KindComponent {
		t.Fatalf("expected kind component, got %s", redis.Kind)
	}
}

func TestDownstreamsAreComponents(t *testing.T) {
	arch, err := Compile("../internal/examples/ecommerce")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	payments, _ := arch.GetComponent("payments")

	if len(payments.Downstreams) == 0 {
		t.Fatal("expected payments to have downstreams")
	}

	for _, ds := range payments.Downstreams {
		if ds.Name == "" {
			t.Fatal("downstream component has empty name")
		}
		if ds.Kind == "" {
			t.Fatal("downstream component has empty kind")
		}
	}
}

func TestUpstreamsAreComponents(t *testing.T) {
	arch, err := Compile("../internal/examples/ecommerce")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	redis, _ := arch.GetComponent("redis")

	if len(redis.Upstreams) == 0 {
		t.Fatal("expected redis to have upstreams")
	}

	for _, us := range redis.Upstreams {
		if us.Name == "" {
			t.Fatal("upstream component has empty name")
		}
	}
}

func TestCircularReferences(t *testing.T) {
	arch, err := Compile("../internal/examples/ecommerce")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	checkout, _ := arch.GetComponent("checkout")
	payments, _ := arch.GetComponent("payments")

	var found *Component
	for _, ds := range checkout.Downstreams {
		if ds.Name == "payments" {
			found = ds
			break
		}
	}

	if found != payments {
		t.Fatal("expected downstream pointer to be the same object as GetComponent result")
	}
}

func TestGetComponentNotFound(t *testing.T) {
	arch, err := Compile("../internal/examples/ecommerce")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	_, ok := arch.GetComponent("nonexistent")
	if ok {
		t.Fatal("expected 'nonexistent' to not exist")
	}
}

func TestCompileNoFiles(t *testing.T) {
	_, err := Compile("../doc")
	if err == nil {
		t.Fatal("expected error for directory with no .arch files")
	}
}

func TestCompileBadPath(t *testing.T) {
	_, err := Compile("/nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent directory")
	}
}
