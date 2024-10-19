package cmd

import (
	"github.com/jenkins-x-plugins/jx-changelog/pkg/cmd/create"
	"github.com/jenkins-x/jx-helpers/v3/pkg/options"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"

	"github.com/jenkins-x-plugins/jx-changelog/pkg/cmd/version"
	"github.com/jenkins-x-plugins/jx-changelog/pkg/rootcmd"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras"
	"github.com/spf13/cobra"
)

// Options a few common options we tend to use in command line tools
type Options struct {
	options.BaseOptions
}

// Main creates the new command
func Main() *cobra.Command {
	cmd := &cobra.Command{
		Use:   rootcmd.TopLevelCommand,
		Short: "Command for working with Changelogs",
		Run: func(cmd *cobra.Command, args []string) {
			err := cmd.Help()
			if err != nil {
				log.Logger().Error(err.Error())
			}
		},
	}
	o := options.BaseOptions{}
	o.AddBaseFlags(cmd)
	cmd.AddCommand(cobras.SplitCommand(create.NewCmdChangelogCreate()))
	cmd.AddCommand(cobras.SplitCommand(version.NewCmdVersion()))
	return cmd
}
