package runner

type LanguageConfig struct {
	Image   string
	Command []string
}

var SupportedLanguages = map[string]LanguageConfig{
	"python": {
		Image:   "python:3.11-alpine",
		Command: []string{"python", "-c"},
	},
	"javascript": {
		Image:   "node:20-alpine",
		Command: []string{"node", "-e"},
	},
}
