# Switctl Interactive UI Components

This package provides beautiful and interactive user interface components for the Swit framework scaffolding tool (switctl).

## Overview

The UI components are designed to create an engaging and user-friendly experience for developers using switctl to generate services, check code quality, and manage projects.

## Components

### 1. Terminal UI (`terminal.go`)

The core terminal-based UI implementation providing:

- **Color-coded output** with themes for different message types
- **Progress bars** with customizable messages and completion tracking
- **Spinner animations** for long-running operations
- **Table rendering** with proper alignment and borders
- **Styled headers and separators** for visual organization
- **Cursor control** and terminal manipulation

#### Features:
- ✅ Success messages with green checkmarks
- ❌ Error messages with red X marks
- ⚠️ Warning messages with yellow warning signs
- ℹ️ Info messages with blue info icons
- 🚀 Welcome screen with ASCII art
- Progress bars with Unicode block characters
- Beautiful table rendering with box-drawing characters

### 2. Interactive Prompts (`prompt.go`)

Comprehensive input collection with validation:

- **Text input prompts** with real-time validation
- **Confirmation dialogs** (yes/no) with defaults
- **Single selection** from option lists
- **Multi-selection** with comma-separated input
- **Password input** (basic implementation)

#### Built-in Validators:
- `ServiceNameValidator` - Validates service names (lowercase, hyphens)
- `PackageNameValidator` - Validates Go package names
- `PortValidator` - Validates port numbers (1024-65535)
- `EmailValidator` - Validates email addresses
- `URLValidator` - Validates HTTP/HTTPS URLs
- `ModulePathValidator` - Validates Go module paths
- `NonEmptyValidator` - Ensures non-empty input

#### Validator Builders:
- `CreateRangeValidator(min, max)` - Numeric range validation
- `CreateLengthValidator(min, max)` - String length validation
- `CreateRegexValidator(pattern, message)` - Custom regex validation
- `CreateChoiceValidator(choices, caseSensitive)` - Predefined choices
- `CombineValidators(validators...)` - Combine multiple validators

### 3. Menu System (`menu.go`)

Interactive menu components with keyboard navigation:

- **Single-select menus** with numbered options
- **Multi-select menus** with comma-separated selection
- **Progress menus** for multi-step operations
- **Confirmation menus** for yes/no decisions

#### Specialized Menus:
- `ShowFeatureSelectionMenu()` - Service feature selection
- `ShowDatabaseSelectionMenu()` - Database type selection
- `ShowAuthSelectionMenu()` - Authentication type selection

#### Menu Option Builders:
- `BuildMenuOption()` - Full option with label, description, value, icon
- `BuildSimpleMenuOption()` - Simple option with label and value
- `BuildMenuOptions()` - Create options from map
- `BuildStringMenuOptions()` - Create options from string slice

## Usage Examples

### Basic Terminal UI

```go
ui := NewTerminalUI()

// Show welcome screen
ui.ShowWelcome()

// Display messages
ui.ShowSuccess("Operation completed!")
ui.ShowError(errors.New("Something went wrong"))
ui.ShowWarning("This is a warning")
ui.ShowInfo("Here's some information")

// Progress tracking
pb := ui.ShowProgress("Processing files", 100)
pb.Update(50)
pb.SetMessage("Half way there...")
pb.Finish()
```

### Input Collection

```go
// Text input with validation
serviceName, err := ui.PromptInput("Enter service name", ServiceNameValidator)

// Confirmation
confirmed, err := ui.PromptConfirm("Continue with operation", true)

// Selection from options
options := []string{"MySQL", "PostgreSQL", "MongoDB"}
index, selection, err := ui.PromptSelect("Choose database", options, 0)

// Multi-selection
indices, selections, err := ui.PromptMultiSelect("Choose features", options)
```

### Menu System

```go
// Simple menu
options := []interfaces.MenuOption{
    BuildMenuOption("Create Service", "Generate a new microservice", "service", "🔧", true),
    BuildMenuOption("Run Checks", "Execute quality checks", "check", "✅", true),
}
selectedIndex, err := ui.ShowMenu("Main Menu", options)

// Feature selection
features, err := ui.ShowFeatureSelectionMenu()
```

### Table Display

```go
headers := []string{"Service", "Status", "Port"}
rows := [][]string{
    {"user-service", "Running", "8080"},
    {"auth-service", "Stopped", "8081"},
}
ui.ShowTable(headers, rows)
```

## Configuration Options

### Terminal UI Options

```go
ui := NewTerminalUI(
    WithOutput(customWriter),     // Custom output writer
    WithInput(customReader),      // Custom input reader
    WithNoColor(true),           // Disable colors
    WithVerbose(true),           // Enable verbose output
    WithStyle(customStyle),      // Custom color scheme
)
```

### UI Styling

```go
style := interfaces.UIStyle{
    Primary:   &ColorImpl{color: color.New(color.FgCyan, color.Bold)},
    Success:   &ColorImpl{color: color.New(color.FgGreen, color.Bold)},
    Warning:   &ColorImpl{color: color.New(color.FgYellow, color.Bold)},
    Error:     &ColorImpl{color: color.New(color.FgRed, color.Bold)},
    Info:      &ColorImpl{color: color.New(color.FgBlue)},
    Highlight: &ColorImpl{color: color.New(color.FgMagenta)},
}
```

## Testing

The package includes comprehensive tests demonstrating:

- Basic UI component functionality
- Input validation and error handling
- Menu navigation and selection
- Progress tracking and completion
- Color and styling features
- Complete workflow examples

Run tests with:
```bash
go test ./internal/switctl/ui/... -v
```

## Dependencies

- `github.com/fatih/color` - Terminal color support
- `github.com/stretchr/testify` - Testing framework

## Design Principles

1. **User-Friendly**: Clear prompts, helpful error messages, and intuitive navigation
2. **Beautiful**: Colorful output, icons, progress indicators, and well-formatted tables
3. **Robust**: Input validation, error handling, and graceful fallbacks
4. **Flexible**: Configurable styling, extensible validators, and modular components
5. **Consistent**: Unified interface design following Swit framework patterns

## Future Enhancements

- Advanced keyboard navigation (arrow keys, tab completion)
- Terminal size detection and responsive layouts
- Plugin system for custom UI components
- Theme management and user preferences
- Advanced password input with hidden characters
- Real-time search and filtering in menus