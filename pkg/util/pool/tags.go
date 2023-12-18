// SPDX-License-Identifier: MIT

package pool

import (
	"sort"

	providerParams "github.com/cloudbase/garm-provider-common/params"
	"github.com/cloudbase/garm-provider-common/util"
	"github.com/cloudbase/garm/params"
	"github.com/life4/genesis/slices"
)

// from: https://github.com/cloudbase/garm/blob/46ac1b81666250e21a5d31fc8c35d754e8d0b601/runner/runner.go#L779-L786

// github automatically adds the "self-hosted" tag as well as the OS type (linux, windows, etc)
// and architecture (arm, x64, etc) to all self hosted runners. When a workflow job comes in, we try
// to find a pool based on the labels that are set in the workflow. If we don't explicitly define these
// default tags for each pool, and the user targets these labels, we won't be able to match any pools.
// The downside is that all pools with the same OS and arch will have these default labels. Users should
// set distinct and unique labels on each pool, and explicitly target those labels, or risk assigning
// the job to the wrong worker type.

func CreateComparableRunnerTags(poolTags []string, osArch providerParams.OSArch, osType providerParams.OSType) ([]params.Tag, error) {
	githubDefaultTags, err := getGithubDefaultTags(osArch, osType)
	if err != nil {
		return []params.Tag{}, err
	}

	poolTags = append(poolTags, githubDefaultTags...)

	tags := []params.Tag{}

	for _, tag := range poolTags {
		tags = append(tags, params.Tag{
			Name: tag,
		})
	}

	// sort tags to ensure that the order is always the same
	sort.Slice(tags, func(i, j int) bool {
		return tags[i].Name < tags[j].Name
	})

	return slices.Uniq(tags), nil
}

func getGithubDefaultTags(osArch providerParams.OSArch, osType providerParams.OSType) ([]string, error) {
	ghArch, err := util.ResolveToGithubArch(string(osArch))
	if err != nil {
		return []string{}, err
	}
	ghOSType, err := util.ResolveToGithubTag(osType)
	if err != nil {
		return []string{}, err
	}

	return []string{
		"self-hosted",
		ghArch,
		ghOSType,
	}, nil
}
