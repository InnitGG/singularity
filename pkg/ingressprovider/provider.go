package ingressprovider

import "net"

type Provider interface {
	// Create creates an ingress and return the id
	Create(hostName string, backendSet []*Backend) (string, error)

	// Update updates an existing ingress' backends
	Update(hostName string, backendSet []*Backend) error

	// Delete deletes an existing ingress
	Delete(id string) error
}

// Backend represents a Minecraft server's connection details
type Backend struct {
	IP   net.IP
	Port uint16
}
