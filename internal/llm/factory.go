package llm

import (
	"fmt"
	"sync"
)

// ClientFactory is a function that creates a Client instance
type ClientFactory func(config *ClientConfig) (Client, error)

// registry holds registered client factories
var (
	registry     = make(map[string]ClientFactory)
	registryLock sync.RWMutex
)

// Register registers a client factory with the given name
func Register(name string, factory ClientFactory) {
	registryLock.Lock()
	defer registryLock.Unlock()
	registry[name] = factory
}

// Create creates a client by name using the registered factory
func Create(name string, config *ClientConfig) (Client, error) {
	registryLock.RLock()
	factory, ok := registry[name]
	registryLock.RUnlock()

	if !ok {
		return nil, NewClientError(name, "create", fmt.Sprintf("client '%s' not registered", name), nil)
	}

	// Ensure config has the correct name
	if config == nil {
		config = NewClientConfig(name)
	} else if config.Name == "" {
		config.Name = name
	}

	return factory(config)
}

// List returns all registered client names
func List() []string {
	registryLock.RLock()
	defer registryLock.RUnlock()

	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	return names
}

// IsRegistered checks if a client is registered
func IsRegistered(name string) bool {
	registryLock.RLock()
	defer registryLock.RUnlock()
	_, ok := registry[name]
	return ok
}

// Unregister removes a client factory (mainly for testing)
func Unregister(name string) {
	registryLock.Lock()
	defer registryLock.Unlock()
	delete(registry, name)
}





