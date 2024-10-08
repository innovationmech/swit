package stop

import (
	"fmt"
	"net/http"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func NewStopCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stop",
		Short: "Stop the SWIT server",
		RunE: func(cmd *cobra.Command, args []string) error {
			return stopServer()
		},
	}

	return cmd
}

func stopServer() error {
	serverAddress := viper.GetString("address")
	serverPort := viper.GetString("port")

	url := fmt.Sprintf("http://%s:%s/stop", serverAddress, serverPort)
	resp, err := http.Post(url, "application/json", nil)
	if err != nil {
		return fmt.Errorf("failed to stop server: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		fmt.Printf("Stopping server %s:%s\n", serverAddress, serverPort)
	} else {
		return fmt.Errorf("failed to stop server, status code: %d", resp.StatusCode)
	}

	return nil
}
