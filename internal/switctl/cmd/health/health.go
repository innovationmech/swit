package health

import (
	"fmt"
	"net/http"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func NewHealthCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "health",
		Short: "Check the health status of the SWIT server",
		RunE:  runHealthCheck,
	}
}

func runHealthCheck(cmd *cobra.Command, args []string) error {
	serverAddress := viper.GetString("address")
	serverPort := viper.GetString("port")

	url := fmt.Sprintf("http://%s:%s/health", serverAddress, serverPort)

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("health check failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		fmt.Printf("SWIT server (%s:%s) is healthy\n", serverAddress, serverPort)
	} else {
		fmt.Printf("SWIT server (%s:%s) might have issues, status code: %d\n", serverAddress, serverPort, resp.StatusCode)
	}

	return nil
}
