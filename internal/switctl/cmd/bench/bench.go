package bench

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/spf13/cobra"

	"github.com/innovationmech/swit/pkg/messaging"
	"github.com/innovationmech/swit/pkg/messaging/benchmark"
)

const (
	defaultOutputDir = "docs/_output/benchmarks"
)

// NewBenchCommand creates the `switctl bench` command that executes cross-broker benchmarks.
func NewBenchCommand() *cobra.Command {
	var (
		configPath   string
		csvPath      string
		markdownPath string
		outputDir    string
		profile      string
		timeoutFlag  string
		printReport  bool
	)

	cmd := &cobra.Command{
		Use:   "bench",
		Short: "Run cross-broker messaging benchmarks",
		Long: `Execute standardized throughput and latency benchmarks across configured message broker adapters.

Examples:
  # Run benchmarks using a YAML config and write CSV/Markdown outputs
  switctl bench --config configs/benchmarks/messaging.yaml --csv results.csv --markdown results.md

  # Inspect results in the terminal without writing files
  switctl bench --config configs/benchmarks/messaging.yaml --print`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(configPath) == "" {
				return errors.New("--config is required to run benchmarks")
			}

			if outputDir == "" {
				outputDir = defaultOutputDir
			}

			opts := runOptions{
				ConfigPath:   configPath,
				OutputDir:    outputDir,
				CSVPath:      csvPath,
				MarkdownPath: markdownPath,
				Profile:      profile,
				TimeoutFlag:  timeoutFlag,
				PrintReport:  printReport,
			}

			return runBench(cmd.Context(), cmd, opts)
		},
	}

	cmd.Flags().StringVar(&configPath, "config", "", "Path to benchmark configuration file (YAML)")
	cmd.Flags().StringVar(&csvPath, "csv", "", "Relative or absolute path for CSV output (defaults to output-dir/benchmarks.csv)")
	cmd.Flags().StringVar(&markdownPath, "markdown", "", "Relative or absolute path for Markdown output (defaults to output-dir/benchmarks.md)")
	cmd.Flags().StringVar(&outputDir, "output-dir", defaultOutputDir, "Directory where benchmark artifacts are written")
	cmd.Flags().StringVar(&profile, "profile", "default", "Workload profile to use when config omits workloads (default|light|balanced|burst)")
	cmd.Flags().StringVar(&timeoutFlag, "timeout", "", "Per-target timeout override (e.g. 2m, 30s)")
	cmd.Flags().BoolVar(&printReport, "print", false, "Print reports to stdout in addition to writing files")

	return cmd
}

type runOptions struct {
	ConfigPath   string
	OutputDir    string
	CSVPath      string
	MarkdownPath string
	Profile      string
	TimeoutFlag  string
	PrintReport  bool
}

type fileConfig struct {
	Timeout   string          `yaml:"timeout"`
	Brokers   []brokerEntry   `yaml:"brokers"`
	Workloads []workloadEntry `yaml:"workloads"`
}

type brokerEntry struct {
	Name        string                  `yaml:"name"`
	Description string                  `yaml:"description"`
	TopicPrefix string                  `yaml:"topic_prefix"`
	Config      *messaging.BrokerConfig `yaml:"config"`
}

type workloadEntry struct {
	Name        string `yaml:"name"`
	Topic       string `yaml:"topic"`
	Messages    int    `yaml:"messages"`
	MessageSize int    `yaml:"message_size"`
	Publishers  int    `yaml:"publishers"`
	BatchSize   int    `yaml:"batch_size"`
}

func runBench(ctx context.Context, cmd *cobra.Command, opts runOptions) error {
	cfg, err := loadConfig(opts.ConfigPath)
	if err != nil {
		return err
	}

	targets, workloads, timeout, err := cfg.toSuiteInputs(opts.Profile)
	if err != nil {
		return err
	}

	if opts.TimeoutFlag != "" {
		parsed, perr := time.ParseDuration(opts.TimeoutFlag)
		if perr != nil {
			return fmt.Errorf("invalid --timeout value: %w", perr)
		}
		timeout = parsed
	}

	if len(targets) == 0 {
		return errors.New("configuration must specify at least one broker target")
	}

	suite := benchmark.Suite{
		Factory:   messaging.GetDefaultFactory(),
		Targets:   targets,
		Workloads: workloads,
		Timeout:   timeout,
	}

	report, err := suite.Run(ctx)
	if err != nil {
		return err
	}

	if opts.PrintReport {
		fmt.Fprintln(cmd.OutOrStdout(), report.ToMarkdown())
		fmt.Fprintln(cmd.OutOrStdout())
		fmt.Fprintln(cmd.OutOrStdout(), report.ToCSV())
	}

	if err := ensureDir(opts.OutputDir); err != nil {
		return err
	}

	if err := writeArtifacts(report, opts); err != nil {
		return err
	}

	fmt.Fprintf(cmd.ErrOrStderr(), "benchmarks completed for %d targets and %d workloads\n", len(targets), len(workloads))
	return nil
}

func loadConfig(path string) (*fileConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed reading config %s: %w", path, err)
	}

	var cfg fileConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed parsing config %s: %w", path, err)
	}

	return &cfg, nil
}

func (cfg *fileConfig) toSuiteInputs(profile string) ([]benchmark.Target, []benchmark.Workload, time.Duration, error) {
	targets := make([]benchmark.Target, 0, len(cfg.Brokers))
	for _, broker := range cfg.Brokers {
		if broker.Config == nil {
			return nil, nil, 0, fmt.Errorf("broker %q missing config", broker.Name)
		}
		broker.Config.SetDefaults()
		if err := broker.Config.Validate(); err != nil {
			return nil, nil, 0, fmt.Errorf("broker %q has invalid configuration: %w", broker.Name, err)
		}
		targets = append(targets, benchmark.Target{
			Name:        broker.Name,
			Description: broker.Description,
			Config:      broker.Config,
			TopicPrefix: broker.TopicPrefix,
		})
	}

	var workloads []benchmark.Workload
	if len(cfg.Workloads) > 0 {
		workloads = make([]benchmark.Workload, 0, len(cfg.Workloads))
		for _, wl := range cfg.Workloads {
			workloads = append(workloads, benchmark.Workload{
				Name:        wl.Name,
				Topic:       wl.Topic,
				Messages:    wl.Messages,
				MessageSize: wl.MessageSize,
				Publishers:  wl.Publishers,
				BatchSize:   wl.BatchSize,
			})
		}
	} else {
		workloads = defaultProfile(profile)
	}

	var timeout time.Duration
	if cfg.Timeout != "" {
		parsed, err := time.ParseDuration(cfg.Timeout)
		if err != nil {
			return nil, nil, 0, fmt.Errorf("invalid timeout in config: %w", err)
		}
		timeout = parsed
	}

	return targets, workloads, timeout, nil
}

func defaultProfile(profile string) []benchmark.Workload {
	switch strings.ToLower(strings.TrimSpace(profile)) {
	case "light":
		return []benchmark.Workload{{
			Name:        "light",
			Messages:    500,
			MessageSize: 512,
			Publishers:  1,
			BatchSize:   1,
		}}
	case "balanced":
		return []benchmark.Workload{{
			Name:        "balanced",
			Messages:    2000,
			MessageSize: 1024,
			Publishers:  4,
			BatchSize:   1,
		}}
	case "burst":
		return []benchmark.Workload{{
			Name:        "burst",
			Messages:    5000,
			MessageSize: 4096,
			Publishers:  8,
			BatchSize:   10,
		}}
	default:
		return benchmark.DefaultWorkloads()
	}
}

func ensureDir(dir string) error {
	if dir == "" {
		return nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed creating output directory %s: %w", dir, err)
	}
	return nil
}

func writeArtifacts(report *benchmark.Report, opts runOptions) error {
	if report == nil {
		return errors.New("nil report")
	}

	csvPath := resolvePath(opts.OutputDir, opts.CSVPath, "benchmarks.csv")
	markdownPath := resolvePath(opts.OutputDir, opts.MarkdownPath, "benchmarks.md")

	if csvPath != "" {
		if err := writeFile(csvPath, report.ToCSV()); err != nil {
			return err
		}
	}
	if markdownPath != "" {
		if err := writeFile(markdownPath, report.ToMarkdown()); err != nil {
			return err
		}
	}
	return nil
}

func resolvePath(baseDir, override, fallback string) string {
	if override == "-" {
		return ""
	}
	target := override
	if target == "" {
		target = filepath.Join(baseDir, fallback)
	} else if !filepath.IsAbs(target) {
		target = filepath.Join(baseDir, target)
	}
	return target
}

func writeFile(path string, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("failed creating directory for %s: %w", path, err)
	}
	if err := os.WriteFile(path, []byte(content), fs.FileMode(0o644)); err != nil {
		return fmt.Errorf("failed writing %s: %w", path, err)
	}
	return nil
}
