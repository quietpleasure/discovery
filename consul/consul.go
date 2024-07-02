package consul

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/consul/api"
)

var ErrServicesNotFound error = fmt.Errorf("no service addresses found")

const (
	SELF_NAME    = "consul"
	default_port = 8500
	default_host = "localhost"
)

func DefaultConfig() *ConsulConfig {
	return &ConsulConfig{
		Host: default_host,
		Port: default_port,
	}
}

// Registry defines a Consul-based service regisry.
type Registry struct {
	client *api.Client
	config *api.Config
}

type ConsulConfig struct {
	Host string
	Port int
	User string
	Pass string
}

type ServiceConfig struct {
	Name string
	Host string
	Port int
	Tags []string
}

// NewRegistry creates a new Consul-based service registry instance.
func NewRegistry(config *ConsulConfig) (*Registry, error) {
	cfg := api.DefaultConfig()
	if config != nil {
		cfg.Address = fmt.Sprintf("%s:%d", config.Host, config.Port)
		cfg.HttpAuth = &api.HttpBasicAuth{
			Username: config.User,
			Password: config.Pass,
		}
	}
	client, err := api.NewClient(cfg)
	if err != nil {
		return nil, err
	}
	return &Registry{client: client, config: cfg}, nil
}

// Register creates a service record in the registry and return instanceID
func (r *Registry) Register(serviceName, instanceID, serviceHost string, servicePort int, serviceTags []string) error {
	if err := r.client.Agent().ServiceRegister(
		&api.AgentServiceRegistration{
			Address: serviceHost,
			ID:      instanceID,
			Name:    serviceName,
			Port:    servicePort,
			Tags:    serviceTags,
			Check:   &api.AgentServiceCheck{CheckID: instanceID, TTL: "5s"},
		},
	); err != nil {
		return err
	}
	return nil
}

// Deregister removes a service record from the registry.
func (r *Registry) Deregister(_, instanceID string) error {
	return r.client.Agent().ServiceDeregister(instanceID)
}

// ServiceAddresses returns the list of addresses of active instances of the given service.
func (r *Registry) ServiceAddresses(serviceName string) ([]string, error) {
	entries, _, err := r.client.Health().Service(serviceName, "", true, nil)
	if err != nil {
		return nil, err
	} else if len(entries) == 0 {
		return nil, ErrServicesNotFound
	}
	var res []string
	for _, e := range entries {
		res = append(res, fmt.Sprintf("%s:%d", e.Service.Address, e.Service.Port))
	}
	return res, nil
}

// ReportHealthyState is a push mechanism for reporting healthy state to the registry.
func (r *Registry) ReportHealthyState(_, instanceID string, outputComment ...string) error {
	return r.client.Agent().UpdateTTL(instanceID, strings.Join(outputComment, "|"), api.HealthPassing)
}

func MakeRegistryAndRegisterService(ctx context.Context, instanceID string, cfgService *ServiceConfig, cfgConsul *ConsulConfig) (*Registry, error) {
	if cfgConsul == nil {
		cfgConsul = DefaultConfig()
	}
	registry, err := NewRegistry(cfgConsul)
	if err != nil {
		return nil, err
	}
	if cfgService == nil {
		return nil, fmt.Errorf("service configuration not defined")
	}
	if err := registry.Register(cfgService.Name, instanceID, cfgService.Host, cfgService.Port, cfgService.Tags); err != nil {
		return nil, err
	}

	return registry, nil
}
