package oauthhandlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/bete7512/goauth/models"
	"github.com/bete7512/goauth/types"
	"github.com/bete7512/goauth/utils"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type GoogleOauth struct {
	Auth *types.Auth
}

func NewGoogleOauth(auth *types.Auth) *GoogleOauth {
	return &GoogleOauth{
		Auth: auth,
	}
}

// GoogleUserInfo represents the user information returned by Google
type GoogleUserInfo struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	VerifiedEmail bool   `json:"verified_email"`
	Name          string `json:"name"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
	Picture       string `json:"picture"`
	Locale        string `json:"locale"`
}

// getGoogleOAuthConfig creates the OAuth2 config for Google
func (g *GoogleOauth) getGoogleOAuthConfig() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     g.Auth.Config.Providers.Google.ClientID,
		ClientSecret: g.Auth.Config.Providers.Google.ClientSecret,
		RedirectURL:  g.Auth.Config.Providers.Google.RedirectURL,
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		},
		Endpoint: google.Endpoint,
	}
}

// SignIn initiates the Google OAuth flow
func (g *GoogleOauth) SignIn(w http.ResponseWriter, r *http.Request) {
	config := g.getGoogleOAuthConfig()

	// Generate a random state for CSRF protection
	state, err := utils.GenerateRandomToken(32)
	if err != nil {
		http.Error(w, "Failed to generate state", http.StatusInternalServerError)
		return
	}

	// Store state in a cookie
	stateCookie := &http.Cookie{
		Name:     "oauth_state",
		Value:    state,
		Path:     "/",
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(time.Hour.Seconds()),
	}
	http.SetCookie(w, stateCookie)

	// Redirect user to Google's consent page
	url := config.AuthCodeURL(state, oauth2.AccessTypeOffline)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

// Callback handles the OAuth callback from Google
func (g *GoogleOauth) Callback(w http.ResponseWriter, r *http.Request) {
	// Verify state to prevent CSRF
	stateCookie, err := r.Cookie("oauth_state")
	if err != nil {
		http.Error(w, "State cookie not found", http.StatusBadRequest)
		return
	}

	if r.FormValue("state") != stateCookie.Value {
		http.Error(w, "Invalid state parameter", http.StatusBadRequest)
		return
	}

	// Exchange the authorization code for a token
	code := r.FormValue("code")
	config := g.getGoogleOAuthConfig()

	token, err := config.Exchange(context.Background(), code)
	if err != nil {
		http.Error(w, "Failed to exchange token: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Get user info from Google
	userInfo, err := g.getUserInfo(token.AccessToken)
	if err != nil {
		http.Error(w, "Failed to get user info: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Create or update user in your system
	user := models.User{
		Email:      userInfo.Email,
		FirstName:  userInfo.Name,
		SigninVia:  "google",
		ProviderId: &userInfo.ID,
		Avatar:     userInfo.Picture,
	}
	// TODO:
	err = g.Auth.Repository.GetUserRepository().UpsertUserByEmail(&user)
	if err != nil {
		http.Error(w, "Failed to create/update user: "+err.Error(), http.StatusInternalServerError)
		return
	}
	// Generate tokens
	accessToken, refreshToken, err := utils.GenerateTokens(user.ID, g.Auth.Config.AccessTokenTTL, g.Auth.Config.JWTSecret)
	if err != nil {
		http.Error(w, "Failed to generate authentication tokens", http.StatusInternalServerError)
		return
	}
	// Save refresh token
	err = g.Auth.Repository.GetTokenRepository().SaveRefreshToken(user.ID, refreshToken, g.Auth.Config.RefreshTokenTTL)
	if err != nil {
		http.Error(w, "Failed to save refresh token", http.StatusInternalServerError)
		return
	}

	// Set the token in a cookie or return it in the response
	tokenCookie := &http.Cookie{
		Name:     g.Auth.Config.CookieName,
		Value:    accessToken,
		Path:     g.Auth.Config.CookiePath,
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(g.Auth.Config.AccessTokenTTL.Seconds()),
	}
	http.SetCookie(w, tokenCookie)

	// Redirect to the frontend
	http.Redirect(w, r, g.Auth.Config.FrontendURL, http.StatusTemporaryRedirect)
}

// getUserInfo fetches the user information from Google API
func (g *GoogleOauth) getUserInfo(accessToken string) (*GoogleUserInfo, error) {
	resp, err := http.Get("https://www.googleapis.com/oauth2/v2/userinfo?access_token=" + accessToken)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get user info: %s", body)
	}

	var userInfo GoogleUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, err
	}

	return &userInfo, nil
}
