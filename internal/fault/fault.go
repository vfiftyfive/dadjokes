package fault

import (
	"sync"
	"time"
)

// FaultType represents the type of fault to inject
type FaultType string

const (
	FaultDelay     FaultType = "delay"
	FaultError     FaultType = "error"
	FaultPartition FaultType = "partition"
)

// Component represents a system component that can be faulted
type Component string

const (
	ComponentMongo  Component = "mongo"
	ComponentRedis  Component = "redis"
	ComponentNATS   Component = "nats"
	ComponentOpenAI Component = "openai"
)

// Fault represents an injected fault
type Fault struct {
	Component Component
	Type      FaultType
	Duration  time.Duration
	Active    bool
}

// FaultInjector manages fault injection
type FaultInjector struct {
	mu     sync.RWMutex
	faults map[Component]*Fault
}

// NewFaultInjector creates a new fault injector
func NewFaultInjector() *FaultInjector {
	return &FaultInjector{
		faults: make(map[Component]*Fault),
	}
}

// InjectFault injects a fault for a component
func (fi *FaultInjector) InjectFault(component Component, faultType FaultType, duration time.Duration) {
	fi.mu.Lock()
	defer fi.mu.Unlock()

	fi.faults[component] = &Fault{
		Component: component,
		Type:      faultType,
		Duration:  duration,
		Active:    true,
	}

	// Auto-restore after duration
	if duration > 0 {
		go func() {
			time.Sleep(duration)
			fi.RestoreFault(component)
		}()
	}
}

// RestoreFault removes a fault for a component
func (fi *FaultInjector) RestoreFault(component Component) {
	fi.mu.Lock()
	defer fi.mu.Unlock()

	if fault, exists := fi.faults[component]; exists {
		fault.Active = false
		delete(fi.faults, component)
	}
}

// GetFault returns the active fault for a component
func (fi *FaultInjector) GetFault(component Component) *Fault {
	fi.mu.RLock()
	defer fi.mu.RUnlock()

	if fault, exists := fi.faults[component]; exists && fault.Active {
		return fault
	}
	return nil
}

// ShouldFail returns true if the component should fail
func (fi *FaultInjector) ShouldFail(component Component) bool {
	fault := fi.GetFault(component)
	return fault != nil && fault.Type == FaultError
}

// GetDelay returns the delay duration for a component
func (fi *FaultInjector) GetDelay(component Component) time.Duration {
	fault := fi.GetFault(component)
	if fault != nil && fault.Type == FaultDelay {
		return 5 * time.Second // Fixed delay for simplicity
	}
	return 0
}

// IsPartitioned returns true if the component is partitioned
func (fi *FaultInjector) IsPartitioned(component Component) bool {
	fault := fi.GetFault(component)
	return fault != nil && fault.Type == FaultPartition
}
