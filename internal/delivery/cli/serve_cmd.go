package cli

import (
	"fmt"
	"net/http"

	deliveryHTTP "github.com/rafliaditya0125/server-project-management/internal/delivery/http"
	"github.com/rafliaditya0125/server-project-management/internal/delivery/http/handler"
	"github.com/rafliaditya0125/server-project-management/pkg/logger"
	"github.com/spf13/cobra"
)

func (c *CLI) newServeCmd() *cobra.Command {
	var port string

	serveCmd := &cobra.Command{
		Use:   "serve",
		Short: "Menjalankan REST HTTP API Server untuk Project Management",
		RunE: func(cmd *cobra.Command, args []string) error {
			if port == "" {
				port = c.httpPort
			}
			if port == "" {
				port = ":8080"
			}
			if port[0] != ':' {
				port = ":" + port
			}

			appHandler := handler.NewAppHandler(c.appUsecase, c.serviceUsecase)
			setupHandler := handler.NewSetupHandler(c.setupUsecase)
			router := deliveryHTTP.NewRouter(appHandler, setupHandler)

			logger.Success("Menjalankan HTTP API Server pada port %s...", port)
			fmt.Printf("Health check: http://localhost%s/health\n", port)
			fmt.Printf("API endpoint: http://localhost%s/api/v1/apps\n", port)

			return http.ListenAndServe(port, router)
		},
	}

	serveCmd.Flags().StringVarP(&port, "port", "p", ":8080", "Port untuk HTTP REST API Server")
	return serveCmd
}
