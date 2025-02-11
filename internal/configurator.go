package internal

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

// configurator performs read and write operations on configuration settings values.
type configurator interface {
	show(o outputFormat) error
	set(key, value string) error
}

// defaultConfigurator holds configuration settings values.
type defaultConfigurator struct {
	cfg  *config
	deps injectables
}

// newConfigurator creates a new configurator and returns it.
func newConfigurator(cfgFile string, deps injectables) (configurator, error) {
	cfg, cfgErr := newConfig(cfgFile, deps)
	if cfgErr != nil && !errors.Is(cfgErr, ErrEmptyHostValue) {
		// single ErrEmptyHostValue is skipped as 'host' value must be provided for 'inspect' command.
		return nil, fmt.Errorf("cannot load configuration: %w", cfgErr)
	}

	return &defaultConfigurator{cfg: cfg, deps: deps}, nil
}

// show displays actual configuration in requested (yaml by default) format.
func (c *defaultConfigurator) show(o outputFormat) error {
	var (
		b   []byte
		err error
	)

	switch o {
	case outputFormatJSON:
		b, err = json.MarshalIndent(c.cfg, "", "\t")
	default:
		o = outputFormatYAML
		b, err = yaml.Marshal(c.cfg)
	}

	if err != nil {
		return fmt.Errorf("cannot marshal configuration into %s", o)
	}

	// output configuration file status.
	_, _ = c.deps.getPrintFn()(c.getConfigFile())

	// output configuration.
	_, _ = c.deps.getPrintFn()(string(b))

	return nil
}

// set assigns given value to the given configuration key.
// Configuration modification is saved into the configuration file.
// In case configuration file does not exist, it is created
// with the default name at the default location.
func (c *defaultConfigurator) set(key, value string) error {
	viper.Set(key, value)

	var cfg config
	if err := viper.Unmarshal(&cfg); err != nil {
		return fmt.Errorf("cannot unmarshal configuration: %w", err)
	}

	if err := cfg.validate(); err != nil {
		return err
	}

	cfgFile := viper.ConfigFileUsed()
	if cfgFile == "" {
		var err error
		if cfgFile, err = getConfigFilePath(true, c.deps); err != nil {
			return err
		}

		viper.SetConfigFile(cfgFile)
		viper.SetConfigPermissions(0o600)
	}

	return viper.WriteConfig()
}

// getConfigFile returns path to the used configuration file, if it exists, or
// a message describing its status.
func (c *defaultConfigurator) getConfigFile() string {
	// return path to known to viper configuration file, if it exists.
	cfgFilePath := viper.ConfigFileUsed()
	if cfgFilePath != "" {
		return "Configuration file path: " + cfgFilePath
	}

	// check configuration file existence at default location.
	var err error
	if cfgFilePath, err = getConfigFilePath(false, c.deps); err != nil {
		return "Error retrieving configuration file at default location: " + err.Error()
	}

	_, err = c.deps.getStat()(cfgFilePath)

	switch {
	case err == nil:
		return "Configuration file path: " + cfgFilePath

	case !errors.Is(err, fs.ErrNotExist):
		return "Configuration file does not exist at default location: " + cfgFilePath

	default:
		return "Error retrieving configuration file at default location: " + err.Error()
	}
}
