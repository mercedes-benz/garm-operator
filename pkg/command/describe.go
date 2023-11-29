package command

import "github.com/spf13/cobra"

var describeCmd = &cobra.Command{
	Use:     "pools",
	Aliases: []string{"pool", "p"},
	Short:   "analyze and prepare captured traces",
	Long:    "analyze and prepare previous captured traces for further processing",
}

func init() {
	RootCommand.AddCommand(describeCmd)
}
