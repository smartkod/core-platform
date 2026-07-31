package module

import (
"core-platform/core/logger"

"go.uber.org/zap"
)

// BaseModule temel modül implementasyonu
type BaseModule struct {
name      string
state     ModuleState
version   string
logger    *logger.Logger
dependsOn []string
}

// NewBaseModule yeni base module oluşturur
func NewBaseModule(name string, version string, state ModuleState, logger *logger.Logger) *BaseModule {
return &BaseModule{
name:      name,
state:     state,
version:   version,
logger:    logger,
dependsOn: []string{},
}
}

// Name modül adını döndürür
func (m *BaseModule) Name() string {
return m.name
}

// State modül durumunu döndürür
func (m *BaseModule) State() ModuleState {
return m.state
}

// Version modül versiyonunu döndürür
func (m *BaseModule) Version() string {
return m.version
}

// DependsOn bağımlılıkları döndürür
func (m *BaseModule) DependsOn() []string {
return m.dependsOn
}

// SetDependsOn bağımlılıkları ayarlar
func (m *BaseModule) SetDependsOn(deps []string) *BaseModule {
m.dependsOn = deps
return m
}

// Initialize modülü başlatır
func (m *BaseModule) Initialize() error {
m.logger.Info("Module initialized",
zap.String("module", m.name),
zap.String("version", m.version),
)
return nil
}

// Start modülü başlatır
func (m *BaseModule) Start() error {
m.logger.Info("Module started",
zap.String("module", m.name),
)
return nil
}

// Stop modülü durdurur
func (m *BaseModule) Stop() error {
m.logger.Info("Module stopped",
zap.String("module", m.name),
)
return nil
}
