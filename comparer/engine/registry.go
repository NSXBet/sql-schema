package engine

import (
	"sync"

	"github.com/pkg/errors"
)

// Registry holds the registered comparers for each engine.
type Registry struct {
	mu        sync.RWMutex
	comparers map[Engine]Comparer
}

// NewRegistry creates a new comparer registry.
func NewRegistry() *Registry {
	return &Registry{
		comparers: make(map[Engine]Comparer),
	}
}

// Register registers a comparer for a specific engine.
// If a comparer is already registered for the engine, it will be replaced.
func (r *Registry) Register(engine Engine, comparer Comparer) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.comparers[engine] = comparer
}

// Get retrieves the comparer for a specific engine.
// Returns an error if no comparer is registered for the engine.
func (r *Registry) Get(engine Engine) (Comparer, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	comparer, ok := r.comparers[engine]
	if !ok {
		return nil, errors.Errorf("no comparer registered for engine: %s", engine)
	}
	return comparer, nil
}

// Has checks if a comparer is registered for the given engine.
func (r *Registry) Has(engine Engine) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.comparers[engine]
	return ok
}

// Unregister removes the comparer for a specific engine.
func (r *Registry) Unregister(engine Engine) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.comparers, engine)
}

// ListEngines returns all registered engines.
func (r *Registry) ListEngines() []Engine {
	r.mu.RLock()
	defer r.mu.RUnlock()

	engines := make([]Engine, 0, len(r.comparers))
	for engine := range r.comparers {
		engines = append(engines, engine)
	}
	return engines
}

// Global registry instance.
var defaultRegistry = NewRegistry()

// Register registers a comparer in the global registry.
func Register(engine Engine, comparer Comparer) {
	defaultRegistry.Register(engine, comparer)
}

// Get retrieves a comparer from the global registry.
func Get(engine Engine) (Comparer, error) {
	return defaultRegistry.Get(engine)
}

// Has checks if a comparer is registered in the global registry.
func Has(engine Engine) bool {
	return defaultRegistry.Has(engine)
}

// Unregister removes a comparer from the global registry.
func Unregister(engine Engine) {
	defaultRegistry.Unregister(engine)
}

// ListEngines returns all registered engines from the global registry.
func ListEngines() []Engine {
	return defaultRegistry.ListEngines()
}

// GetRegistry returns the global registry instance.
// This is useful for testing or custom registry scenarios.
func GetRegistry() *Registry {
	return defaultRegistry
}
