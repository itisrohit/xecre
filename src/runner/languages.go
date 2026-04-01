package runner

type LanguageConfig struct {
	Image  string
	RunCmd string
}

var SupportedLanguages = map[string]LanguageConfig{
	"python": {
		Image:  "python:3.11-alpine",
		RunCmd: "python -c \"$1\"",
	},
	"javascript": {
		Image:  "node:20-alpine",
		RunCmd: "node -e \"$1\"",
	},
}
