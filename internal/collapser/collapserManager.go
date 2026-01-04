package collapser

import "time"

type Collapser struct {
	CollapseWindow time.Duration
}

func NewCollapser(collapseWindow time.Duration) *Collapser {
	return &Collapser{
		CollapseWindow: collapseWindow,
	}
}

func (c *Collapser) Start() error {
	// TODO: Implement service startup logic
	return nil
}

func (c *Collapser) Stop() error {
	// TODO: Implement graceful shutdown
	return nil
}
