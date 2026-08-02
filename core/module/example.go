package module

import (
"core-platform/core/logger"
)

// ExampleModule ÃƒÂ¶rnek modÃƒÂ¼l
type ExampleModule struct {
*BaseModule
}

// NewExampleModule ÃƒÂ¶rnek modÃƒÂ¼l oluÃ…Å¸turur
func NewExampleModule(logger *logger.Logger) *ExampleModule {
base := NewBaseModule("example", "1.0.0", StateEnabled, logger)
// BaÃ„Å¸Ã„Â±mlÃ„Â±lÃ„Â±klarÃ„Â± kaldÃ„Â±r - Ã…Å¸imdilik boÃ…Å¸
base.SetDependsOn([]string{})

return &ExampleModule{
BaseModule: base,
}
}
