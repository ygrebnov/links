package internal

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func setupEnv(t *testing.T, key, value string) {
	err := os.Setenv(key, value)
	require.NoError(t, err)

	t.Cleanup(func() {
		err = os.Unsetenv(key)
		require.NoError(t, err)
	})
}
