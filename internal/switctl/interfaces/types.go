// Copyright © 2025 jackelyj <dreamerlyj@gmail.com>
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package interfaces

import (
	"fmt"
	"time"
)

// ServiceConfig represents configuration for service generation.
type ServiceConfig struct {
	Name        string            `yaml:"name" json:"name" validate:"required"`
	Description string            `yaml:"description" json:"description"`
	Author      string            `yaml:"author" json:"author"`
	Version     string            `yaml:"version" json:"version" default:"0.1.0"`
	Features    ServiceFeatures   `yaml:"features" json:"features"`
	Database    DatabaseConfig    `yaml:"database" json:"database"`
	Auth        AuthConfig        `yaml:"auth" json:"auth"`
	Ports       PortConfig        `yaml:"ports" json:"ports"`
	Metadata    map[string]string `yaml:"metadata" json:"metadata"`
	ModulePath  string            `yaml:"module_path" json:"module_path"`
	OutputDir   string            `yaml:"output_dir" json:"output_dir"`
}

// ServiceFeatures represents feature flags for service generation.
type ServiceFeatures struct {
	Database       bool `yaml:"database" json:"database" default:"true"`
	Authentication bool `yaml:"authentication" json:"authentication" default:"false"`
	Cache          bool `yaml:"cache" json:"cache" default:"false"`
	MessageQueue   bool `yaml:"message_queue" json:"message_queue" default:"false"`
	Monitoring     bool `yaml:"monitoring" json:"monitoring" default:"true"`
	Tracing        bool `yaml:"tracing" json:"tracing" default:"true"`
	Logging        bool `yaml:"logging" json:"logging" default:"true"`
	HealthCheck    bool `yaml:"health_check" json:"health_check" default:"true"`
	Metrics        bool `yaml:"metrics" json:"metrics" default:"true"`
	Docker         bool `yaml:"docker" json:"docker" default:"true"`
	Kubernetes     bool `yaml:"kubernetes" json:"kubernetes" default:"false"`
}

// DatabaseConfig represents database configuration.
type DatabaseConfig struct {
	Type     string `yaml:"type" json:"type" default:"mysql"`
	Host     string `yaml:"host" json:"host" default:"localhost"`
	Port     int    `yaml:"port" json:"port" default:"3306"`
	Database string `yaml:"database" json:"database"`
	Username string `yaml:"username" json:"username"`
	Password string `yaml:"password" json:"password"`
	Schema   string `yaml:"schema" json:"schema"`
}

// AuthConfig represents authentication configuration.
type AuthConfig struct {
	Type       string        `yaml:"type" json:"type" default:"jwt"`
	SecretKey  string        `yaml:"secret_key" json:"secret_key"`
	Expiration time.Duration `yaml:"expiration" json:"expiration" default:"15m"`
	Issuer     string        `yaml:"issuer" json:"issuer"`
	Audience   string        `yaml:"audience" json:"audience"`
	Algorithm  string        `yaml:"algorithm" json:"algorithm" default:"HS256"`
}

// PortConfig represents port configuration.
type PortConfig struct {
	HTTP    int `yaml:"http" json:"http" default:"9000"`
	GRPC    int `yaml:"grpc" json:"grpc" default:"10000"`
	Metrics int `yaml:"metrics" json:"metrics" default:"9090"`
	Health  int `yaml:"health" json:"health" default:"8080"`
}

// APIConfig represents configuration for API generation.
type APIConfig struct {
	Name        string            `yaml:"name" json:"name" validate:"required"`
	Version     string            `yaml:"version" json:"version" default:"v1"`
	Service     string            `yaml:"service" json:"service" validate:"required"`
	BasePath    string            `yaml:"base_path" json:"base_path"`
	Methods     []HTTPMethod      `yaml:"methods" json:"methods"`
	GRPCMethods []GRPCMethod      `yaml:"grpc_methods" json:"grpc_methods"`
	Models      []Model           `yaml:"models" json:"models"`
	Middleware  []string          `yaml:"middleware" json:"middleware"`
	Auth        bool              `yaml:"auth" json:"auth" default:"false"`
	Metadata    map[string]string `yaml:"metadata" json:"metadata"`
}

// HTTPMethod represents an HTTP API method.
type HTTPMethod struct {
	Name        string            `yaml:"name" json:"name" validate:"required"`
	Method      string            `yaml:"method" json:"method" validate:"required"`
	Path        string            `yaml:"path" json:"path" validate:"required"`
	Description string            `yaml:"description" json:"description"`
	Request     RequestModel      `yaml:"request" json:"request"`
	Response    ResponseModel     `yaml:"response" json:"response"`
	Auth        bool              `yaml:"auth" json:"auth" default:"false"`
	Middleware  []string          `yaml:"middleware" json:"middleware"`
	Tags        []string          `yaml:"tags" json:"tags"`
	Headers     map[string]string `yaml:"headers" json:"headers"`
}

// GRPCMethod represents a gRPC service method.
type GRPCMethod struct {
	Name        string        `yaml:"name" json:"name" validate:"required"`
	Description string        `yaml:"description" json:"description"`
	Request     RequestModel  `yaml:"request" json:"request"`
	Response    ResponseModel `yaml:"response" json:"response"`
	Streaming   StreamingType `yaml:"streaming" json:"streaming"`
}

// StreamingType represents the streaming type for gRPC methods.
type StreamingType string

const (
	StreamingNone   StreamingType = "none"
	StreamingClient StreamingType = "client"
	StreamingServer StreamingType = "server"
	StreamingBidi   StreamingType = "bidi"
)

// RequestModel represents a request model.
type RequestModel struct {
	Type   string            `yaml:"type" json:"type"`
	Fields []Field           `yaml:"fields" json:"fields"`
	Rules  map[string]string `yaml:"rules" json:"rules"`
}

// ResponseModel represents a response model.
type ResponseModel struct {
	Type   string       `yaml:"type" json:"type"`
	Fields []Field      `yaml:"fields" json:"fields"`
	Errors []ErrorModel `yaml:"errors" json:"errors"`
}

// Model represents a data model.
type Model struct {
	Name        string  `yaml:"name" json:"name" validate:"required"`
	Description string  `yaml:"description" json:"description"`
	Fields      []Field `yaml:"fields" json:"fields"`
}

// ModelConfig represents configuration for model generation.
type ModelConfig struct {
	Name         string            `yaml:"name" json:"name" validate:"required"`
	Description  string            `yaml:"description" json:"description"`
	Package      string            `yaml:"package" json:"package"`
	Fields       []Field           `yaml:"fields" json:"fields"`
	Table        string            `yaml:"table" json:"table"`
	TableName    string            `yaml:"table_name" json:"table_name"`
	Database     string            `yaml:"database" json:"database"`
	CRUD         bool              `yaml:"crud" json:"crud" default:"true"`
	GenerateCRUD bool              `yaml:"generate_crud" json:"generate_crud" default:"true"`
	GenerateAPI  bool              `yaml:"generate_api" json:"generate_api" default:"false"`
	Validation   bool              `yaml:"validation" json:"validation" default:"true"`
	Timestamps   bool              `yaml:"timestamps" json:"timestamps" default:"true"`
	SoftDelete   bool              `yaml:"soft_delete" json:"soft_delete" default:"false"`
	Indexes      []Index           `yaml:"indexes" json:"indexes"`
	Relations    []Relation        `yaml:"relations" json:"relations"`
	Metadata     map[string]string `yaml:"metadata" json:"metadata"`
}

// Field represents a model field.
type Field struct {
	Name        string            `yaml:"name" json:"name" validate:"required"`
	Type        string            `yaml:"type" json:"type" validate:"required"`
	Tags        map[string]string `yaml:"tags" json:"tags"`
	Required    bool              `yaml:"required" json:"required" default:"false"`
	Unique      bool              `yaml:"unique" json:"unique" default:"false"`
	Index       bool              `yaml:"index" json:"index" default:"false"`
	Default     interface{}       `yaml:"default" json:"default"`
	Description string            `yaml:"description" json:"description"`
	Validation  []ValidationRule  `yaml:"validation" json:"validation"`
}

// ValidationRule represents a field validation rule.
type ValidationRule struct {
	Type    string      `yaml:"type" json:"type" validate:"required"`
	Value   interface{} `yaml:"value" json:"value"`
	Message string      `yaml:"message" json:"message"`
}

// Index represents a database index.
type Index struct {
	Name   string   `yaml:"name" json:"name" validate:"required"`
	Fields []string `yaml:"fields" json:"fields" validate:"required"`
	Unique bool     `yaml:"unique" json:"unique" default:"false"`
	Type   string   `yaml:"type" json:"type" default:"btree"`
}

// Relation represents a model relationship.
type Relation struct {
	Type       string `yaml:"type" json:"type" validate:"required"`
	Model      string `yaml:"model" json:"model" validate:"required"`
	ForeignKey string `yaml:"foreign_key" json:"foreign_key"`
	LocalKey   string `yaml:"local_key" json:"local_key"`
	PivotTable string `yaml:"pivot_table" json:"pivot_table"`
	Through    string `yaml:"through" json:"through"`
}

// MiddlewareConfig represents configuration for middleware generation.
type MiddlewareConfig struct {
	Name         string                 `yaml:"name" json:"name" validate:"required"`
	Type         string                 `yaml:"type" json:"type" validate:"required"`
	Description  string                 `yaml:"description" json:"description"`
	Package      string                 `yaml:"package" json:"package"`
	Features     []string               `yaml:"features" json:"features"`
	Dependencies []string               `yaml:"dependencies" json:"dependencies"`
	Config       map[string]interface{} `yaml:"config" json:"config"`
	Metadata     map[string]string      `yaml:"metadata" json:"metadata"`
}

// ProjectConfig represents configuration for project initialization.
type ProjectConfig struct {
	Name           string            `yaml:"name" json:"name" validate:"required"`
	Type           ProjectType       `yaml:"type" json:"type"`
	Description    string            `yaml:"description" json:"description"`
	Author         string            `yaml:"author" json:"author"`
	License        string            `yaml:"license" json:"license" default:"MIT"`
	ModulePath     string            `yaml:"module_path" json:"module_path" validate:"required"`
	GoVersion      string            `yaml:"go_version" json:"go_version" default:"1.19"`
	Services       []ServiceConfig   `yaml:"services" json:"services"`
	Infrastructure Infrastructure    `yaml:"infrastructure" json:"infrastructure"`
	Metadata       map[string]string `yaml:"metadata" json:"metadata"`
	OutputDir      string            `yaml:"output_dir" json:"output_dir"`
}

// ProjectType represents the type of project.
type ProjectType string

const (
	ProjectTypeMonolith     ProjectType = "monolith"
	ProjectTypeMicroservice ProjectType = "microservice"
	ProjectTypeLibrary      ProjectType = "library"
	ProjectTypeAPIGateway   ProjectType = "api_gateway"
	ProjectTypeCLI          ProjectType = "cli"
)

// Infrastructure represents infrastructure configuration.
type Infrastructure struct {
	Database         DatabaseConfig         `yaml:"database" json:"database"`
	Cache            CacheConfig            `yaml:"cache" json:"cache"`
	MessageQueue     MessageQueueConfig     `yaml:"message_queue" json:"message_queue"`
	ServiceDiscovery ServiceDiscoveryConfig `yaml:"service_discovery" json:"service_discovery"`
	Monitoring       MonitoringConfig       `yaml:"monitoring" json:"monitoring"`
	Logging          LoggingConfig          `yaml:"logging" json:"logging"`
}

// CacheConfig represents cache configuration.
type CacheConfig struct {
	Type     string `yaml:"type" json:"type" default:"redis"`
	Host     string `yaml:"host" json:"host" default:"localhost"`
	Port     int    `yaml:"port" json:"port" default:"6379"`
	Database int    `yaml:"database" json:"database" default:"0"`
	Password string `yaml:"password" json:"password"`
}

// MessageQueueConfig represents message queue configuration.
type MessageQueueConfig struct {
	Type     string `yaml:"type" json:"type" default:"rabbitmq"`
	Host     string `yaml:"host" json:"host" default:"localhost"`
	Port     int    `yaml:"port" json:"port" default:"5672"`
	Username string `yaml:"username" json:"username"`
	Password string `yaml:"password" json:"password"`
	VHost    string `yaml:"vhost" json:"vhost" default:"/"`
}

// ServiceDiscoveryConfig represents service discovery configuration.
type ServiceDiscoveryConfig struct {
	Type       string `yaml:"type" json:"type" default:"consul"`
	Host       string `yaml:"host" json:"host" default:"localhost"`
	Port       int    `yaml:"port" json:"port" default:"8500"`
	Datacenter string `yaml:"datacenter" json:"datacenter" default:"dc1"`
}

// MonitoringConfig represents monitoring configuration.
type MonitoringConfig struct {
	Prometheus PrometheusConfig `yaml:"prometheus" json:"prometheus"`
	Jaeger     JaegerConfig     `yaml:"jaeger" json:"jaeger"`
}

// PrometheusConfig represents Prometheus configuration.
type PrometheusConfig struct {
	Enabled bool   `yaml:"enabled" json:"enabled" default:"true"`
	Host    string `yaml:"host" json:"host" default:"localhost"`
	Port    int    `yaml:"port" json:"port" default:"9090"`
	Path    string `yaml:"path" json:"path" default:"/metrics"`
}

// JaegerConfig represents Jaeger configuration.
type JaegerConfig struct {
	Enabled bool   `yaml:"enabled" json:"enabled" default:"true"`
	Host    string `yaml:"host" json:"host" default:"localhost"`
	Port    int    `yaml:"port" json:"port" default:"14268"`
}

// LoggingConfig represents logging configuration.
type LoggingConfig struct {
	Level  string `yaml:"level" json:"level" default:"info"`
	Format string `yaml:"format" json:"format" default:"json"`
	Output string `yaml:"output" json:"output" default:"stdout"`
}

// TemplateData represents data passed to templates.
type TemplateData struct {
	Service    ServiceConfig     `json:"service"`
	Package    PackageInfo       `json:"package"`
	Imports    []ImportInfo      `json:"imports"`
	Functions  []FunctionInfo    `json:"functions"`
	Structs    []StructInfo      `json:"structs"`
	Interfaces []InterfaceInfo   `json:"interfaces"`
	Metadata   map[string]string `json:"metadata"`
	Timestamp  time.Time         `json:"timestamp"`
	Author     string            `json:"author"`
	Version    string            `json:"version"`
}

// PackageInfo represents Go package information.
type PackageInfo struct {
	Name       string `json:"name"`
	Path       string `json:"path"`
	ModulePath string `json:"module_path"`
	Version    string `json:"version"`
}

// ImportInfo represents Go import information.
type ImportInfo struct {
	Alias string `json:"alias,omitempty"`
	Path  string `json:"path"`
}

// FunctionInfo represents Go function information.
type FunctionInfo struct {
	Name       string          `json:"name"`
	Receiver   string          `json:"receiver,omitempty"`
	Parameters []ParameterInfo `json:"parameters"`
	Returns    []ReturnInfo    `json:"returns"`
	Body       string          `json:"body"`
	Comment    string          `json:"comment"`
}

// ParameterInfo represents function parameter information.
type ParameterInfo struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// ReturnInfo represents function return information.
type ReturnInfo struct {
	Name string `json:"name,omitempty"`
	Type string `json:"type"`
}

// StructInfo represents Go struct information.
type StructInfo struct {
	Name    string      `json:"name"`
	Fields  []FieldInfo `json:"fields"`
	Tags    []TagInfo   `json:"tags"`
	Comment string      `json:"comment"`
}

// FieldInfo represents struct field information.
type FieldInfo struct {
	Name    string            `json:"name"`
	Type    string            `json:"type"`
	Tags    map[string]string `json:"tags"`
	Comment string            `json:"comment"`
}

// TagInfo represents struct tag information.
type TagInfo struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// InterfaceInfo represents Go interface information.
type InterfaceInfo struct {
	Name    string       `json:"name"`
	Methods []MethodInfo `json:"methods"`
	Comment string       `json:"comment"`
}

// MethodInfo represents interface method information.
type MethodInfo struct {
	Name       string          `json:"name"`
	Parameters []ParameterInfo `json:"parameters"`
	Returns    []ReturnInfo    `json:"returns"`
	Comment    string          `json:"comment"`
}

// ErrorModel represents an error response model.
type ErrorModel struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Status  int    `json:"status"`
}

// PluginConfig represents plugin configuration.
type PluginConfig struct {
	Name     string            `yaml:"name" json:"name"`
	Version  string            `yaml:"version" json:"version"`
	Enabled  bool              `yaml:"enabled" json:"enabled" default:"true"`
	Config   map[string]string `yaml:"config" json:"config"`
	Path     string            `yaml:"path" json:"path"`
	Metadata map[string]string `yaml:"metadata" json:"metadata"`
}

// ProjectInfo represents project information.
type ProjectInfo struct {
	Name          string    `json:"name"`
	Type          string    `json:"type"`
	Version       string    `json:"version"`
	Description   string    `json:"description"`
	Author        string    `json:"author"`
	License       string    `json:"license"`
	ModulePath    string    `json:"module_path"`
	GoVersion     string    `json:"go_version"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	ServicesCount int       `json:"services_count"`
	Dependencies  []string  `json:"dependencies"`
}

// Dependency represents a project dependency.
type Dependency struct {
	Name    string    `json:"name"`
	Version string    `json:"version"`
	Type    string    `json:"type"`
	Direct  bool      `json:"direct"`
	ModPath string    `json:"mod_path"`
	License string    `json:"license,omitempty"`
	Updated time.Time `json:"updated,omitempty"`
}

// CheckStatus represents the status of a check operation.
type CheckStatus string

const (
	CheckStatusPass    CheckStatus = "pass"
	CheckStatusFail    CheckStatus = "fail"
	CheckStatusWarning CheckStatus = "warning"
	CheckStatusSkip    CheckStatus = "skip"
	CheckStatusError   CheckStatus = "error"
)

// CheckResult represents the result of a quality check.
type CheckResult struct {
	Name     string        `json:"name"`
	Status   CheckStatus   `json:"status"`
	Message  string        `json:"message"`
	Details  []CheckDetail `json:"details,omitempty"`
	Duration time.Duration `json:"duration"`
	Score    int           `json:"score,omitempty"`
	MaxScore int           `json:"max_score,omitempty"`
	Fixable  bool          `json:"fixable"`
}

// CheckDetail represents detailed information about a check result.
type CheckDetail struct {
	File     string      `json:"file,omitempty"`
	Line     int         `json:"line,omitempty"`
	Column   int         `json:"column,omitempty"`
	Message  string      `json:"message"`
	Rule     string      `json:"rule,omitempty"`
	Severity string      `json:"severity,omitempty"`
	Context  interface{} `json:"context,omitempty"`
}

// TestResult represents the result of test execution.
type TestResult struct {
	TotalTests   int                 `json:"total_tests"`
	PassedTests  int                 `json:"passed_tests"`
	FailedTests  int                 `json:"failed_tests"`
	SkippedTests int                 `json:"skipped_tests"`
	Coverage     float64             `json:"coverage"`
	Duration     time.Duration       `json:"duration"`
	Failures     []TestFailure       `json:"failures,omitempty"`
	Packages     []PackageTestResult `json:"packages,omitempty"`
}

// TestFailure represents a test failure.
type TestFailure struct {
	Test    string `json:"test"`
	Package string `json:"package"`
	Message string `json:"message"`
	Output  string `json:"output"`
	File    string `json:"file,omitempty"`
	Line    int    `json:"line,omitempty"`
}

// PackageTestResult represents test results for a specific package.
type PackageTestResult struct {
	Package  string        `json:"package"`
	Tests    int           `json:"tests"`
	Passed   int           `json:"passed"`
	Failed   int           `json:"failed"`
	Coverage float64       `json:"coverage"`
	Duration time.Duration `json:"duration"`
}

// SecurityResult represents the result of security scans.
type SecurityResult struct {
	Issues   []SecurityIssue `json:"issues"`
	Severity string          `json:"severity"`
	Score    int             `json:"score"`
	MaxScore int             `json:"max_score"`
	Scanned  int             `json:"scanned"`
	Duration time.Duration   `json:"duration"`
}

// SecurityIssue represents a security issue.
type SecurityIssue struct {
	ID          string  `json:"id"`
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Severity    string  `json:"severity"`
	File        string  `json:"file,omitempty"`
	Line        int     `json:"line,omitempty"`
	CWE         string  `json:"cwe,omitempty"`
	CVSS        float64 `json:"cvss,omitempty"`
	Fix         string  `json:"fix,omitempty"`
}

// CoverageResult represents the result of coverage analysis.
type CoverageResult struct {
	TotalLines   int                     `json:"total_lines"`
	CoveredLines int                     `json:"covered_lines"`
	Coverage     float64                 `json:"coverage"`
	Threshold    float64                 `json:"threshold"`
	Packages     []PackageCoverageResult `json:"packages"`
	Duration     time.Duration           `json:"duration"`
}

// PackageCoverageResult represents coverage results for a specific package.
type PackageCoverageResult struct {
	Package      string  `json:"package"`
	TotalLines   int     `json:"total_lines"`
	CoveredLines int     `json:"covered_lines"`
	Coverage     float64 `json:"coverage"`
}

// ValidationResult represents the result of configuration validation.
type ValidationResult struct {
	Valid    bool              `json:"valid"`
	Errors   []ValidationError `json:"errors,omitempty"`
	Warnings []ValidationError `json:"warnings,omitempty"`
	Duration time.Duration     `json:"duration"`
}

// Error implements the error interface for ValidationResult.
func (vr *ValidationResult) Error() string {
	if vr.Valid {
		return ""
	}

	if len(vr.Errors) == 0 {
		return "validation failed"
	}

	return fmt.Sprintf("validation failed: %s", vr.Errors[0].Message)
}

// ValidationError represents a validation error.
type ValidationError struct {
	Field   string      `json:"field"`
	Message string      `json:"message"`
	Value   interface{} `json:"value,omitempty"`
	Rule    string      `json:"rule,omitempty"`
	Code    string      `json:"code,omitempty"`
}

// AuditResult represents the result of dependency audit.
type AuditResult struct {
	Vulnerabilities []Vulnerability `json:"vulnerabilities"`
	Severity        string          `json:"severity"`
	Scanned         int             `json:"scanned"`
	Duration        time.Duration   `json:"duration"`
}

// Vulnerability represents a dependency vulnerability.
type Vulnerability struct {
	ID          string    `json:"id"`
	Package     string    `json:"package"`
	Version     string    `json:"version"`
	Severity    string    `json:"severity"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	CVSS        float64   `json:"cvss,omitempty"`
	Published   time.Time `json:"published,omitempty"`
	Updated     time.Time `json:"updated,omitempty"`
	Fix         string    `json:"fix,omitempty"`
	URL         string    `json:"url,omitempty"`
}

// CompatibilityResult represents the result of version compatibility check.
type CompatibilityResult struct {
	Compatible      bool                 `json:"compatible"`
	CurrentVersion  string               `json:"current_version"`
	LatestVersion   string               `json:"latest_version"`
	Issues          []CompatibilityIssue `json:"issues,omitempty"`
	MigrationNeeded bool                 `json:"migration_needed"`
	Duration        time.Duration        `json:"duration"`
}

// CompatibilityIssue represents a compatibility issue.
type CompatibilityIssue struct {
	Type       string `json:"type"`
	Component  string `json:"component"`
	Message    string `json:"message"`
	Breaking   bool   `json:"breaking"`
	Fix        string `json:"fix,omitempty"`
	Deprecated bool   `json:"deprecated,omitempty"`
}
