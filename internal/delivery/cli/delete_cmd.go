package cli

import (
	"fmt"

	"github.com/rafliaditya0125/server-project-management/internal/domain"
	"github.com/rafliaditya0125/server-project-management/pkg/logger"
	"github.com/rafliaditya0125/server-project-management/pkg/terminal"
	"github.com/spf13/cobra"
)

func (c *CLI) newDeleteCmd() *cobra.Command {
	var forceFlag bool

	deleteCmd := &cobra.Command{
		Use:   "delete [app-name]",
		Short: "Menghapus user sistem, direktori home, database, dan service",
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
			app, _ := c.appRepo.FindByName(appName)
			dbName := appName
			dbUser := appName
			if app != nil {
				dbName = app.DBName
				dbUser = app.DBUser
			}

			if !forceFlag {
				logger.Raw("\n%s%sPERINGATAN:%s Tindakan ini akan menghapus:\n", logger.ColorBold, logger.ColorRed, logger.ColorReset)
				fmt.Printf("  1. User sistem '%s' dan semua prosesnya\n", appName)
				fmt.Printf("  2. Seluruh isi direktori home '/home/apps/%s'\n", appName)
				fmt.Printf("  3. Service systemd user '%s.service'\n", appName)
				fmt.Printf("  4. Database '%s' dan user database '%s' (jika ada)\n\n", dbName, dbUser)

				if !terminal.Confirm("Apakah Anda yakin ingin melanjutkan?", false) {
					logger.Info("Penghapusan dibatalkan.")
					return nil
				}
			}

			logger.Raw("\n%sKredensial Root Database untuk menghapus database '%s':%s\n", logger.ColorYellow, dbName, logger.ColorReset)
			dbRootUser := terminal.ReadPrompt("Username Root Database", "root")
			dbRootPass, _ := terminal.ReadPassword("Password Root Database: ")

			dto := &domain.DeleteAppDTO{
				Name:           appName,
				DBRootUser:     dbRootUser,
				DBRootPassword: dbRootPass,
				Force:          forceFlag,
			}

			return c.appUsecase.Delete(dto)
		},
	}

	deleteCmd.Flags().BoolVarP(&forceFlag, "force", "f", false, "Lewati konfirmasi")
	return deleteCmd
}

func (c *CLI) getAppNamesCompletion() []string {
	apps, err := c.appRepo.FindAll()
	if err != nil {
		return nil
	}
	var names []string
	for _, a := range apps {
		names = append(names, a.Name)
	}
	return names
}
