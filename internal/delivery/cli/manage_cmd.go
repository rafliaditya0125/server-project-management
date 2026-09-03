package cli

import (
	"fmt"
	"strings"

	"github.com/rafliaditya0125/server-project-management/internal/domain"
	"github.com/spf13/cobra"
)

func (c *CLI) newManageCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "manage [app-name] [action]",
		Short: "Mengelola service aplikasi (restart, stop, start, status)",
		Args:  cobra.ExactArgs(2),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) == 0 {
				return c.getAppNamesCompletion(), cobra.ShellCompDirectiveNoFileComp
			}
			if len(args) == 1 {
				actions := []string{"restart", "stop", "start", "status"}
				return actions, cobra.ShellCompDirectiveNoFileComp
			}
			return nil, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := checkRoot(); err != nil {
				return err
			}

			appName := args[0]
			actionStr := strings.ToLower(args[1])
			action := domain.ServiceAction(actionStr)

			out, err := c.serviceUsecase.Manage(appName, action)
			if action == domain.ActionStatus && out != "" {
				fmt.Print(out)
			}
			return err
		},
	}
}
