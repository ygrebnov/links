package internal

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

var outYAML = `inspector:
    host: http://localhost
    requestTimeout: 30s
    doNotFollowRedirects: false
    logExternalLinks: false
    retryAttempts: 0
    retryDelay: 2ms
printer:
    sortOutput: true
    displayOccurrences: false
    skipOK: false
    doNotOpenFileReport: false
`

func TestConfigurator_New(t *testing.T) {
	tests := []struct {
		name          string
		before        func(t *testing.T) injectables
		cfgFile       string
		expectedError string
	}{
		{
			name: "default",
			before: func(t *testing.T) injectables {
				setupEnv(t, "LINKS_INSPECTOR_HOST", "localhost")

				return injectables{
					userConfigDir: func() (string, error) {
						return t.TempDir(), nil
					},
				}
			},
		},

		{
			name:          "cannot load configuration",
			cfgFile:       "*invalid",
			expectedError: "cannot load configuration",
		},
	}

	viper.Reset()
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			deps := injectables{}
			if test.before != nil {
				deps = test.before(t)
			}

			_, err := newConfigurator(test.cfgFile, deps)

			if test.expectedError != "" {
				require.ErrorContains(t, err, test.expectedError)
			} else {
				require.NoError(t, err)
			}
		})
	}
	viper.Reset()
}

func TestConfigurator_Show(t *testing.T) {
	tests := []struct {
		name          string
		cfg           *config
		outputFormat  outputFormat
		expected      string
		expectedError string
	}{
		{
			name: "yaml",
			cfg: &config{
				Inspector: &inspectorConfig{
					Host:           "http://localhost",
					RequestTimeout: 30 * time.Second,
					RetryDelay:     2 * time.Millisecond,
				},
				Printer: printerConfig{
					SortOutput: true,
				},
			},
			outputFormat: outputFormatYAML,
			expected:     outYAML,
		},

		{
			name: "json",
			cfg: &config{
				Inspector: &inspectorConfig{
					RetryDelay: 2 * time.Millisecond,
				},
				Printer: printerConfig{
					SortOutput: true,
				},
			},
			outputFormat: outputFormatJSON,
			expected: `{
	"inspector": {
		"doNotFollowRedirects": false,
		"logExternalLinks": false,
		"retryAttempts": 0,
		"retryDelay": 2000000
	},
	"printer": {
		"sortOutput": true,
		"displayOccurrences": false,
		"skipOK": false,
		"doNotOpenFileReport": false
	}
}`,
		},

		{
			name: "default",
			cfg: &config{
				Inspector: &inspectorConfig{
					Host:           "http://localhost",
					RequestTimeout: 30 * time.Second,
					RetryDelay:     2 * time.Millisecond,
				},
				Printer: printerConfig{
					SortOutput: true,
				},
			},
			expected: outYAML,
		},
	}

	viper.Reset()
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var actual string
			c := defaultConfigurator{
				cfg: test.cfg,
				deps: injectables{
					printFn: func(a ...any) (n int, err error) {
						// TODO: include the first line into test.
						actual = strings.Trim(fmt.Sprint(a), "[]")
						return 0, nil
					},
				},
			}
			err := c.show(test.outputFormat)
			if test.expectedError != "" {
				require.ErrorContains(t, err, test.expectedError)
			} else {
				require.Equal(t, test.expected, actual)
			}
		})
	}
	viper.Reset()
}

func TestConfigurator_Set(t *testing.T) {
	viper.Reset()

	setupEnv(t, "LINKS_INSPECTOR_HOST", "localhost")

	tempDir := t.TempDir()
	var actual string
	deps := injectables{
		userConfigDir: func() (string, error) {
			return tempDir, nil
		},
		printFn: func(a ...any) (n int, err error) {
			actual = strings.Trim(fmt.Sprint(a), "[]")
			return 0, nil
		},
	}

	// start with default configuration.
	c, err := newConfigurator("", deps)
	require.NoError(t, err)

	expected := `inspector:
    host: http://localhost
    requestTimeout: 30s
    doNotFollowRedirects: false
    logExternalLinks: false
    retryAttempts: 3
    retryDelay: 2ms
printer:
    sortOutput: false
    displayOccurrences: false
    skipOK: false
    doNotOpenFileReport: false
`

	err = c.show(outputFormatYAML)
	require.NoError(t, err)
	require.Equal(t, expected, actual)

	// set inspector.retryAttempts to zero.
	err = c.set("inspector.retryAttempts", "0")
	require.NoError(t, err)

	expected = `inspector:
    host: http://localhost
    requestTimeout: 30s
    doNotFollowRedirects: false
    logExternalLinks: false
    retryAttempts: 0
    retryDelay: 2ms
printer:
    sortOutput: false
    displayOccurrences: false
    skipOK: false
    doNotOpenFileReport: false
`

	c, err = newConfigurator("", deps) // to simulate a user issuing commands.
	require.NoError(t, err)
	err = c.show(outputFormatYAML)
	require.NoError(t, err)
	require.Equal(t, expected, actual)

	// set printer.sortOutput to true.
	err = c.set("printer.sortOutput", "true")
	require.NoError(t, err)

	c, err = newConfigurator("", deps) // to simulate a user issuing commands.
	require.NoError(t, err)
	err = c.show(outputFormatYAML)
	require.NoError(t, err)
	require.Equal(t, outYAML, actual)

	viper.Reset()
}
