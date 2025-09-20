// Package exitcode defines standardized exit codes for the ChoreoAtlas CLI.
// These codes follow Unix conventions and provide clear semantics for CI/CD integration.
package exitcode

// Exit codes for the ChoreoAtlas CLI
const (
	// OK indicates successful execution
	OK = 0

	// CLIError indicates general CLI errors (invalid arguments, etc.)
	CLIError = 1

	// InputError indicates file not found or parsing errors
	InputError = 2

	// ValidationFailed indicates validation check failures
	ValidationFailed = 3

	// GateFailed indicates gate policy violations
	GateFailed = 4
)