package links

import (
	"github.com/spf13/cobra"

	"github.com/ygrebnov/links/internal"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show links tool version",
	Run: func(_ *cobra.Command, _ []string) {
		internal.ShowVersion()
	},
}
