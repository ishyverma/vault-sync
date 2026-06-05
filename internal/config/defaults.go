package config

const (
	DefaultVaultPath       = "~/.vault/notes"
	DefaultEditor          = "nvim"
	DefaultTemplateDir     = "~/.vault/templates"
	DefaultDefaultTemplate = "blank"
	DefaultSyncInterval    = 60
	DefaultConflictStrat   = "ask"
	DefaultQueueRetryLimit = 5
	DefaultTheme           = "dark"
	DefaultDateFormat      = "2006-01-02"
	DefaultListSort        = "modified"
	DefaultMaxResults      = 50
	DefaultFuzzy           = true
	DefaultHighlight       = true

	DefaultObsidianSubfolder = "VaultSync"
	DefaultObsidianWikilinks = true

	DefaultGitAutoCommit    = false
	DefaultGitCommitMessage = "vault: sync {filename}"
)
