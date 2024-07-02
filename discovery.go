package discovery

import (
	"errors"

	"github.com/google/uuid"
	"github.com/quietpleasure/discovery/consul"
	"google.golang.org/grpc"
)

// Registry defines a service registry.
type Registry interface {
	// Register creates a service instance record in the registry.
	Register(serviceName, instanceID, serviceHost string, servicePort int, serviceTags []string) error
	// Deregister removes a service instance record from the registry
	Deregister(serviceName, instanceID string) error
	// ServiceAddresses returns the list of addresses of
	// active instances of the given service.
	ServiceAddresses(serviceName string) ([]string, error)
	// ReportHealthyState is a push mechanism for reporting
	// healthy state to the registry.
	ReportHealthyState(serviceName, instanceID string, outputComment ...string) error

	ServiceConnectGRPC(serviceName string, opts ...consul.OptionFunc) (*grpc.ClientConn, error)
}

// ErrNotFound is returned when no service addresses are found
var ErrNotFound = errors.New("no service addresses found")

// GenerateInstanceID generates a pseudo-random service
// instance identifier, using a service name
// suffixed by dash and a random number.
func GenerateInstanceID() string {
	return uuid.New().String()
}
