package global

import (
	"encoding/json"

	kengine2 "github.com/khulnasoft/kengine/v2"
	"github.com/khulnasoft/kengine/v2/kengineconfig"
	"github.com/khulnasoft/kengine/v2/modules/kenginetls"
	"github.com/khulnasoft/ingress/pkg/converter"
	"github.com/khulnasoft/ingress/pkg/store"
	"github.com/mholt/acmez/v2/acme"
)

type ConfigMapPlugin struct{}

func init() {
	converter.RegisterPlugin(ConfigMapPlugin{})
}

func (p ConfigMapPlugin) IngressPlugin() converter.PluginInfo {
	return converter.PluginInfo{
		Name: "configmap",
		New:  func() converter.Plugin { return new(ConfigMapPlugin) },
	}
}

func (p ConfigMapPlugin) GlobalHandler(config *converter.Config, store *store.Store) error {
	cfgMap := store.ConfigMap

	tlsApp := config.GetTLSApp()
	httpServer := config.GetHTTPServer()

	if cfgMap.Debug {
		config.Logging.Logs = map[string]*kengine2.CustomLog{"default": {BaseLog: kengine2.BaseLog{Level: "DEBUG"}}}
	}

	if cfgMap.AcmeCA != "" || cfgMap.Email != "" {
		acmeIssuer := kenginetls.ACMEIssuer{}

		if cfgMap.AcmeCA != "" {
			acmeIssuer.CA = cfgMap.AcmeCA
		}

		if cfgMap.AcmeEABKeyId != "" && cfgMap.AcmeEABMacKey != "" {
			acmeIssuer.ExternalAccount = &acme.EAB{
				KeyID:  cfgMap.AcmeEABKeyId,
				MACKey: cfgMap.AcmeEABMacKey,
			}
		}

		if cfgMap.Email != "" {
			acmeIssuer.Email = cfgMap.Email
		}

		var onDemandConfig *kenginetls.OnDemandConfig
		if cfgMap.OnDemandTLS {
			onDemandConfig = &kenginetls.OnDemandConfig{
				RateLimit: &kenginetls.RateLimit{
					Interval: cfgMap.OnDemandRateLimitInterval,
					Burst:    cfgMap.OnDemandRateLimitBurst,
				},
				Ask: cfgMap.OnDemandAsk,
			}
		}

		tlsApp.Automation = &kenginetls.AutomationConfig{
			OnDemand:          onDemandConfig,
			OCSPCheckInterval: cfgMap.OCSPCheckInterval,
			Policies: []*kenginetls.AutomationPolicy{
				{
					IssuersRaw: []json.RawMessage{
						kengineconfig.JSONModuleObject(acmeIssuer, "module", "acme", nil),
					},
					OnDemand: cfgMap.OnDemandTLS,
				},
			},
		}
	}

	if cfgMap.ProxyProtocol {
		httpServer.ListenerWrappersRaw = []json.RawMessage{
			json.RawMessage(`{"wrapper":"proxy_protocol"}`),
			json.RawMessage(`{"wrapper":"tls"}`),
		}
	}
	return nil
}

// Interface guards
var (
	_ = converter.GlobalMiddleware(ConfigMapPlugin{})
)
