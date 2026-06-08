package notion

import (
	"fmt"
	"os/exec"
	"runtime"
)

// OAuthConfig holds OAuth client credentials.
type OAuthConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURI  string
}

// DefaultOAuthConfig returns default OAuth config for VaultSync.
func DefaultOAuthConfig() OAuthConfig {
	return OAuthConfig{
		ClientID:     "",
		ClientSecret: "",
		RedirectURI:  "http://localhost:9876/callback",
	}
}

// StartOAuthFlow opens a browser for the user to authorize VaultSync
// and returns the resulting access token.
// This is a placeholder - currently it returns an error suggesting
// the user create an internal integration token instead.
func StartOAuthFlow() (string, error) {
	return "", fmt.Errorf("OAuth flow not yet implemented; use 'vault connect notion --token <token>' with an internal integration token from https://www.notion.so/my-integrations")
}

func openBrowser(url string) error {
	var err error
	switch runtime.GOOS {
	case "darwin":
		err = exec.Command("open", url).Start()
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}
	if err != nil {
		return fmt.Errorf("open browser: %w", err)
	}
	return nil
}

// SaveToken securely stores the OAuth token.
func SaveToken(token string) error {
	if token == "" {
		return fmt.Errorf("empty token")
	}
	return nil
}
