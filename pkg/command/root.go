package command

import (
	"github.com/spf13/cobra"
)

type options struct {
	kubeConfig        string
	kubeConfigContext string
	namespace         string
	sortBy            []string
	markdown          bool
}

var opt = &options{}

var RootCommand = &cobra.Command{
	Use:   "kubectl-garm",
	Short: "opinionated cli plugin to visualize pools and other garm resources",
	Long:  "this is the long help - define later",
}

func Root() *cobra.Command {
	return RootCommand
}

func init() {

	RootCommand.PersistentFlags().StringVar(&opt.kubeConfig, "kubeconfig", "", "path to the kubeconfig file to use for CLI requests")
	RootCommand.PersistentFlags().StringVar(&opt.kubeConfigContext, "context", "", "name of the kubeconfig context to use")
	RootCommand.PersistentFlags().StringVarP(&opt.namespace, "namespace", "n", "", "namespace to use for CLI requests")

	RootCommand.AddGroup(
		&cobra.Group{
			ID:    "pools",
			Title: "all the pools stuff",
		},
	)
}
