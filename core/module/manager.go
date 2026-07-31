package module

import (
"fmt"
"sync"

"core-platform/core/config"
"core-platform/core/logger"

"go.uber.org/zap"
)

// ModuleState modül durumları
type ModuleState string

const (
StateEnabled      ModuleState = "ENABLED"
StateDisabled     ModuleState = "DISABLED"
StateMaintenance  ModuleState = "MAINTENANCE"
StateExperimental ModuleState = "EXPERIMENTAL"
StateDeprecated   ModuleState = "DEPRECATED"
)

// Module modül arayüzü
type Module interface {
Name() string
State() ModuleState
Initialize() error
Start() error
Stop() error
DependsOn() []string
}

// ModuleInfo modül bilgileri
type ModuleInfo struct {
Name      string       `json:"name"`
State     ModuleState  `json:"state"`
Version   string       `json:"version"`
DependsOn []string     `json:"depends_on"`
Started   bool         `json:"started"`
}

// Manager modül yöneticisi
type Manager struct {
mu       sync.RWMutex
modules  map[string]Module
states   map[string]ModuleState
logger   *logger.Logger
config   *config.ModulesConfig
}

// NewManager yeni modül yöneticisi oluşturur
func NewManager(logger *logger.Logger, cfg *config.ModulesConfig) *Manager {
return &Manager{
modules: make(map[string]Module),
states:  make(map[string]ModuleState),
logger:  logger,
config:  cfg,
}
}

// Register modülü kaydeder
func (m *Manager) Register(module Module) error {
m.mu.Lock()
defer m.mu.Unlock()

name := module.Name()
if _, exists := m.modules[name]; exists {
return fmt.Errorf("module %s already registered", name)
}

// Modül durumunu kontrol et
state := module.State()
if state == "" {
state = StateDisabled
}

// Config'ten durumu override et
if customState, ok := m.config.States[name]; ok {
state = ModuleState(customState)
}

m.modules[name] = module
m.states[name] = state

m.logger.Info("Module registered",
zap.String("module", name),
zap.String("state", string(state)),
)

return nil
}

// Initialize tüm modülleri başlatır
func (m *Manager) Initialize() error {
m.mu.Lock()
defer m.mu.Unlock()

// Önce bağımlılıkları kontrol et
if err := m.validateDependencies(); err != nil {
return err
}

// Modülleri sıralı olarak başlat
for name, module := range m.modules {
if m.isModuleEnabled(name) {
m.logger.Info("Initializing module",
zap.String("module", name),
)
if err := module.Initialize(); err != nil {
return fmt.Errorf("failed to initialize module %s: %w", name, err)
}
}
}

return nil
}

// Start tüm modülleri başlatır
func (m *Manager) Start() error {
m.mu.Lock()
defer m.mu.Unlock()

for name, module := range m.modules {
if m.isModuleEnabled(name) {
m.logger.Info("Starting module",
zap.String("module", name),
)
if err := module.Start(); err != nil {
return fmt.Errorf("failed to start module %s: %w", name, err)
}
}
}

return nil
}

// Stop tüm modülleri durdurur
func (m *Manager) Stop() error {
m.mu.Lock()
defer m.mu.Unlock()

for name, module := range m.modules {
if m.isModuleEnabled(name) {
m.logger.Info("Stopping module",
zap.String("module", name),
)
if err := module.Stop(); err != nil {
m.logger.Error("Failed to stop module",
zap.String("module", name),
zap.Error(err),
)
}
}
}

return nil
}

// GetModule modülü döndürür
func (m *Manager) GetModule(name string) (Module, bool) {
m.mu.RLock()
defer m.mu.RUnlock()
module, exists := m.modules[name]
return module, exists
}

// GetModuleState modül durumunu döndürür
func (m *Manager) GetModuleState(name string) ModuleState {
m.mu.RLock()
defer m.mu.RUnlock()
if state, exists := m.states[name]; exists {
return state
}
return StateDisabled
}

// SetModuleState modül durumunu değiştirir
func (m *Manager) SetModuleState(name string, state ModuleState) error {
m.mu.Lock()
defer m.mu.Unlock()

if _, exists := m.modules[name]; !exists {
return fmt.Errorf("module %s not found", name)
}

m.states[name] = state
m.logger.Info("Module state changed",
zap.String("module", name),
zap.String("state", string(state)),
)

return nil
}

// GetModules tüm modülleri döndürür
func (m *Manager) GetModules() map[string]ModuleInfo {
m.mu.RLock()
defer m.mu.RUnlock()

result := make(map[string]ModuleInfo)
for name, module := range m.modules {
result[name] = ModuleInfo{
Name:      name,
State:     m.states[name],
DependsOn: module.DependsOn(),
Started:   m.isModuleEnabled(name),
}
}
return result
}

// validateDependencies bağımlılıkları doğrular
func (m *Manager) validateDependencies() error {
for name, module := range m.modules {
if !m.isModuleEnabled(name) {
continue
}
for _, dep := range module.DependsOn() {
if !m.isModuleEnabled(dep) {
return fmt.Errorf("module %s depends on %s which is not enabled", name, dep)
}
if _, exists := m.modules[dep]; !exists {
return fmt.Errorf("module %s depends on %s which does not exist", name, dep)
}
}
}
return nil
}

// isModuleEnabled modülün aktif olup olmadığını kontrol eder
func (m *Manager) isModuleEnabled(name string) bool {
state, exists := m.states[name]
if !exists {
return false
}
return state == StateEnabled || state == StateExperimental
}
