package main

import (
	"fmt"
	"sync"
)

// TODO add native logs

// Service for registration
type Service struct {
	ID    string
	Name  string
	Type  string
	Event string // regiestered new service, received result from algorithmia, received problem
	// other details
}

type Registry struct {
	RegistryMutex sync.RWMutex
	Services      map[string]Service
}

func (registry *Registry) Register(service Service) error {

	registry.RegistryMutex.Lock()
	defer registry.RegistryMutex.Unlock()
	if _, ok := registry.Services[service.ID]; ok {
		return fmt.Errorf("service with ID %s already exists", service.ID)
	}
	registry.Services[service.ID] = service
	return nil
}

func (registry *Registry) Unregister(id string) error {
	registry.RegistryMutex.Lock()
	defer registry.RegistryMutex.Unlock()
	if _, ok := registry.Services[id]; !ok {
		return fmt.Errorf("service with ID %s not found", id)
	}
	delete(registry.Services, id)
	return nil
}

func (registry *Registry) Get(id string) (Service, bool) {
	registry.RegistryMutex.RLock()
	defer registry.RegistryMutex.RUnlock()
	service, ok := registry.Services[id]
	return service, ok
}

func main() {

	registry := &Registry{
		Services: make(map[string]Service),
	}

	// Register services (example)
	serviceExample := Service{
		ID:    "serviceExample",
		Name:  "Service One",
		Type:  "Algorithmia type",
		Event: "Registered",
	}
	registry.Register(serviceExample)

	// TODO add database

	// Get service information (example)
	service, ok := registry.Get("serviceExample")
	if ok {
		fmt.Printf("Service: %v registered ", service.Name)
	} else {
		fmt.Println("Service unregistered succesfully ")
	}
}
