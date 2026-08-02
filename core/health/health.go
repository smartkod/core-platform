package health

import (
"core-platform/core/database"
"core-platform/core/logger"
"core-platform/core/version"

"go.uber.org/zap"
)

type HealthChecker struct {
logger *logger.Logger
db     *database.DB
}

func New(logger *logger.Logger, db *database.DB) *HealthChecker {
return &HealthChecker{
logger: logger,
db:     db,
}
}

type HealthResponse struct {
Status   string            `json:"status"`
Service  string            `json:"service"`
Version  string            `json:"version"`
Database string            `json:"database,omitempty"`
Details  map[string]string `json:"details,omitempty"`
}

func (h *HealthChecker) Check() HealthResponse {
response := HealthResponse{
Status:  "ok",
Service: "Core Platform",
Version: version.GetVersion(),
Details: make(map[string]string),
}

// Check database
if h.db != nil {
if err := h.db.Health(); err != nil {
response.Status = "degraded"
response.Database = "unhealthy"
response.Details["database_error"] = err.Error()
h.logger.Error("Database health check failed",
zap.String("error", err.Error()),
)
} else {
response.Database = "healthy"
response.Details["database"] = "connected"
}
}

return response
}
