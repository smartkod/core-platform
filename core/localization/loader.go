package localization

import (
    "encoding/json"
    "fmt"
    "io/ioutil"
    "path/filepath"
)

// Loader çeviri dosyalarını yükler
type Loader struct {
    localesPath string
    translations map[string]map[string]interface{}
}

// NewLoader yeni loader oluşturur
func NewLoader(localesPath string) *Loader {
    return &Loader{
        localesPath: localesPath,
        translations: make(map[string]map[string]interface{}),
    }
}

// Load tüm çeviri dosyalarını yükler
func (l *Loader) Load() error {
    files, err := ioutil.ReadDir(l.localesPath)
    if err != nil {
        return fmt.Errorf("failed to read locales directory: %w", err)
    }

    for _, file := range files {
        if file.IsDir() || filepath.Ext(file.Name()) != ".json" {
            continue
        }

        locale := file.Name()[:len(file.Name())-5] // .json uzantısını kaldır
        path := filepath.Join(l.localesPath, file.Name())

        data, err := ioutil.ReadFile(path)
        if err != nil {
            return fmt.Errorf("failed to read file %s: %w", file.Name(), err)
        }

        var translations map[string]interface{}
        if err := json.Unmarshal(data, &translations); err != nil {
            return fmt.Errorf("failed to parse JSON %s: %w", file.Name(), err)
        }

        l.translations[locale] = translations
    }

    return nil
}

// GetLocaleTranslations locale çevirilerini döndürür
func (l *Loader) GetLocaleTranslations(locale string) map[string]interface{} {
    if translations, ok := l.translations[locale]; ok {
        return translations
    }
    // Fallback to English
    if translations, ok := l.translations["en"]; ok {
        return translations
    }
    return nil
}

// GetAllLocales tüm locale'leri döndürür
func (l *Loader) GetAllLocales() []string {
    locales := make([]string, 0, len(l.translations))
    for locale := range l.translations {
        locales = append(locales, locale)
    }
    return locales
}
