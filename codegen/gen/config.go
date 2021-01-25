package gen

import (
	"fmt"
	"io/ioutil"
)

// Config is the parsed configuration.
type Config struct {
}

// merge merges this config with a new config.
func (c Config) merge(new Config) Config {
	return c
}

// mergeConfigs parses and merges all the configs using "last write wins" to
// resolve conflicts.
func mergeConfigs(configs []string) (Config, error) {
	conf := Config{}
	for _, config := range configs {
		bs, err := ioutil.ReadFile(config)
		if err != nil {
			return Config{}, fmt.Errorf("read pggen config file: %w", err)
		}
		c, err := parseConfig(bs)
		if err != nil {
			return Config{}, fmt.Errorf("parse pggen config file: %w", err)
		}
		conf = conf.merge(c)
	}
	return conf, nil
}

func parseConfig(bs []byte) (Config, error) {
	return Config{}, nil
}
