package utils

import (
	"github.com/DataDog/datadog-agent/pkg/autodiscovery/integration"
	"github.com/DataDog/datadog-agent/pkg/config"
)

// AddContainerCollectAllConfigs adds a config template containing an empty
// LogsConfig when `logs_config.container_collect_all` is set.  This config
// will be filtered out during config resolution if another config template
// also has logs configuration.
func AddContainerCollectAllConfigs(configs []integration.Config, adIdentifier string) []integration.Config {
	if !config.Datadog.GetBool("logs_config.container_collect_all") {
		return configs
	}

	// TODO: what does this need to look like?
	configs = append(configs, integration.Config{
		Name:          "container_collect_all",
		ADIdentifiers: []string{adIdentifier},
		LogsConfig:    []byte("{}"),
	})
	return configs
}
