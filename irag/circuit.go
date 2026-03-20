package irag

import (
	"sync"
	"time"
)

type circuitState struct {
	failures  int
	openUntil time.Time
}

type CircuitBreaker struct {
	mu            sync.Mutex
	failThreshold int
	cooldown      time.Duration
	state         map[ProviderName]*circuitState
}

func NewCircuitBreaker(failThreshold int, cooldown time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		failThreshold: failThreshold,
		cooldown:      cooldown,
		state:         make(map[ProviderName]*circuitState),
	}
}

func (c *CircuitBreaker) Allow(name ProviderName) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	state := c.ensure(name)
	if state.openUntil.IsZero() {
		return true
	}
	if time.Now().After(state.openUntil) {
		state.openUntil = time.Time{}
		state.failures = 0
		return true
	}
	return false
}

func (c *CircuitBreaker) Success(name ProviderName) {
	c.mu.Lock()
	defer c.mu.Unlock()

	state := c.ensure(name)
	state.failures = 0
	state.openUntil = time.Time{}
}

func (c *CircuitBreaker) Failure(name ProviderName) {
	c.mu.Lock()
	defer c.mu.Unlock()

	state := c.ensure(name)
	state.failures++
	if state.failures >= c.failThreshold {
		state.openUntil = time.Now().Add(c.cooldown)
	}
}

func (c *CircuitBreaker) ensure(name ProviderName) *circuitState {
	state, ok := c.state[name]
	if !ok {
		state = &circuitState{}
		c.state[name] = state
	}
	return state
}
