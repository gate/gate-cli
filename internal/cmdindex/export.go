package cmdindex

// CLICommandLine formats a slash-free gate-cli invocation prefix.
func CLICommandLine(path string) string {
	return cliCommandLine(path)
}
