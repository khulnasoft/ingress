package global

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/khulnasoft/kengine/v2"
	"github.com/khulnasoft/kengine/v2/kengineconfig"
	"github.com/khulnasoft/kengine/v2/modules/kenginehttp"
	"github.com/khulnasoft/ingress/pkg/converter"
	"github.com/khulnasoft/ingress/pkg/store"
)

type HealthzPlugin struct{}

func (p HealthzPlugin) IngressPlugin() converter.PluginInfo {
	return converter.PluginInfo{
		Name:     "healthz",
		Priority: -20,
		New:      func() converter.Plugin { return new(HealthzPlugin) },
	}
}

func init() {
	converter.RegisterPlugin(HealthzPlugin{})
}

func (p HealthzPlugin) GlobalHandler(config *converter.Config, store *store.Store) error {
	healthzHandler := kenginehttp.StaticResponse{StatusCode: kenginehttp.WeakString(strconv.Itoa(http.StatusOK))}

	healthzRoute := kenginehttp.Route{
		HandlersRaw: []json.RawMessage{
			kengineconfig.JSONModuleObject(healthzHandler, "handler", healthzHandler.KengineModule().ID.Name(), nil),
		},
		MatcherSetsRaw: []kengine.ModuleMap{{
			"path": kengineconfig.JSON(kenginehttp.MatchPath{"/healthz"}, nil),
		}},
	}

	config.GetMetricsServer().Routes = append(config.GetMetricsServer().Routes, healthzRoute)
	return nil
}

// Interface guards
var (
	_ = converter.GlobalMiddleware(HealthzPlugin{})
)
