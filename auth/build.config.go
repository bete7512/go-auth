package goauth


import (
	"errors"
	"fmt"
	"time"

	"github.com/bete7512/go-auth/auth/types"
)

type AuthBuilder struct {
	config types.Config
}

func DefaultConfig() types.Config {
	return types.Config{
		Database: types.DatabaseConfig{
			Type: types.PostgreSQL,
		},
		Server: types.ServerConfig{
			Type: types.GinServer,
		},
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 7 * 24 * time.Hour,
		EnableTwoFactor: false,
		EnableEmailVerification: false,
		EnableSmsVerification: false,
		PasswordPolicy: types.PasswordPolicy{
			MinLength:      4,
			RequireUpper:   false,
			RequireLower:   false,
			RequireNumber:  false,
			RequireSpecial: false,
		},
		CookieSecure: false,
	}
}

func NewBuilder() *AuthBuilder {
	return &AuthBuilder{
		config: DefaultConfig(),
	}
}

func (b *AuthBuilder) WithServer(serverType types.ServerType, basePath string) *AuthBuilder {
	b.config.Server.Type = serverType
	b.config.BasePath = basePath
	return b
}

func (b *AuthBuilder) WithEmailVerification(enabled bool, url string) *AuthBuilder {
	b.config.EnableEmailVerification = enabled
	b.config.EmailVerificationURL = url
	return b
}

func (b *AuthBuilder) WithPasswordReset(url string) *AuthBuilder {
	b.config.PasswordResetURL = url
	return b
}

func (b *AuthBuilder) WithEmailSender(sender types.EmailSender) *AuthBuilder {
	b.config.EmailSender = sender
	return b
}

func (b *AuthBuilder) WithSMSSender(sender types.SMSSender) *AuthBuilder {
	b.config.SMSSender = sender
	return b
}

func (b *AuthBuilder) WithDatabase(config types.DatabaseConfig) *AuthBuilder {
	b.config.Database = config
	return b
}

func (b *AuthBuilder) WithJWT(secret string, accessTTL, refreshTTL time.Duration) *AuthBuilder {
	b.config.JWTSecret = secret
	b.config.AccessTokenTTL = accessTTL
	b.config.RefreshTokenTTL = refreshTTL
	return b
}

func (b *AuthBuilder) WithPasswordPolicy(policy types.PasswordPolicy) *AuthBuilder {
	b.config.PasswordPolicy = policy
	return b
}

func (b *AuthBuilder) WithTwoFactor(enabled bool, method string) *AuthBuilder {
	b.config.EnableTwoFactor = enabled
	b.config.TwoFactorMethod = method
	return b
}

func (b *AuthBuilder) WithProvider(provider types.AuthProvider, config types.ProviderConfig) *AuthBuilder {
	if b.config.Providers.Enabled == nil {
		b.config.Providers.Enabled = make([]types.AuthProvider, 0, 1)
	}
	b.config.Providers.Enabled = append(b.config.Providers.Enabled, provider)

	switch provider {
	case types.Google:
		b.config.Providers.Google = config
	case types.GitHub:
		b.config.Providers.GitHub = config
	case types.Facebook:
		b.config.Providers.Facebook = config
	case types.Microsoft:
		b.config.Providers.Microsoft = config
	case types.Apple:
		b.config.Providers.Apple = config
	}
	return b
}

func (b *AuthBuilder) WithCookie(secure bool, domain string) *AuthBuilder {
	b.config.CookieSecure = secure
	b.config.CookieDomain = domain
	return b
}

func (b *AuthBuilder) Build() (*types.Auth, error) {
	if err := b.validate(); err != nil {
		return nil, err
	}

	return &types.Auth{Config: b.config}, nil
}

func (b *AuthBuilder) validate() error {
	if b.config.Server.Type == "" {
		return errors.New("server type is required")
	}
	if b.config.Database.URL == "" || b.config.Database.Type == "" {
		return errors.New("database configuration is required")
	}
	if b.config.JWTSecret == "" {
		return errors.New("JWT secret is required")
	}
	if b.config.EnableTwoFactor && b.config.TwoFactorMethod == "" {
		return errors.New("2FA method is required when 2FA is enabled")
	}

	return b.validateProviders()
}

func (b *AuthBuilder) validateProviders() error {
	for _, provider := range b.config.Providers.Enabled {
		var config types.ProviderConfig
		
		switch provider {
		case types.Google:
			config = b.config.Providers.Google
		case types.GitHub:
			config = b.config.Providers.GitHub
		case types.Facebook:
			config = b.config.Providers.Facebook
		case types.Microsoft:
			config = b.config.Providers.Microsoft
		case types.Apple:
			config = b.config.Providers.Apple
		default:
			return fmt.Errorf("unsupported provider: %s", provider)
		}

		if config.ClientID == "" || config.ClientSecret == "" {
			return fmt.Errorf("incomplete configuration for provider: %s", provider)
		}
	}
	
	return nil
}
