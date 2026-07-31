package localization

// Localizer ana localization modÃƒÂ¼lÃƒÂ¼
type Localizer struct {
    loader      *Loader
    translator  *Translator
    middleware  *Middleware
    config      Config
}

// Config localization yapÃ„Â±landÃ„Â±rmasÃ„Â±
type Config struct {
    LocalesPath     string
    DefaultLocale   string
    SupportedLocales []string
}

// DefaultConfig varsayÃ„Â±lan yapÃ„Â±landÃ„Â±rma
func DefaultConfig() Config {
    return Config{
        LocalesPath:     "core/localization/locales",
        DefaultLocale:   "en-SG",
        SupportedLocales: []string{"tr", "en", "de", "es", "it", "fr", "en-SG", "zh-SG"},
    }
}

// NewLocalizer yeni localizer oluÃ…Å¸turur
func NewLocalizer(config Config) (*Localizer, error) {
    // Loader oluÃ…Å¸tur
    loader := NewLoader(config.LocalesPath)
    if err := loader.Load(); err != nil {
        return nil, err
    }

    // Translator oluÃ…Å¸tur
    translator := NewTranslator(loader)

    // Middleware oluÃ…Å¸tur
    middleware := NewMiddleware(translator, config.DefaultLocale, config.SupportedLocales)

    return &Localizer{
        loader:     loader,
        translator: translator,
        middleware: middleware,
        config:     config,
    }, nil
}

// GetMiddleware middleware'i dÃƒÂ¶ndÃƒÂ¼rÃƒÂ¼r
func (l *Localizer) GetMiddleware() *Middleware {
    return l.middleware
}

// GetTranslator translator'Ã„Â± dÃƒÂ¶ndÃƒÂ¼rÃƒÂ¼r
func (l *Localizer) GetTranslator() *Translator {
    return l.translator
}

// GetLoader loader'Ã„Â± dÃƒÂ¶ndÃƒÂ¼rÃƒÂ¼r
func (l *Localizer) GetLoader() *Loader {
    return l.loader
}

// GetSupportedLocales desteklenen dilleri dÃƒÂ¶ndÃƒÂ¼rÃƒÂ¼r
func (l *Localizer) GetSupportedLocales() []string {
    return l.config.SupportedLocales
}

// GetDefaultLocale varsayÃ„Â±lan dili dÃƒÂ¶ndÃƒÂ¼rÃƒÂ¼r
func (l *Localizer) GetDefaultLocale() string {
    return l.config.DefaultLocale
}
