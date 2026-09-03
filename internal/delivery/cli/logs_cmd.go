package cli

import (
	"fmt"

	"github.com/rafliaditya0125/server-project-management/pkg/logger"
	"github.com/spf13/cobra"
)

func (c *CLI) newLogsCmd() *cobra.Command {
	var lines int

	logsCmd := &cobra.Command{
		Use:   "logs [app-name]",
		Short: "Menampilkan log journalctl service systemd aplikasi",
		Args:  cobra.ExactArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) == 0 {
				return c.getAppNamesCompletion(), cobra.ShellCompDirectiveNoFileComp
			}
			return nil, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := checkRoot(); err != nil {
				return err
			}

			appName := args[0]
			logger.Info("Menampilkan %d baris log terakhir untuk service '%s.service'...", lines, appName)
			logger.Raw("%s-----------------------------------------------------------------%s\n", logger.ColorBold, logger.ColorReset)

			logs, err := c.appUsecase.GetLogs(appName, lines)
			if err != nil {
				logger.Warn("Gagal membaca log journalctl: %v", err)
			}
			if logs != "" {
				fmt.Print(logs)
			}

			return nil
		},
	}

	logsCmd.Flags().IntVarP(&lines, "lines", "n", 100, "Jumlah baris log yang ditampilkan")
	return logsCmd
}
