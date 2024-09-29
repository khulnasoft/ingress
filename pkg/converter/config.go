package converter

import (
	"github.com/khulnasoft/kengine/v2"
	"github.com/khulnasoft/kengine/v2/modules/kenginehttp"
	"github.com/khulnasoft/kengine/v2/modules/kenginetls"
)

// StorageValues represents the config for certmagic storage providers.
type StorageValues struct {
	Namespace string `json:"namespace"`
	LeaseId   string `json:"leaseId"`
}

// Storage represents the certmagic storage configuration.
type Storage struct {
	System string `json:"module"`
	StorageValues
}

// Config represents a kengine2 config file.
type Config struct {
	Admin   kengine.AdminConfig      `json:"admin,omitempty"`
	Storage Storage                `json:"storage"`
	Apps    map[string]interface{} `json:"apps"`
	Logging kengine.Logging          `json:"logging"`
}

func (c Config) GetHTTPServer() *kenginehttp.Server {
	return c.Apps["http"].(*kenginehttp.App).Servers[HttpServer]
}

func (c Config) GetMetricsServer() *kenginehttp.Server {
	return c.Apps["http"].(*kenginehttp.App).Servers[MetricsServer]
}

func (c Config) GetTLSApp() *kenginetls.TLS {
	return c.Apps["tls"].(*kenginetls.TLS)
}

func NewConfig() *Config {
	return &Config{
		Logging: kengine.Logging{},
		Apps: map[string]interface{}{
			"tls": &kenginetls.TLS{CertificatesRaw: kengine.ModuleMap{}},
			"http": &kenginehttp.App{
				Servers: map[string]*kenginehttp.Server{
					HttpServer: {
						AutoHTTPS: &kenginehttp.AutoHTTPSConfig{},
						// Listen to both :80 and :443 ports in order
						// to use the same listener wrappers (PROXY protocol use it)
						Listen: []string{":80", ":443"},
						TLSConnPolicies: kenginetls.ConnectionPolicies{
							&kenginetls.ConnectionPolicy{},
						},
					},
					MetricsServer: {
						Listen:    []string{":9765"},
						AutoHTTPS: &kenginehttp.AutoHTTPSConfig{Disabled: true},
					},
				},
			},
		},
	}
}
