package kengine

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/khulnasoft/ingress/pkg/store"
	"github.com/stretchr/testify/require"
)

func TestConvertToKengineConfig(t *testing.T) {
	tests := []struct {
		name               string
		expectedConfigPath string
	}{
		{
			name:               "default",
			expectedConfigPath: "./test_data/default.json",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg, err := Converter{}.ConvertToKengineConfig(store.NewStore(store.Options{}, &store.PodInfo{}))
			require.NoError(t, err)

			cfgJson, err := json.Marshal(cfg)
			require.NoError(t, err)

			expectedCfg, err := os.ReadFile(test.expectedConfigPath)
			require.NoError(t, err)

			require.JSONEq(t, string(expectedCfg), string(cfgJson))
		})
	}
}
