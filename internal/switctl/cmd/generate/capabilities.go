package generate

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/innovationmech/swit/pkg/messaging"
	"github.com/spf13/cobra"
)

// NewCapabilitiesCommand adds "switctl generate capabilities" to output a Markdown parity matrix.
func NewCapabilitiesCommand(config *GenerateConfig) *cobra.Command {
	var out string

	cmd := &cobra.Command{
		Use:   "capabilities",
		Short: "Generate cross-broker capabilities matrix (Markdown)",
		Long:  "Generate a Markdown document summarizing capabilities across supported message brokers.",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Default output path if not specified
			if out == "" {
				out = filepath.Join(config.WorkDir, "docs", "pages", "en", "reference", "capabilities.md")
			} else if !filepath.IsAbs(out) {
				out = filepath.Join(config.WorkDir, out)
			}

			content, err := messaging.GenerateCapabilitiesMarkdown()
			if err != nil {
				return err
			}

			// Dry-run prints to stdout
			if config != nil && config.DryRun {
				fmt.Fprintln(cmd.OutOrStdout(), content)
				return nil
			}

			// Ensure directory exists and write
			if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
				return fmt.Errorf("failed to create output dir: %w", err)
			}
			if err := os.WriteFile(out, []byte(content), 0o644); err != nil {
				return fmt.Errorf("failed to write capabilities file: %w", err)
			}

			if config != nil && config.Verbose {
				fmt.Fprintf(cmd.ErrOrStderr(), "generated capabilities to %s\n", out)
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&out, "file", "f", "", "Output file path (default: docs/pages/en/reference/capabilities.md)")
	return cmd
}
