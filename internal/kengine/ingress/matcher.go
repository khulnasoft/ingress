package ingress

import (
	"github.com/khulnasoft/kengine/v2"
	"github.com/khulnasoft/kengine/v2/kengineconfig"
	"github.com/khulnasoft/kengine/v2/modules/kenginehttp"
	"github.com/khulnasoft/ingress/pkg/converter"
	v1 "k8s.io/api/networking/v1"
)

type MatcherPlugin struct{}

func (p MatcherPlugin) IngressPlugin() converter.PluginInfo {
	return converter.PluginInfo{
		Name: "ingress.matcher",
		New:  func() converter.Plugin { return new(MatcherPlugin) },
	}
}

// IngressHandler Generate matchers for the route.
func (p MatcherPlugin) IngressHandler(input converter.IngressMiddlewareInput) (*kenginehttp.Route, error) {
	match := kengine.ModuleMap{}

	if getAnnotation(input.Ingress, disableSSLRedirect) != "true" {
		match["protocol"] = kengineconfig.JSON(kenginehttp.MatchProtocol("https"), nil)
	}

	if input.Rule.Host != "" {
		match["host"] = kengineconfig.JSON(kenginehttp.MatchHost{input.Rule.Host}, nil)
	}

	if input.Path.Path != "" {
		p := input.Path.Path

		if *input.Path.PathType == v1.PathTypePrefix {
			p += "*"
		}
		match["path"] = kengineconfig.JSON(kenginehttp.MatchPath{p}, nil)
	}

	input.Route.MatcherSetsRaw = append(input.Route.MatcherSetsRaw, match)
	return input.Route, nil
}

func init() {
	converter.RegisterPlugin(MatcherPlugin{})
}

// Interface guards
var (
	_ = converter.IngressMiddleware(MatcherPlugin{})
)
