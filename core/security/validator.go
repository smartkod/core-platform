package security

import (
    "regexp"
    "strings"
)

// Validator request validasyonu
type Validator struct{}

// NewValidator yeni validator oluşturur
func NewValidator() *Validator {
    return &Validator{}
}

// ValidateEmail email formatını doğrular
func (v *Validator) ValidateEmail(email string) bool {
    pattern := `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
    matched, _ := regexp.MatchString(pattern, email)
    return matched
}

// ValidatePassword şifre gücünü doğrular
func (v *Validator) ValidatePassword(password string) bool {
    if len(password) < 6 {
        return false
    }
    if !regexp.MustCompile(`[0-9]`).MatchString(password) {
        return false
    }
    if !regexp.MustCompile(`[A-Z]`).MatchString(password) {
        return false
    }
    return true
}

// SanitizeInput input'u temizler
func (v *Validator) SanitizeInput(input string) string {
    input = regexp.MustCompile(`<[^>]*>`).ReplaceAllString(input, "")
    dangerous := []string{"'", "\"", ";", "--", "/*", "*/"}
    for _, char := range dangerous {
        input = strings.ReplaceAll(input, char, "")
    }
    return strings.TrimSpace(input)
}
