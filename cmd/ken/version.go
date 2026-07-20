package main

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version and platform info",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("ken %s\n", version)
		fmt.Printf("%s/%s\n", runtime.GOOS, runtime.GOARCH)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
