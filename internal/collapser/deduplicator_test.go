package collapser

import (
	"context"
	"testing"
)

func TestNewRequestDeduplicator(t *testing.T) {
	rd := NewRequestDeduplicator()
	if rd == nil {
		t.Fatal("NewRequestDeduplicator() returned nil")
	}
	if rd.inflight == nil {
		t.Error("inflight map not initialized")
	}
}

func TestRequestDeduplicator_Execute(t *testing.T) {
	rd := NewRequestDeduplicator()
	ctx := context.Background()

	called := 0
	fn := func() (interface{}, error) {
		called++
		return "result", nil
	}

	result, err := rd.Execute(ctx, "test-key", fn)
	if err != nil {
		t.Errorf("Execute() error = %v", err)
	}
	if result != "result" {
		t.Errorf("Execute() result = %v, want %v", result, "result")
	}
	if called != 1 {
		t.Errorf("Function called %d times, want 1", called)
	}
}
