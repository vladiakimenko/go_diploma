package auth

import (
	"context"
	"errors"
	"unicode"
	"unicode/utf8"

	"github.com/tailscale/golang-x-crypto/bcrypt"

	"blog-api/pkg/settings"
)

// constants
const maxPasswordBytes = 72

// errors
var (
	ErrPasswordTooShort = errors.New("password is too short")
	ErrPasswordTooLong  = errors.New("password is too long")
	ErrPasswordTooWeak  = errors.New("password too weak")
)

// config
type PasswordConfig struct {
	MinLength         int
	Cost              int
	CaseShiftRequired bool
	DigitsRequired    bool
	SymbolsRequired   bool
}

func (c *PasswordConfig) Setup() []settings.EnvLoadable {
	return []settings.EnvLoadable{
		settings.Item[int]{Name: "PASSWORD_MIN_LENGTH", Default: 6, Field: &c.MinLength},
		settings.Item[int]{Name: "PASSWORD_COST", Default: bcrypt.DefaultCost, Field: &c.Cost},
		settings.Item[bool]{Name: "PASSWORD_MUST_SHIFT_CASE", Default: true, Field: &c.CaseShiftRequired},
		settings.Item[bool]{Name: "PASSWORD_MUST_HAVE_DIGITS", Default: true, Field: &c.DigitsRequired},
		settings.Item[bool]{Name: "PASSWORD_MUST_HAVE_SYMBOLS", Default: true, Field: &c.SymbolsRequired},
	}
}

// manager
type PasswordManager struct {
	config *PasswordConfig
}

func NewPasswordManager(config *PasswordConfig) *PasswordManager {
	if config == nil {
		panic("PasswordManager requires a non-nil config")
	}
	return &PasswordManager{config: config}
}

func (pm *PasswordManager) ValidatePasswordStrength(password string) error {
	if utf8.RuneCountInString(password) < pm.config.MinLength {
		return ErrPasswordTooShort
	}

	if len(password) > maxPasswordBytes {
		return ErrPasswordTooLong
	}

	var hasUpper, hasLower, hasDigit, hasSymbol bool
	for _, r := range password {
		if !hasUpper && unicode.IsUpper(r) {
			hasUpper = true
		}
		if !hasLower && unicode.IsLower(r) {
			hasLower = true
		}
		if !hasDigit && unicode.IsDigit(r) {
			hasDigit = true
		}
		if !hasSymbol && (unicode.IsSymbol(r) || unicode.IsPunct(r)) {
			hasSymbol = true
		}
		if (!pm.config.CaseShiftRequired || (hasUpper && hasLower)) &&
			(!pm.config.DigitsRequired || hasDigit) &&
			(!pm.config.SymbolsRequired || hasSymbol) {
			break
		}
	}
	if (pm.config.CaseShiftRequired && (!hasUpper || !hasLower)) ||
		(pm.config.DigitsRequired && !hasDigit) ||
		(pm.config.SymbolsRequired && !hasSymbol) {
		return ErrPasswordTooWeak
	}

	return nil
}

func (pm *PasswordManager) HashPassword(ctx context.Context, password string) (string, error) {
	if err := pm.ValidatePasswordStrength(password); err != nil {
		return "", err
	}

	done := make(chan struct{})
	var passwordHash []byte
	var err error

	go func() {
		passwordHash, err = bcrypt.GenerateFromPassword([]byte(password), pm.config.Cost)
		close(done)
	}()

	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case <-done:
		if err != nil {
			return "", err
		}
		return string(passwordHash), nil
	}
}

func (pm *PasswordManager) CheckPassword(password, hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}
