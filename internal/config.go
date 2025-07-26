package internal

import (
	"errors"
	"io/fs"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/viper"
	"github.com/ygrebnov/errorc"
)

const (
	defaultCfgDir  = "com.yaroslavgrebnov.links"
	defaultCfgFile = "config.yaml"

	envPrefix = "LINKS"

	configKeyInspectorHost           = "inspector.host"
	configKeyInspectorRequestTimeout = "inspector.requestTimeout"
	configKeyInspectorRetryAttempts  = "inspector.retryAttempts"
	configKeyInspectorRetryDelay     = "inspector.retryDelay"
	configKeyPrinterOutputFormat     = "printer.outputFormat"

	defaultInspectorHost           = ""
	defaultInspectorRequestTimeout = 30 * time.Second
	defaultInspectorRetryAttempts  = 3
	defaultInspectorRetryDelay     = 2 * time.Millisecond
)

// inspectorConfig is a configuration for the inspector.
//
//nolint:lll // ignore long lines.
type inspectorConfig struct {
	Host                 string        `mapstructure:"host" yaml:"host,omitempty" json:"host,omitempty"`
	RequestTimeout       time.Duration `mapstructure:"requestTimeout" yaml:"requestTimeout,omitempty" json:"requestTimeout,omitempty"`
	DoNotFollowRedirects bool          `mapstructure:"doNotFollowRedirects" yaml:"doNotFollowRedirects" json:"doNotFollowRedirects"`
	LogExternalLinks     bool          `mapstructure:"logExternalLinks" yaml:"logExternalLinks" json:"logExternalLinks"`
	SkipStatusCodes      []int         `mapstructure:"skipStatusCodes" yaml:"skipStatusCodes,omitempty" json:"skipStatusCodes,omitempty"`
	RetryAttempts        byte          `mapstructure:"retryAttempts" yaml:"retryAttempts" json:"retryAttempts"`
	RetryDelay           time.Duration `mapstructure:"retryDelay" yaml:"retryDelay,omitempty" json:"retryDelay,omitempty"`
}

// printerConfig is a configuration for the printer.
//
//nolint:lll // ignore long lines.
type printerConfig struct {
	SortOutput          bool         `mapstructure:"sortOutput" yaml:"sortOutput" json:"sortOutput"`
	DisplayOccurrences  bool         `mapstructure:"displayOccurrences" yaml:"displayOccurrences" json:"displayOccurrences"`
	SkipOK              bool         `mapstructure:"skipOK" yaml:"skipOK" json:"skipOK"`
	OutputFormat        outputFormat `mapstructure:"outputFormat" yaml:"-" json:"-"`
	DoNotOpenFileReport bool         `mapstructure:"doNotOpenFileReport" yaml:"doNotOpenFileReport" json:"doNotOpenFileReport"`
}

type config struct {
	Inspector *inspectorConfig `mapstructure:"inspector,dive" yaml:"inspector" json:"inspector"`
	Printer   printerConfig    `mapstructure:"printer,dive" yaml:"printer" json:"printer"`
}

func (c *config) validate() error {
	return errors.Join(
		c.validateInspectorHost(),
		c.validatePrinterOutputFormat(),
	)
}

func (c *config) validateInspectorHost() error {
	host := c.Inspector.Host

	if host == "" {
		return ErrEmptyHostValue
	}

	if !strings.HasPrefix(host, "http://") && !strings.HasPrefix(host, "https://") {
		host = "http://" + host
	}

	u, err := url.Parse(host)
	switch {
	case err != nil || u.Host == "":
		return ErrInvalidHostValue // TODO: parse error and return more specific error.

	case u.Scheme == "":
		u.Scheme = "http"
	}

	if u.Port() == "" {
		c.Inspector.Host = u.Scheme + "://" + u.Host
		return nil
	}

	c.Inspector.Host = u.Scheme + "://" + net.JoinHostPort(u.Hostname(), u.Port())
	return nil
}

func (c *config) validatePrinterOutputFormat() error {
	if c.Printer.OutputFormat != outputFormatStdOut &&
		c.Printer.OutputFormat != outputFormatHTML &&
		c.Printer.OutputFormat != outputFormatCSV {
		return errorc.With(
			ErrInvalidPrinterOutputFormatValue,
			errorc.Field("value", string(c.Printer.OutputFormat)),
		)
	}

	return nil
}

func newConfig(cfgFile string, deps injectables) (*config, error) {
	withFile := true

	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		var err error
		if cfgFile, err = getConfigFilePath(false, deps); err != nil {
			return nil, err
		}

		_, err = deps.getStat()(cfgFile)

		switch {
		case err == nil:
			viper.SetConfigFile(cfgFile)

		case !errors.Is(err, fs.ErrNotExist):
			return nil, err

		default:
			// file is created only on configurator.set
			withFile = false
		}
	}

	setDefaults()

	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.SetEnvPrefix(envPrefix)

	if withFile {
		if err := viper.ReadInConfig(); err != nil {
			return nil, ErrInvalidConfigurationSettings // TODO: parse viper error and return more specific error.
		}
	}

	var cfg *config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, ErrInvalidConfigurationSettings // TODO: parse viper error and return more specific error.
	}

	err := cfg.validate()

	return cfg, err
}

func getConfigFilePath(createDir bool, deps injectables) (string, error) {
	userCfgDir, err := deps.getUserConfigDir()()
	if err != nil {
		return "", err
	}

	cfgDirPath := filepath.Join(userCfgDir, defaultCfgDir)

	if createDir {
		if err := os.MkdirAll(cfgDirPath, 0o700); err != nil {
			return "", err
		}
	}

	return filepath.Join(cfgDirPath, defaultCfgFile), nil
}

func setDefaults() {
	viper.SetDefault(configKeyInspectorHost, defaultInspectorHost) // env variable value is not read without this.
	viper.SetDefault(configKeyInspectorRequestTimeout, defaultInspectorRequestTimeout)
	viper.SetDefault(configKeyInspectorRetryAttempts, defaultInspectorRetryAttempts)
	viper.SetDefault(configKeyInspectorRetryDelay, defaultInspectorRetryDelay)
	viper.SetDefault(configKeyPrinterOutputFormat, outputFormatStdOut)
}
