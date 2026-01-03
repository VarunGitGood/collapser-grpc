package collapser

import "testing"

func TestNewCollapser(t *testing.T) {
	c := NewCollapser()
	if c == nil {
		t.Fatal("NewCollapser() returned nil")
	}
}

func TestCollapser_Start(t *testing.T) {
	c := NewCollapser()
	err := c.Start()
	if err != nil {
		t.Errorf("Start() error = %v", err)
	}
}

func TestCollapser_Stop(t *testing.T) {
	c := NewCollapser()
	err := c.Stop()
	if err != nil {
		t.Errorf("Stop() error = %v", err)
	}
}
