package module

import (
"core-platform/core/logger"

"go.uber.org/zap"
)

// BaseModule temel modÃƒÂ¼l implementasyonu
type BaseModule struct {
name      string
state     ModuleState
version   string
logger    *logger.Logger
dependsOn []string
}

// NewBaseModule yeni base module oluÃ…Å¸turur
func NewBaseModule(name string, version string, state ModuleState, logger *logger.Logger) *BaseModule {
return &BaseModule{
name:      name,
state:     state,
version:   version,
logger:    logger,
dependsOn: []string{},
}
}

// Name modÃƒÂ¼l adÃ„Â±nÃ„Â± dÃƒÂ¶ndÃƒÂ¼rÃƒÂ¼r
func (m *BaseModule) Name() string {
return m.name
}

// State modÃƒÂ¼l durumunu dÃƒÂ¶ndÃƒÂ¼rÃƒÂ¼r
func (m *BaseModule) State() ModuleState {
return m.state
}

// Version modÃƒÂ¼l versiyonunu dÃƒÂ¶ndÃƒÂ¼rÃƒÂ¼r
func (m *BaseModule) Version() string {
return m.version
}

// DependsOn baÃ„Å¸Ã„Â±mlÃ„Â±lÃ„Â±klarÃ„Â± dÃƒÂ¶ndÃƒÂ¼rÃƒÂ¼r
func (m *BaseModule) DependsOn() []string {
return m.dependsOn
}

// SetDependsOn baÃ„Å¸Ã„Â±mlÃ„Â±lÃ„Â±klarÃ„Â± ayarlar
func (m *BaseModule) SetDependsOn(deps []string) *BaseModule {
m.dependsOn = deps
return m
}

// Initialize modÃƒÂ¼lÃƒÂ¼ baÃ…Å¸latÃ„Â±r
func (m *BaseModule) Initialize() error {
m.logger.Info("Module initialized",
zap.String("module", m.name),
zap.String("version", m.version),
)
return nil
}

// Start modÃƒÂ¼lÃƒÂ¼ baÃ…Å¸latÃ„Â±r
func (m *BaseModule) Start() error {
m.logger.Info("Module started",
zap.String("module", m.name),
)
return nil
}

// Stop modÃƒÂ¼lÃƒÂ¼ durdurur
func (m *BaseModule) Stop() error {
m.logger.Info("Module stopped",
zap.String("module", m.name),
)
return nil
}
