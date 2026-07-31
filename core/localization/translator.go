package localization

import (
    "fmt"
    "strings"
    "text/template"
)

// Translator çeviri işlemlerini yapar
type Translator struct {
    loader *Loader
}

// NewTranslator yeni translator oluşturur
func NewTranslator(loader *Loader) *Translator {
    return &Translator{
        loader: loader,
    }
}

// Translate key'e göre çeviri yapar
func (t *Translator) Translate(locale, key string, data interface{}) string {
    translations := t.loader.GetLocaleTranslations(locale)
    if translations == nil {
        return key
    }

    value := getNestedValue(translations, key)
    if value == nil {
        return key
    }

    str, ok := value.(string)
    if !ok {
        return fmt.Sprintf("%v", value)
    }

    // Template değişkenlerini işle
    if data != nil && strings.Contains(str, "{{") {
        tmpl, err := template.New("trans").Parse(str)
        if err != nil {
            return str
        }

        var result strings.Builder
        if err := tmpl.Execute(&result, data); err != nil {
            return str
        }
        return result.String()
    }

    return str
}

// getNestedValue nested key'den değer alır
func getNestedValue(data map[string]interface{}, key string) interface{} {
    parts := strings.Split(key, ".")
    current := data

    for i, part := range parts {
        if i == len(parts)-1 {
            return current[part]
        }

        if next, ok := current[part].(map[string]interface{}); ok {
            current = next
        } else {
            return nil
        }
    }

    return nil
}
