package internal

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestNewConfig(t *testing.T) {
	tests := []struct {
		name        string
		cfgFile     string
		before      func(t *testing.T) injectables
		expected    *config
		expectedErr string
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
			expected: &config{
				Inspector: &inspectorConfig{
					Host:           "http://localhost",
					RequestTimeout: 30 * time.Second,
					RetryDelay:     2 * time.Millisecond,
					RetryAttempts:  3,
				},
				Printer: printerConfig{
					OutputFormat: outputFormatStdOut,
				},
			},
		},

		{
			name: "file with default name and path",
			before: func(t *testing.T) injectables {
				dir := t.TempDir()

				testCfgDir := filepath.Join(dir, defaultCfgDir)

				err := os.Mkdir(testCfgDir, 0o700)
				require.NoError(t, err)

				testCfgFile := filepath.Join(testCfgDir, defaultCfgFile)

				c := config{
					Inspector: &inspectorConfig{
						RetryAttempts: 10,
						Host:          "testhost",
					},
					Printer: printerConfig{
						SortOutput: true,
					},
				}

				b, err := yaml.Marshal(c)
				require.NoError(t, err)

				err = os.WriteFile(testCfgFile, b, 0o600)
				require.NoError(t, err)

				return injectables{
					userConfigDir: func() (string, error) {
						return dir, nil
					},
				}
			},
			expected: &config{
				Inspector: &inspectorConfig{
					RequestTimeout: 30 * time.Second,
					RetryDelay:     2 * time.Millisecond,
					RetryAttempts:  10,
					Host:           "http://testhost",
				},
				Printer: printerConfig{
					SortOutput:   true,
					OutputFormat: outputFormatStdOut,
				},
			},
		},

		{
			name: "invalid output format",
			before: func(t *testing.T) injectables {
				setupEnv(t, "LINKS_PRINTER_OUTPUTFORMAT", "invalid")

				return injectables{
					userConfigDir: func() (string, error) {
						return t.TempDir(), nil
					},
				}
			},
			expectedErr: ErrInvalidPrinterOutputFormatValue.Error(),
		},

		{
			name: "os.stat error",
			before: func(t *testing.T) injectables {
				return injectables{
					userConfigDir: func() (string, error) {
						return t.TempDir(), nil
					},
					stat: func(string) (os.FileInfo, error) {
						return nil, errors.New("os.stat error")
					},
				}
			},
			expectedErr: "os.stat error",
		},

		{
			name: "viper.ReadInConfig error",
			before: func(t *testing.T) injectables {
				dir := t.TempDir()

				testCfgDir := filepath.Join(dir, defaultCfgDir)

				err := os.Mkdir(testCfgDir, 0o700)
				require.NoError(t, err)

				testCfgFile := filepath.Join(testCfgDir, defaultCfgFile)

				b := []byte(`inspector:
	retryAttempts: [`)

				err = os.WriteFile(testCfgFile, b, 0o600)
				require.NoError(t, err)

				return injectables{
					userConfigDir: func() (string, error) {
						return dir, nil
					},
				}
			},
			expectedErr: ErrInvalidConfigurationSettings.Error(),
		},

		{
			name: "viper.Unmarshal error",
			before: func(t *testing.T) injectables {
				setupEnv(t, "LINKS_INSPECTOR_RETRYATTEMPTS", "invalid")

				return injectables{
					userConfigDir: func() (string, error) {
						return t.TempDir(), nil
					},
				}
			},
			expectedErr: ErrInvalidConfigurationSettings.Error(),
		},

		{
			name: "os.userConfigDir error",
			before: func(*testing.T) injectables {
				return injectables{
					userConfigDir: func() (string, error) {
						return "", errors.New("os.userConfigDir error")
					},
				}
			},
			expectedErr: "os.userConfigDir error",
		},
	}

	viper.Reset()
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			deps := injectables{}
			if test.before != nil {
				deps = test.before(t)
			}

			c, cErr := newConfig(test.cfgFile, deps)

			if test.expectedErr != "" {
				require.ErrorContains(t, cErr, test.expectedErr)
			} else {
				require.Nil(t, cErr)
				require.Equal(t, test.expected, c)
			}
		})
	}
	viper.Reset()
}
