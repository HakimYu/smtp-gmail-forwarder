package gmail

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

// Client wraps Gmail API client with authentication
type Client struct {
	service   *gmail.Service
	config    *oauth2.Config
	tokenFile string
}

// NewClient creates a new Gmail API client
func NewClient(ctx context.Context, credentialsFile, tokenFile string) (*Client, error) {
	// Read credentials file
	credentials, err := os.ReadFile(credentialsFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read credentials file: %w", err)
	}

	// Parse OAuth2 config
	config, err := google.ConfigFromJSON(credentials, gmail.GmailSendScope)
	if err != nil {
		return nil, fmt.Errorf("failed to parse credentials: %w", err)
	}

	// Get token
	token, err := getToken(config, tokenFile)
	if err != nil {
		return nil, fmt.Errorf("failed to get token: %w", err)
	}

	// Create Gmail service
	httpClient := config.Client(ctx, token)
	service, err := gmail.NewService(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		return nil, fmt.Errorf("failed to create Gmail service: %w", err)
	}

	return &Client{
		service:   service,
		config:    config,
		tokenFile: tokenFile,
	}, nil
}

// getToken retrieves a token from file or initiates OAuth2 flow
func getToken(config *oauth2.Config, tokenFile string) (*oauth2.Token, error) {
	// Try to read token from file
	token, err := tokenFromFile(tokenFile)
	if err == nil {
		// Verify token is still valid by attempting to refresh it if needed
		// This catches expired tokens early
		tokenSource := config.TokenSource(context.Background(), token)
		freshToken, err := tokenSource.Token()
		if err == nil {
			// Save the refreshed token if it's different
			if freshToken.AccessToken != token.AccessToken || freshToken.RefreshToken != token.RefreshToken {
				if saveErr := saveToken(tokenFile, freshToken); saveErr != nil {
					fmt.Printf("Warning: could not save refreshed token: %v\n", saveErr)
				}
				return freshToken, nil
			}
			return token, nil
		}
		// If token refresh failed, we'll proceed to get a new token
		fmt.Printf("Token refresh failed: %v, will get new token\n", err)
	}

	// Token not found or invalid, initiate OAuth2 flow
	token, err = getTokenFromWeb(config)
	if err != nil {
		return nil, err
	}

	// Save token to file
	if err := saveToken(tokenFile, token); err != nil {
		return nil, err
	}

	return token, nil
}

// tokenFromFile reads token from file
func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	token := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(token)
	return token, err
}

// getTokenFromWeb initiates OAuth2 flow in browser
func getTokenFromWeb(config *oauth2.Config) (*oauth2.Token, error) {
	stateToken := generateStateToken()
	authURL := config.AuthCodeURL(stateToken, oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser:\n%v\n", authURL)
	fmt.Print("Enter authorization code: ")

	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		return nil, fmt.Errorf("failed to read authorization code: %w", err)
	}

	token, err := config.Exchange(context.Background(), authCode)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange authorization code: %w", err)
	}

	return token, nil
}

// generateStateToken creates a random state token for OAuth2 CSRF protection
func generateStateToken() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// saveToken saves token to file with secure permissions
func saveToken(file string, token *oauth2.Token) error {
	f, err := os.OpenFile(file, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("failed to create token file: %w", err)
	}
	defer f.Close()

	return json.NewEncoder(f).Encode(token)
}

// SendMessage sends an email via Gmail API
func (c *Client) SendMessage(ctx context.Context, rawMessage []byte) error {
	return c.sendMessageWithTokenRefresh(ctx, rawMessage, true)
}

func (c *Client) sendMessageWithTokenRefresh(ctx context.Context, rawMessage []byte, retryOnTokenError bool) error {
	encoded := base64.URLEncoding.EncodeToString(rawMessage)

	message := &gmail.Message{
		Raw: encoded,
	}

	_, err := c.service.Users.Messages.Send("me", message).Context(ctx).Do()
	if err != nil {
		if retryOnTokenError && isTokenError(err) {
			fmt.Printf("Token error detected: %v, attempting to refresh token\n", err)

			token, tokenErr := getToken(c.config, c.tokenFile)
			if tokenErr != nil {
				return fmt.Errorf("failed to get new token: %w (original error: %w)", tokenErr, err)
			}

			httpClient := c.config.Client(ctx, token)
			newService, serviceErr := gmail.NewService(ctx, option.WithHTTPClient(httpClient))
			if serviceErr != nil {
				return fmt.Errorf("failed to create new Gmail service: %w (original error: %w)", serviceErr, err)
			}

			c.service = newService

			return c.sendMessageWithTokenRefresh(ctx, rawMessage, false)
		}
		return fmt.Errorf("failed to send message: %w", err)
	}

	return nil
}

func isTokenError(err error) bool {
	errStr := err.Error()
	return containsAny(errStr, []string{
		"invalid_grant",
		"invalid_token",
		"token expired",
		"access_token",
		"401",
		"403",
	})
}

func containsAny(s string, substrings []string) bool {
	for _, substr := range substrings {
		if containsIgnoreCase(s, substr) {
			return true
		}
	}
	return false
}

func containsIgnoreCase(s, substr string) bool {
	sLower := strings.ToLower(s)
	substrLower := strings.ToLower(substr)
	return strings.Contains(sLower, substrLower)
}
