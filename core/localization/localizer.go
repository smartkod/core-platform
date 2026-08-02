package localization

// Localizer ana localization modÃƒÆ’Ã†â€™Ãƒâ€šÃ‚Â¼lÃƒÆ’Ã†â€™Ãƒâ€šÃ‚Â¼
type Localizer struct {
    loader      *Loader
    translator  *Translator
    middleware  *Middleware
    config      Config
}

// Config localization yapÃƒÆ’Ã¢â‚¬ÂÃƒâ€šÃ‚Â±landÃƒÆ’Ã¢â‚¬ÂÃƒâ€šÃ‚Â±rmasÃƒÆ’Ã¢â‚¬ÂÃƒâ€šÃ‚Â±
type Config struct {
    LocalesPath     string
    DefaultLocale   string
    SupportedLocales []string
}

// DefaultConfig varsayÃƒÆ’Ã¢â‚¬ÂÃƒâ€šÃ‚Â±lan yapÃƒÆ’Ã¢â‚¬ÂÃƒâ€šÃ‚Â±landÃƒÆ’Ã¢â‚¬ÂÃƒâ€šÃ‚Â±rma
func DefaultConfig() Config {
    return Config{
        LocalesPath:     "core/localization/locales",
        DefaultLocale:   "en-SG",
        SupportedLocales: []string{"tr", "en", "de", "es", "it", "fr", "en-SG", "zh-SG"},
    }
}

// NewLocalizer yeni localizer oluÃƒÆ’Ã¢â‚¬Â¦Ãƒâ€¦Ã‚Â¸turur
func NewLocalizer(config Config) (*Localizer, error) {
    // Loader oluÃƒÆ’Ã¢â‚¬Â¦Ãƒâ€¦Ã‚Â¸tur
    loader := NewLoader(config.LocalesPath)
    if err := loader.Load(); err != nil {
        return nil, err
    }

    // Translator oluÃƒÆ’Ã¢â‚¬Â¦Ãƒâ€¦Ã‚Â¸tur
    translator := NewTranslator(loader)

    // Middleware oluÃƒÆ’Ã¢â‚¬Â¦Ãƒâ€¦Ã‚Â¸tur
    middleware := NewMiddleware(translator, config.DefaultLocale, config.SupportedLocales)

    return &Localizer{
        loader:     loader,
        translator: translator,
        middleware: middleware,
        config:     config,
    }, nil
}

// GetMiddleware middleware'i dÃƒÆ’Ã†â€™Ãƒâ€šÃ‚Â¶ndÃƒÆ’Ã†â€™Ãƒâ€šÃ‚Â¼rÃƒÆ’Ã†â€™Ãƒâ€šÃ‚Â¼r
func (l *Localizer) GetMiddleware() *Middleware {
    return l.middleware
}

// GetTranslator translator'ÃƒÆ’Ã¢â‚¬ÂÃƒâ€šÃ‚Â± dÃƒÆ’Ã†â€™Ãƒâ€šÃ‚Â¶ndÃƒÆ’Ã†â€™Ãƒâ€šÃ‚Â¼rÃƒÆ’Ã†â€™Ãƒâ€šÃ‚Â¼r
func (l *Localizer) GetTranslator() *Translator {
    return l.translator
}

// GetLoader loader'ÃƒÆ’Ã¢â‚¬ÂÃƒâ€šÃ‚Â± dÃƒÆ’Ã†â€™Ãƒâ€šÃ‚Â¶ndÃƒÆ’Ã†â€™Ãƒâ€šÃ‚Â¼rÃƒÆ’Ã†â€™Ãƒâ€šÃ‚Â¼r
func (l *Localizer) GetLoader() *Loader {
    return l.loader
}

// GetSupportedLocales desteklenen dilleri dÃƒÆ’Ã†â€™Ãƒâ€šÃ‚Â¶ndÃƒÆ’Ã†â€™Ãƒâ€šÃ‚Â¼rÃƒÆ’Ã†â€™Ãƒâ€šÃ‚Â¼r
func (l *Localizer) GetSupportedLocales() []string {
    return l.config.SupportedLocales
}

// GetDefaultLocale varsayÃƒÆ’Ã¢â‚¬ÂÃƒâ€šÃ‚Â±lan dili dÃƒÆ’Ã†â€™Ãƒâ€šÃ‚Â¶ndÃƒÆ’Ã†â€™Ãƒâ€šÃ‚Â¼rÃƒÆ’Ã†â€™Ãƒâ€šÃ‚Â¼r
func (l *Localizer) GetDefaultLocale() string {
    return l.config.DefaultLocale
}
