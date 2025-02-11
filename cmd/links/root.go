package links

import (
	"github.com/spf13/cobra"
)

var (
	// Used for flags.
	cfgFile     string
	host, start string
	skipOK      bool

	rootCmd = &cobra.Command{
		Use:               "links",
		Short:             "Links inspecting tool",
		DisableAutoGenTag: true,
		CompletionOptions: cobra.CompletionOptions{
			DisableDefaultCmd: true,
		},
		Run: func(cmd *cobra.Command, args []string) {
			cmd.HelpFunc()(cmd, args)
		},
	}
)

// Execute executes the root command.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.
		PersistentFlags().
		StringVar(
			&cfgFile,
			"config",
			"",
			`path to config file (default location is displayed on the first line 
of 'links config show' command output).
Config file is automatically created only on 'config set' command execution.`,
		)

	cobra.CheckErr(initInspectCmd())
	initConfigCmd()

	rootCmd.AddCommand(inspectCmd, configCmd, versionCmd)
}
