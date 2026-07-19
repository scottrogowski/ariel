package cmd

import (
	"fmt"
	"runtime/debug"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: versionShort,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println(arielVersion())
		return nil
	},
}

func arielVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "unknown"
	}
	return info.Main.Version
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
