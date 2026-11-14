package cmd

const anthropicAPIKeyVar = "ANTHROPIC_API_KEY"

// checkAnthropicAPIKey warns when the shell exports an API key that Claudio
// would otherwise inherit into the Claude Code process. The secret value is
// never printed.
func checkAnthropicAPIKey(value string, warn func(label, value, msg string)) {
	if value == "" {
		return
	}
	warn(
		anthropicAPIKeyVar,
		"set",
		"external API credential is inherited by Claude Code and may change authentication or billing outside the selected profile",
	)
}
