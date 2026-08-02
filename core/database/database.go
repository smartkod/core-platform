package database

import (
"fmt"
"time"

"core-platform/core/config"
"core-platform/core/logger"

"go.uber.org/zap"
"gorm.io/driver/postgres"
"gorm.io/gorm"
gormlogger "gorm.io/gorm/logger"
)

type DB struct {
*gorm.DB
logger *logger.Logger
config *config.DatabaseConfig
}

func New(cfg *config.DatabaseConfig, log *logger.Logger) (*DB, error) {
dsn := fmt.Sprintf(
"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Name, cfg.SSLMode,
)

db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
Logger: gormlogger.Default.LogMode(gormlogger.Info),
})
if err != nil {
return nil, fmt.Errorf("failed to connect to database: %w", err)
}

sqlDB, err := db.DB()
if err != nil {
return nil, fmt.Errorf("failed to get database instance: %w", err)
}

sqlDB.SetMaxIdleConns(10)
sqlDB.SetMaxOpenConns(100)
sqlDB.SetConnMaxLifetime(time.Hour)

log.Info("Database connected successfully",
zap.String("host", cfg.Host),
zap.String("port", cfg.Port),
zap.String("database", cfg.Name),
)

return &DB{
DB:     db,
logger: log,
config: cfg,
}, nil
}

func (d *DB) Health() error {
sqlDB, err := d.DB.DB()
if err != nil {
return fmt.Errorf("failed to get database instance: %w", err)
}
return sqlDB.Ping()
}

func (d *DB) Close() error {
sqlDB, err := d.DB.DB()
if err != nil {
return err
}
return sqlDB.Close()
}