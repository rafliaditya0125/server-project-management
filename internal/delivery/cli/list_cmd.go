package cli

import (
	"fmt"
	"strings"

	"github.com/rafliaditya0125/server-project-management/pkg/logger"
	"github.com/rafliaditya0125/server-project-management/pkg/terminal"
	"github.com/spf13/cobra"
)

func (c *CLI) newListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "Menampilkan daftar aplikasi berserta info database & status service",
		RunE: func(cmd *cobra.Command, args []string) error {
			apps, err := c.appUsecase.List()
			if err != nil {
				return err
			}

			logger.Raw("\n%s%s========================================================================================================%s\n", logger.ColorBold, logger.ColorCyan, logger.ColorReset)
			logger.Raw("%s%s                                       DAFTAR APLIKASI TERISOLASI                                       %s\n", logger.ColorBold, logger.ColorCyan, logger.ColorReset)
			logger.Raw("%s%s========================================================================================================%s\n", logger.ColorBold, logger.ColorCyan, logger.ColorReset)

			if len(apps) == 0 {
				fmt.Println("  (Belum ada aplikasi yang dibuat)")
				fmt.Println("========================================================================================================")
				fmt.Println()
				return nil
			}

			fmt.Println(
				terminal.PadRight("NAMA APLIKASI", 16) +
					terminal.PadRight("STACK", 15) +
					terminal.PadRight("SERVER", 10) +
					terminal.PadRight("PORT(FE/BE)", 18) +
					terminal.PadRight("DATABASE (DB/USER)", 24) +
					terminal.PadRight("STATUS SERVICE", 15),
			)
			fmt.Println(strings.Repeat("-", 104))

			for _, a := range apps {
				var statusFormatted string
				if a.ServiceActive {
					statusFormatted = fmt.Sprintf("%sACTIVE%s", logger.ColorGreen, logger.ColorReset)
				} else {
					statusFormatted = fmt.Sprintf("%sINACTIVE%s", logger.ColorRed, logger.ColorReset)
				}

				dbInfo := fmt.Sprintf("%s / %s", valueOrDefault(a.DBName, "-"), valueOrDefault(a.DBUser, "-"))

				fmt.Println(
					terminal.PadRight(a.Name, 16) +
						terminal.PadRight(string(a.Stack), 15) +
						terminal.PadRight(a.WebServer, 10) +
						terminal.PadRight(a.DisplayPort, 18) +
						terminal.PadRight(dbInfo, 24) +
						statusFormatted,
				)
			}
			fmt.Println("========================================================================================================")
			fmt.Println()

			return nil
		},
	}
}

func valueOrDefault(val, def string) string {
	if val == "" {
		return def
	}
	return val
}
