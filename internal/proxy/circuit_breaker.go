package proxy

import (
	"sync"
	"time"
)

type CircuitBreaker struct {
	mu           sync.Mutex
	failures     map[uint]int
	openedUntil  map[uint]time.Time
	failureLimit int
	cooldown     time.Duration
}

func NewCircuitBreaker(failureLimit int, cooldown time.Duration) *CircuitBreaker {
	if failureLimit <= 0 {
		failureLimit = 3
	}
	if cooldown <= 0 {
		cooldown = 30 * time.Second
	}
	return &CircuitBreaker{
		failures:     make(map[uint]int),
		openedUntil:  make(map[uint]time.Time),
		failureLimit: failureLimit,
		cooldown:     cooldown,
	}
}

func (c *CircuitBreaker) Allow(routeID uint) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	until, open := c.openedUntil[routeID]
	if !open {
		return true
	}
	if time.Now().After(until) {
		delete(c.openedUntil, routeID)
		c.failures[routeID] = 0
		return true
	}
	return false
}

func (c *CircuitBreaker) RecordSuccess(routeID uint) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.failures[routeID] = 0
	delete(c.openedUntil, routeID)
}

func (c *CircuitBreaker) RecordFailure(routeID uint) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.failures[routeID]++
	if c.failures[routeID] >= c.failureLimit {
		c.openedUntil[routeID] = time.Now().Add(c.cooldown)
	}
}
