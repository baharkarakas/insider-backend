package worker

import (
	"sync"
)

type task func()

type Pool struct {
	wg   sync.WaitGroup
	jobs chan task
}

func NewPool(n int) *Pool {
	p := &Pool{jobs: make(chan task, 1024)}
	for i := 0; i < n; i++ {
		p.wg.Add(1)
		go func() {
			defer p.wg.Done()
			for job := range p.jobs { job() }
		}()
	}
	return p
}

func (p *Pool) Submit(f task) { p.jobs <- f }
func (p *Pool) Stop() { close(p.jobs); p.wg.Wait() }