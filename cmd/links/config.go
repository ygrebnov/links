package links

import (
	"github.com/spf13/cobra"

	"github.com/ygrebnov/links/internal"
)

var (
	out string

	configCmd = &cobra.Command{
		Use:   "config",
		Short: "Configure links tool",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.HelpFunc()(cmd, args)
		},
	}

	showConfigCmd = &cobra.Command{
		Use:   "show",
		Short: "Show configuration",
		RunE: func(_ *cobra.Command, _ []string) error {
			return internal.ShowConfig(cfgFile, out)
		},
	}

	setConfigCmd = &cobra.Command{
		Use:   "set",
		Short: "Set configuration parameter",
		Args:  cobra.ExactArgs(2),
		RunE: func(_ *cobra.Command, args []string) error {
			return internal.SetConfig(cfgFile, args[0], args[1])
		},
	}
)

func initConfigCmd() {
	showConfigCmd.
		Flags().
		StringVarP(
			&out,
			"out",
			"o",
			"yaml",
			"output type. Possible values are: yaml (default), json",
		)

	configCmd.AddCommand(showConfigCmd, setConfigCmd)
}
