package tests

import (
	"fmt"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestE2E(t *testing.T) {
	s := newServer()
	defer s.Close()

	tests := []struct {
		name          string
		args          []string
		expected      []string
		expectedFn    func(t *testing.T, actual interface{})
		expectedError string
	}{
		{
			name: "config show",
			args: []string{"config", "show"},
			expectedFn: func(t *testing.T, actual interface{}) {
				typed, ok := actual.([]string)
				require.True(t, ok)
				require.Greater(t, len(typed), 1)

				expectedConfig := []string{
					"inspector:",
					"    requestTimeout: 30s",
					"    doNotFollowRedirects: false",
					"    logExternalLinks: false",
					"    retryAttempts: 3",
					"    retryDelay: 2ms",
					"printer:",
					"    sortOutput: false",
					"    displayOccurrences: false",
					"    skipOK: false",
					"    doNotOpenFileReport: false",
				}

				require.ElementsMatch(t, typed[1:], expectedConfig)
			},
		},

		{
			name: "inspect nominal",
			args: []string{"inspect", "--host", s.URL},
			expected: []string{
				fmt.Sprintf("200 - %s/", s.URL),
				fmt.Sprintf("500 - %s/error", s.URL),
				fmt.Sprintf("404 - %s/notfound", s.URL),
				fmt.Sprintf("200 - %s/nosubsequentlinks", s.URL),
			},
		},

		{
			name:          "inspect no host",
			args:          []string{"inspect"},
			expectedError: "Error: required flag(s) \"host\" not set",
		},

		{
			name: "version",
			args: []string{"version"},
			expectedFn: func(t *testing.T, actual interface{}) {
				typed, ok := actual.([]string)
				require.True(t, ok)
				require.Len(t, typed, 1)

				elements := strings.Split(typed[0], ",")
				require.Len(t, elements, 3)

				require.Equal(t, "links", elements[0])

				el1 := strings.Trim(elements[1], " ")
				require.True(
					t,
					strings.HasPrefix(el1, "version: v"),
					"expected '%s' to start with 'version: v'",
					el1,
				)

				el2 := strings.Trim(elements[2], " ")
				require.True(
					t,
					strings.HasPrefix(el2, "built: "),
					"expected '%s' to start with 'built: '",
					el2,
				)
				require.Greater(t, len(elements[2]), 7)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			out, err := exec.Command("./links", test.args...).CombinedOutput()

			switch {
			case test.expectedError != "":
				require.Error(t, err)
				require.Contains(t, string(out), test.expectedError)

			case test.expectedFn != nil:
				require.NoError(t, err)
				test.expectedFn(t, getOutput(out))

			default:
				require.NoError(t, err)
				require.ElementsMatch(t, test.expected, getOutput(out))
			}
		})
	}
}

func getOutput(b []byte) []string {
	s := strings.Split(string(b), "\n")
	output := make([]string, 0, len(s))
	for _, i := range s {
		if i != "" {
			output = append(output, i)
		}
	}

	return output
}
