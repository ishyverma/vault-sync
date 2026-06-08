package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

type Config struct {
	Vault         VaultConfig         `mapstructure:"vault"`
	Sync          SyncConfig          `mapstructure:"sync"`
	Backends      BackendsConfig      `mapstructure:"backends"`
	TUI           TUIConfig           `mapstructure:"tui"`
	Search        SearchConfig        `mapstructure:"search"`
	Notifications NotificationsConfig `mapstructure:"notifications"`
	Hooks         HooksConfig         `mapstructure:"hooks"`
}

type VaultConfig struct {
	Path                 string `mapstructure:"path"`
	Editor               string `mapstructure:"editor"`
	AutoDaily            bool   `mapstructure:"auto_daily"`
	TemplateDir          string `mapstructure:"template_dir"`
	DefaultTemplate      string `mapstructure:"default_template"`
	WordCountInStatusbar bool   `mapstructure:"word_count_in_statusbar"`
}

type SyncConfig struct {
	AutoSync          bool   `mapstructure:"auto_sync"`
	SyncInterval      int    `mapstructure:"sync_interval"`
	ConflictStrategy  string `mapstructure:"conflict_strategy"`
	QueueRetryLimit   int    `mapstructure:"queue_retry_limit"`
	QueueRetryBackoff string `mapstructure:"queue_retry_backoff"`
}

type BackendsConfig struct {
	Notion   NotionConfig   `mapstructure:"notion"`
	Obsidian ObsidianConfig `mapstructure:"obsidian"`
	Git      GitConfig      `mapstructure:"git"`
}

type NotionConfig struct {
	Enabled       bool   `mapstructure:"enabled"`
	Token         string `mapstructure:"token"`
	WorkspaceID   string `mapstructure:"workspace_id"`
	TargetPageID  string `mapstructure:"target_page_id"`
	DatabaseID    string `mapstructure:"database_id"`
	SyncDirection string `mapstructure:"sync_direction"`
}

type ObsidianConfig struct {
	Enabled       bool   `mapstructure:"enabled"`
	VaultPath     string `mapstructure:"vault_path"`
	Subfolder     string `mapstructure:"subfolder"`
	SyncDirection string `mapstructure:"sync_direction"`
	Wikilinks     bool   `mapstructure:"wikilinks"`
}

type GitConfig struct {
	Enabled       bool   `mapstructure:"enabled"`
	RepoPath      string `mapstructure:"repo_path"`
	AutoCommit    bool   `mapstructure:"auto_commit"`
	CommitMessage string `mapstructure:"commit_message"`
	Remote        string `mapstructure:"remote"`
}

type TUIConfig struct {
	Theme        string `mapstructure:"theme"`
	DateFormat   string `mapstructure:"date_format"`
	ListSort     string `mapstructure:"list_sort"`
	PreviewWidth int    `mapstructure:"preview_width"`
}

type SearchConfig struct {
	Fuzzy      bool `mapstructure:"fuzzy"`
	MaxResults int  `mapstructure:"max_results"`
	Highlight  bool `mapstructure:"highlight"`
}

type NotificationsConfig struct {
	SyncSuccess      bool `mapstructure:"sync_success"`
	SyncFailure      bool `mapstructure:"sync_failure"`
	ConflictDetected bool `mapstructure:"conflict_detected"`
}

type HooksConfig struct {
	PreSync    string `mapstructure:"pre_sync"`
	PostSync   string `mapstructure:"post_sync"`
	OnConflict string `mapstructure:"on_conflict"`
}

func DefaultConfig() Config {
	return Config{
		Vault: VaultConfig{
			Path:                 DefaultVaultPath,
			Editor:               DefaultEditor,
			AutoDaily:            true,
			TemplateDir:          DefaultTemplateDir,
			DefaultTemplate:      DefaultDefaultTemplate,
			WordCountInStatusbar: true,
		},
		Sync: SyncConfig{
			AutoSync:          true,
			SyncInterval:      DefaultSyncInterval,
			ConflictStrategy:  DefaultConflictStrat,
			QueueRetryLimit:   DefaultQueueRetryLimit,
			QueueRetryBackoff: "exponential",
		},
		Backends: BackendsConfig{
			Notion: NotionConfig{
				Enabled:       true,
				SyncDirection: "both",
			},
			Obsidian: ObsidianConfig{
				Enabled:       true,
				VaultPath:     "~/Documents/Obsidian/MyVault",
				Subfolder:     DefaultObsidianSubfolder,
				SyncDirection: "both",
				Wikilinks:     DefaultObsidianWikilinks,
			},
			Git: GitConfig{
				Enabled:       DefaultGitAutoCommit,
				RepoPath:      "~/.vault",
				AutoCommit:    DefaultGitAutoCommit,
				CommitMessage: DefaultGitCommitMessage,
			},
		},
		TUI: TUIConfig{
			Theme:        DefaultTheme,
			DateFormat:   DefaultDateFormat,
			ListSort:     DefaultListSort,
			PreviewWidth: 50,
		},
		Search: SearchConfig{
			Fuzzy:      DefaultFuzzy,
			MaxResults: DefaultMaxResults,
			Highlight:  DefaultHighlight,
		},
		Notifications: NotificationsConfig{
			SyncSuccess:      false,
			SyncFailure:      true,
			ConflictDetected: true,
		},
		Hooks: HooksConfig{},
	}
}

func ConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("home dir: %w", err)
	}
	return filepath.Join(home, ".config", "vault"), nil
}

func ConfigPath() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.toml"), nil
}

func VaultDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("home dir: %w", err)
	}
	return filepath.Join(home, ".vault"), nil
}

func Load() (*Config, error) {
	cfgDir, err := ConfigDir()
	if err != nil {
		return nil, err
	}

	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("toml")
	v.AddConfigPath(cfgDir)

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			return nil, fmt.Errorf("config not found at %s: run 'vault init' first", filepath.Join(cfgDir, "config.toml"))
		}
		return nil, fmt.Errorf("read config: %w", err)
	}

	cfg := DefaultConfig()
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	// Migrate legacy token from config to credentials
	if cfg.Backends.Notion.Token != "" {
		creds, credsErr := LoadCredentials()
		if credsErr == nil && creds.NotionToken == "" {
			creds.NotionToken = cfg.Backends.Notion.Token
			SaveCredentials(creds)
		}
		cfg.Backends.Notion.Token = ""
	}

	// Load token from encrypted credentials
	creds, credsErr := LoadCredentials()
	if credsErr == nil && creds.NotionToken != "" {
		cfg.Backends.Notion.Token = creds.NotionToken
	}

	return &cfg, nil
}

func Save(cfg *Config) error {
	cfgDir, err := ConfigDir()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	cfgPath, err := ConfigPath()
	if err != nil {
		return err
	}

	f, err := os.Create(cfgPath)
	if err != nil {
		return fmt.Errorf("create config file: %w", err)
	}
	defer f.Close()

	v := viper.New()
	v.SetConfigType("toml")
	if err := v.MergeConfigMap(structToMap(cfg)); err != nil {
		return fmt.Errorf("merge config: %w", err)
	}

	if err := v.WriteConfigAs(cfgPath); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	return nil
}

func structToMap(cfg *Config) map[string]interface{} {
	return map[string]interface{}{
		"vault": map[string]interface{}{
			"path":                    cfg.Vault.Path,
			"editor":                  cfg.Vault.Editor,
			"auto_daily":              cfg.Vault.AutoDaily,
			"template_dir":            cfg.Vault.TemplateDir,
			"default_template":        cfg.Vault.DefaultTemplate,
			"word_count_in_statusbar": cfg.Vault.WordCountInStatusbar,
		},
		"sync": map[string]interface{}{
			"auto_sync":           cfg.Sync.AutoSync,
			"sync_interval":       cfg.Sync.SyncInterval,
			"conflict_strategy":   cfg.Sync.ConflictStrategy,
			"queue_retry_limit":   cfg.Sync.QueueRetryLimit,
			"queue_retry_backoff": cfg.Sync.QueueRetryBackoff,
		},
		"backends": map[string]interface{}{
			"notion": map[string]interface{}{
				"enabled":        cfg.Backends.Notion.Enabled,
				"workspace_id":   cfg.Backends.Notion.WorkspaceID,
				"target_page_id": cfg.Backends.Notion.TargetPageID,
				"database_id":    cfg.Backends.Notion.DatabaseID,
				"sync_direction": cfg.Backends.Notion.SyncDirection,
			},
			"obsidian": map[string]interface{}{
				"enabled":        cfg.Backends.Obsidian.Enabled,
				"vault_path":     cfg.Backends.Obsidian.VaultPath,
				"subfolder":      cfg.Backends.Obsidian.Subfolder,
				"sync_direction": cfg.Backends.Obsidian.SyncDirection,
				"wikilinks":      cfg.Backends.Obsidian.Wikilinks,
			},
			"git": map[string]interface{}{
				"enabled":        cfg.Backends.Git.Enabled,
				"repo_path":      cfg.Backends.Git.RepoPath,
				"auto_commit":    cfg.Backends.Git.AutoCommit,
				"commit_message": cfg.Backends.Git.CommitMessage,
				"remote":         cfg.Backends.Git.Remote,
			},
		},
		"tui": map[string]interface{}{
			"theme":         cfg.TUI.Theme,
			"date_format":   cfg.TUI.DateFormat,
			"list_sort":     cfg.TUI.ListSort,
			"preview_width": cfg.TUI.PreviewWidth,
		},
		"search": map[string]interface{}{
			"fuzzy":       cfg.Search.Fuzzy,
			"max_results": cfg.Search.MaxResults,
			"highlight":   cfg.Search.Highlight,
		},
		"notifications": map[string]interface{}{
			"sync_success":      cfg.Notifications.SyncSuccess,
			"sync_failure":      cfg.Notifications.SyncFailure,
			"conflict_detected": cfg.Notifications.ConflictDetected,
		},
		"hooks": map[string]interface{}{
			"pre_sync":    cfg.Hooks.PreSync,
			"post_sync":   cfg.Hooks.PostSync,
			"on_conflict": cfg.Hooks.OnConflict,
		},
	}
}
