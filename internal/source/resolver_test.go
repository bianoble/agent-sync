package source

import (
	"fmt"
	"strings"
	"testing"
)

func TestRegistryGetUnknown(t *testing.T) {
	reg := NewRegistry()
	_, err := reg.Get("nonexistent")
	if err == nil {
		t.Fatal("expected error for unknown type")
	}
	if !strings.Contains(err.Error(), "unknown source type 'nonexistent'") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRegistryRegisterAndGet(t *testing.T) {
	reg := NewRegistry()
	reg.Register("test", nil)
	_, err := reg.Get("test")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestSourceErrorFormat(t *testing.T) {
	err := &SourceError{
		Source:    "my-source",
		Operation: "resolve",
		Err:      fmt.Errorf("connection refused"),
		Hint:     "check network connectivity",
	}
	msg := err.Error()
	if !strings.Contains(msg, "my-source") {
		t.Errorf("missing source name: %s", msg)
	}
	if !strings.Contains(msg, "resolve") {
		t.Errorf("missing operation: %s", msg)
	}
	if !strings.Contains(msg, "connection refused") {
		t.Errorf("missing error detail: %s", msg)
	}
	if !strings.Contains(msg, "check network connectivity") {
		t.Errorf("missing hint: %s", msg)
	}
}

func TestSourceErrorUnwrap(t *testing.T) {
	inner := fmt.Errorf("inner error")
	err := &SourceError{Source: "s", Operation: "fetch", Err: inner}
	if err.Unwrap() != inner {
		t.Error("Unwrap should return inner error")
	}
}
