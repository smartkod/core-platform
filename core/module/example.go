package module

import (
"core-platform/core/logger"
)

// ExampleModule örnek modül
type ExampleModule struct {
*BaseModule
}

// NewExampleModule örnek modül oluşturur
func NewExampleModule(logger *logger.Logger) *ExampleModule {
base := NewBaseModule("example", "1.0.0", StateEnabled, logger)
// Bağımlılıkları kaldır - şimdilik boş
base.SetDependsOn([]string{})

return &ExampleModule{
BaseModule: base,
}
}
