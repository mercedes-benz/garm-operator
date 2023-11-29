package command

import (
	"context"
	"os"
	"slices"

	"github.com/spf13/cobra"

	"github.com/jedib0t/go-pretty/v6/table"

	garmoperatorv1alpha1 "github.com/mercedes-benz/garm-operator/api/v1alpha1"
)

var overviewCmd = &cobra.Command{
	Use:     "overview",
	Aliases: []string{"o"},
	Short:   "analyze and prepare captured traces",
	Long:    "analyze and prepare previous captured traces for further processing",
	RunE:    generateOverview,
}

func init() {
	describeCmd.AddCommand(overviewCmd)

	describeCmd.PersistentFlags().StringSliceVar(&opt.sortBy, "sort-by", []string{"region", "flavor"}, "sorted")
	describeCmd.PersistentFlags().BoolVar(&opt.markdown, "markdown", false, "output as markdown")
}

type providerSummary struct {
	name    string
	flavors []flavor
}

type flavor struct {
	name      string
	maxRunner uint
	minRunner uint
}

func generateOverview(cmd *cobra.Command, args []string) error {

	kubeClient, err := newRestClient()
	if err != nil {
		return err
	}

	poolList := &garmoperatorv1alpha1.PoolList{}
	pools := kubeClient.Get().
		Namespace("garm-infra-stage-prod").
		Resource("pools").
		Do(context.Background())

	pools.Into(poolList)

	summary := []providerSummary{}

	for _, pool := range poolList.Items {

		providerName := pool.Spec.ProviderName
		maxRunners := pool.Spec.MaxRunners
		minIdleRunners := pool.Spec.MinIdleRunners

		providerIndex := slices.IndexFunc(summary, func(provider providerSummary) bool { return provider.name == providerName })
		if providerIndex == -1 {
			summary = append(summary, providerSummary{
				name: providerName,
				flavors: []flavor{
					{
						name:      pool.Spec.Flavor,
						maxRunner: pool.Spec.MaxRunners,
						minRunner: pool.Spec.MinIdleRunners,
					},
				},
			})
		} else {

			flavorIndex := slices.IndexFunc(summary[providerIndex].flavors, func(flavor flavor) bool { return flavor.name == pool.Spec.Flavor })
			if flavorIndex == -1 {
				summary[providerIndex].flavors = append(summary[providerIndex].flavors, flavor{
					name:      pool.Spec.Flavor,
					maxRunner: pool.Spec.MaxRunners,
					minRunner: pool.Spec.MinIdleRunners,
				})
			} else {

				summary[providerIndex].flavors[flavorIndex].maxRunner += maxRunners // calculate the runners on top
				summary[providerIndex].flavors[flavorIndex].minRunner += minIdleRunners
			}
		}

	}

	printOverview(summary)

	return nil
}

func printOverview(summary []providerSummary) {

	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{"region", "flavor", "max runners", "Min Idle Runners"})

	for _, s := range summary {
		for _, f := range s.flavors {
			t.AppendRow([]interface{}{s.name, f.name, f.maxRunner, f.minRunner})
		}
	}

	if len(opt.sortBy) > 0 {
		for _, sortOption := range opt.sortBy {
			t.SortBy([]table.SortBy{
				{Name: sortOption, Mode: table.Asc},
			})
		}

	}

	t.SetStyle(table.StyleColoredBlackOnRedWhite)

	if opt.markdown {
		t.RenderMarkdown()
	} else {
		t.Render()
	}

}
