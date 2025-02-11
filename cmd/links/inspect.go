package links

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ygrebnov/links/internal"
)

var (
	outputFormat string

	inspectCmd = &cobra.Command{
		Use:   "inspect",
		Short: "Discover and check links",
		RunE: func(_ *cobra.Command, _ []string) error {
			return internal.Inspect(cfgFile, start)
		},
	}
)

func initInspectCmd() error {
	inspectCmd.
		Flags().
		StringVar(
			&host,
			"host",
			"",
			"host address",
		)
	if err := inspectCmd.MarkFlagRequired("host"); err != nil {
		return err
	}

	if err := viper.BindPFlag("inspector.host", inspectCmd.Flags().Lookup("host")); err != nil {
		return err
	}

	inspectCmd.
		Flags().
		BoolVar(
			&skipOK,
			"skipok",
			false,
			"do not output links checks returning 200 status code",
		)

	if err := viper.BindPFlag("printer.skipOk", inspectCmd.Flags().Lookup("skipok")); err != nil {
		return err
	}

	inspectCmd.
		Flags().
		StringVarP(
			&outputFormat,
			"out",
			"o",
			"stdout",
			"output format. Possible values are: stdout (default), html, csv",
		)

	if err := viper.BindPFlag("printer.outputFormat", inspectCmd.Flags().Lookup("out")); err != nil {
		return err
	}

	inspectCmd.
		Flags().
		StringVar(
			&start,
			"path",
			"/",
			"start path (default: '/')",
		)

	return nil
}
