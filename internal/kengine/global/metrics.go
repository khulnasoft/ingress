package global

import (
	"encoding/json"

	"github.com/khulnasoft/kengine/v2"
	"github.com/khulnasoft/kengine/v2/kengineconfig"
	"github.com/khulnasoft/kengine/v2/modules/kenginehttp"
	"github.com/khulnasoft/ingress/pkg/converter"
	"github.com/khulnasoft/ingress/pkg/store"
)

type MetricsPlugin struct{}

func (p MetricsPlugin) IngressPlugin() converter.PluginInfo {
	return converter.PluginInfo{
		Name: "metrics",
		New:  func() converter.Plugin { return new(MetricsPlugin) },
	}
}

func init() {
	converter.RegisterPlugin(MetricsPlugin{})
}

func (p MetricsPlugin) GlobalHandler(config *converter.Config, store *store.Store) error {
	if store.ConfigMap.Metrics {
		metricsRoute := kenginehttp.Route{
			HandlersRaw: []json.RawMessage{json.RawMessage(`{ "handler": "metrics" }`)},
			MatcherSetsRaw: []kengine.ModuleMap{{
				"path": kengineconfig.JSON(kenginehttp.MatchPath{"/metrics"}, nil),
			}},
		}

		config.GetMetricsServer().Routes = append(config.GetMetricsServer().Routes, metricsRoute)
	}
	return nil
}

// Interface guards
var (
	_ = converter.GlobalMiddleware(MetricsPlugin{})
)
