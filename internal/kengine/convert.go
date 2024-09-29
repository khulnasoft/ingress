package kengine

import (
	"github.com/khulnasoft/ingress/pkg/converter"
	"github.com/khulnasoft/ingress/pkg/store"

	// Load default plugins
	_ "github.com/khulnasoft/ingress/internal/kengine/global"
	_ "github.com/khulnasoft/ingress/internal/kengine/ingress"
)

type Converter struct{}

func (c Converter) ConvertToKengineConfig(store *store.Store) (interface{}, error) {
	cfg := converter.NewConfig()

	for _, p := range converter.Plugins(store.Options.PluginsOrder) {
		if m, ok := p.(converter.GlobalMiddleware); ok {
			err := m.GlobalHandler(cfg, store)
			if err != nil {
				return cfg, err
			}
		}
	}
	return cfg, nil
}
