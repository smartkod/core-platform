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

// DB veritabanı bağlantısı
type DB struct {
*gorm.DB
log    *logger.Logger
config *config.DatabaseConfig
}

// New veritabanı bağlantısı oluşturur
func New(cfg *config.DatabaseConfig, log *logger.Logger) (*DB, error) {
dsn := fmt.Sprintf(
"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
cfg.Host,
cfg.Port,
cfg.User,
cfg.Password,
cfg.Name,
cfg.SSLMode,
)

// GORM logger konfigürasyonu - basit bir logger kullan
gormLog := gormlogger.New(
&gormLoggerWriter{log: log},
gormlogger.Config{
SlowThreshold:             time.Second,
LogLevel:                  gormlogger.Info,
IgnoreRecordNotFoundError: true,
Colorful:                  false,
},
)

db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
Logger: gormLog,
})
if err != nil {
return nil, fmt.Errorf("failed to connect to database: %w", err)
}

// Connection pool ayarları
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
log:    log,
config: cfg,
}, nil
}

// Health veritabanı sağlık kontrolü
func (d *DB) Health() error {
sqlDB, err := d.DB.DB()
if err != nil {
return fmt.Errorf("failed to get database instance: %w", err)
}

if err := sqlDB.Ping(); err != nil {
return fmt.Errorf("database ping failed: %w", err)
}

return nil
}

// Close veritabanı bağlantısını kapatır
func (d *DB) Close() error {
sqlDB, err := d.DB.DB()
if err != nil {
return err
}
return sqlDB.Close()
}

// gormLoggerWriter GORM için logger writer
type gormLoggerWriter struct {
log *logger.Logger
}

// Printf GORM log'larını yazdırır
func (w *gormLoggerWriter) Printf(format string, args ...interface{}) {
w.log.Info(fmt.Sprintf(format, args...))
}
