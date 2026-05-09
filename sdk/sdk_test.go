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
	if len(components) != 16 {
		t.Fatalf("expected 16 components, got %d", len(components))
	}

	services := arch.Services()
	if len(services) != 8 {
		t.Fatalf("expected 8 services, got %d", len(services))
	}
}

func TestGetComponent(t *testing.T) {
	arch, err := Compile("../internal/examples/ecommerce")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	payments, ok := arch.GetComponent("payments.payment-processing")
	if !ok {
		t.Fatal("expected 'payments.payment-processing' to exist")
	}
	if payments.Kind != KindService {
		t.Fatalf("expected kind service, got %s", payments.Kind)
	}
	if payments.Package != "payments" {
		t.Fatalf("expected package payments, got %s", payments.Package)
	}

	redis, ok := arch.GetComponent("users.users-db")
	if !ok {
		t.Fatal("expected 'users.users-db' to exist")
	}
	if redis.Kind != KindComponent {
		t.Fatalf("expected kind component, got %s", redis.Kind)
	}
}

func TestCrossPackageCollaboration(t *testing.T) {
	arch, err := Compile("../internal/examples/ecommerce")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	orderMgmt, _ := arch.GetComponent("orders.order-management")

	if len(orderMgmt.Downstreams) == 0 {
		t.Fatal("expected order-management to have downstreams")
	}

	// Should have cross-package downstreams
	var hasPayments, hasNotifications bool
	for _, ds := range orderMgmt.Downstreams {
		if ds.Package == "payments" && ds.Name == "payment-processing" {
			hasPayments = true
		}
		if ds.Package == "notifications" && ds.Name == "email" {
			hasNotifications = true
		}
	}

	if !hasPayments {
		t.Fatal("expected order-management to have payments.payment-processing as downstream")
	}
	if !hasNotifications {
		t.Fatal("expected order-management to have notifications.email as downstream")
	}
}

func TestPackages(t *testing.T) {
	arch, err := Compile("../internal/examples/ecommerce")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	packages := arch.Packages()
	if len(packages) != 7 {
		t.Fatalf("expected 7 packages, got %d: %v", len(packages), packages)
	}
}

func TestPointerIdentity(t *testing.T) {
	arch, err := Compile("../internal/examples/ecommerce")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	orderMgmt, _ := arch.GetComponent("orders.order-management")
	paymentProc, _ := arch.GetComponent("payments.payment-processing")

	var found *Component
	for _, ds := range orderMgmt.Downstreams {
		if ds.Name == "payment-processing" {
			found = ds
			break
		}
	}

	if found != paymentProc {
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

func TestCompileBadPath(t *testing.T) {
	_, err := Compile("/nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent directory")
	}
}
