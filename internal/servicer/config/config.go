package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	cfgKey     = "app"
	configPath = "./_config"
	envDefault = "development"
	envVar     = "APP_ENV"

	// ErrMissingAppConfig error that shares the app configuration is not provided
	ErrMissingAppConfig err = "[config] app configuration must be provided in " + configPath + "/<>.yaml file"
)

type (
	err string

	// Servicer is a config servicer
	Servicer interface {
		Populate(key string, cfg interface{}) error
		Env() string
		App() string
		Version() string
	}

	servicer struct {
		content map[string][]byte
		env     string
		app     string
		version string
	}

	appConfig struct {
		Name    string `yaml:"name"`
		Version string `yaml:"version"`
	}
)

// New inits a new Config based on env name
func New() (Servicer, error) {
	// read base yaml file
	basefilename, err := filepath.Abs(configPath + "/base.yaml")
	if err != nil {
		return nil, err
	}

	baseYaml, err := os.ReadFile(basefilename)
	if err != nil {
		return nil, err
	}
	baseCfg := map[string]interface{}{}
	if err := yaml.Unmarshal(baseYaml, &baseCfg); err != nil {
		return nil, err
	}

	// read env yaml file
	env := env()
	envfilename, err := filepath.Abs(fmt.Sprintf(configPath+"/%s.yaml", env))
	if err != nil {
		return nil, err
	}
	envYaml, err := os.ReadFile(envfilename)
	if err != nil {
		return nil, err
	}
	envCfg := map[string]interface{}{}
	if err := yaml.Unmarshal(envYaml, &envCfg); err != nil {
		return nil, err
	}

	merged := mergeMaps(baseCfg, envCfg)
	yamlFile, err := yaml.Marshal(merged)
	if err != nil {
		return nil, err
	}

	// expand env vars
	mapper := func(placeholderName string) string {
		split := strings.Split(placeholderName, ":")
		defValue := ""
		if len(split) == 2 {
			placeholderName = split[0]
			defValue = split[1]
		}

		val, ok := os.LookupEnv(placeholderName)
		if !ok {
			return defValue
		}

		return val
	}
	body := []byte(os.Expand(string(yamlFile), mapper))

	// parse into a generic map
	var cfg map[string]interface{}
	if err := yaml.Unmarshal(body, &cfg); err != nil {
		return nil, err
	}

	content := make(map[string][]byte, len(cfg))
	for k, v := range cfg {
		content[k], _ = yaml.Marshal(v)
	}

	s := &servicer{
		content: content,
		env:     env,
	}

	var appCfg appConfig
	err = s.Populate(cfgKey, &appCfg)
	if err != nil {
		return nil, err
	}

	if appCfg.Name == "" || appCfg.Version == "" {
		return nil, ErrMissingAppConfig
	}

	s.app = appCfg.Name
	s.version = appCfg.Version

	return s, nil
}

// Populate populates configuration
func (s *servicer) Populate(key string, cfg interface{}) error {
	if err := yaml.Unmarshal(s.content[key], cfg); err != nil {
		return err
	}
	return nil
}

// Env return current config environment
func (s *servicer) Env() string {
	if s.env != "" {
		return s.env
	}
	return env()
}

// App return current app name
func (s *servicer) App() string {
	return s.app
}

// Version return current app version
func (s *servicer) Version() string {
	return s.version
}

// Env return current config environment
func env() string {
	env := os.Getenv(envVar)
	if env != "" {
		return env
	}
	return envDefault
}

func mergeMaps(a, b map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{}, len(a))
	for k, v := range a {
		out[k] = v
	}
	for k, v := range b {
		if v, ok := v.(map[string]interface{}); ok {
			if bv, ok := out[k]; ok {
				if bv, ok := bv.(map[string]interface{}); ok {
					out[k] = mergeMaps(bv, v)
					continue
				}
			}
		}
		out[k] = v
	}
	return out
}

func (e err) Error() string {
	return string(e)
}
