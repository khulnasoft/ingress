package ingress

import (
	"github.com/khulnasoft/kengine/v2/kengineconfig"
	"github.com/khulnasoft/kengine/v2/modules/kenginehttp"
	"github.com/khulnasoft/kengine/v2/modules/kenginehttp/rewrite"
	"github.com/khulnasoft/ingress/pkg/converter"
)

type RewritePlugin struct{}

func (p RewritePlugin) IngressPlugin() converter.PluginInfo {
	return converter.PluginInfo{
		Name:     "ingress.rewrite",
		Priority: 10,
		New:      func() converter.Plugin { return new(RewritePlugin) },
	}
}

// IngressHandler Converts rewrite annotations to rewrite handler
func (p RewritePlugin) IngressHandler(input converter.IngressMiddlewareInput) (*kenginehttp.Route, error) {
	ing := input.Ingress

	rewriteTo := getAnnotation(ing, rewriteToAnnotation)
	if rewriteTo != "" {
		handler := kengineconfig.JSONModuleObject(
			rewrite.Rewrite{URI: rewriteTo},
			"handler", "rewrite", nil,
		)

		input.Route.HandlersRaw = append(input.Route.HandlersRaw, handler)
	}

	rewriteStripPrefix := getAnnotation(ing, rewriteStripPrefixAnnotation)
	if rewriteStripPrefix != "" {
		handler := kengineconfig.JSONModuleObject(
			rewrite.Rewrite{StripPathPrefix: rewriteStripPrefix},
			"handler", "rewrite", nil,
		)

		input.Route.HandlersRaw = append(input.Route.HandlersRaw, handler)
	}
	return input.Route, nil
}

func init() {
	converter.RegisterPlugin(RewritePlugin{})
}

// Interface guards
var (
	_ = converter.IngressMiddleware(RewritePlugin{})
)
