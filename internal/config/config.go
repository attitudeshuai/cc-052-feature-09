package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	DBHost     string
	DBPort     int
	DBUser     string
	DBPassword string
	DBName     string

	RedisHost     string
	RedisPort     int
	RedisPassword string

	MinIOEndpoint       string
	MinIOPublicEndpoint string
	MinIOAccessKey      string
	MinIOSecretKey      string
	MinIOBucket         string
	MinIOUseSSL         bool

	ServerPort string

	// GeoRegionMap 是扫码地区的服务端网段映射，格式 "cidr=地区码;cidr=地区码"。
	// 扫码地区只按真实对端 IP 查该映射得出，未配置或匹配不到即为未知，
	// 绝不采信客户端自报的来源（X-Forwarded-For 等）。
	GeoRegionMap string
}

func Load() *Config {
	godotenv.Load()

	cfg := &Config{
		DBHost:              getEnv("DB_HOST", "localhost"),
		DBPort:              getEnvInt("DB_PORT", 5432),
		DBUser:              getEnv("DB_USER", "farm"),
		DBPassword:          getEnv("DB_PASSWORD", "farm_secret"),
		DBName:              getEnv("DB_NAME", "farm_trace"),
		RedisHost:           getEnv("REDIS_HOST", "localhost"),
		RedisPort:           getEnvInt("REDIS_PORT", 6379),
		RedisPassword:       getEnv("REDIS_PASSWORD", ""),
		MinIOEndpoint:       getEnv("MINIO_ENDPOINT", "localhost:9000"),
		MinIOPublicEndpoint: getEnv("MINIO_PUBLIC_ENDPOINT", "localhost:9000"),
		MinIOAccessKey:      getEnv("MINIO_ACCESS_KEY", "minioadmin"),
		MinIOSecretKey:      getEnv("MINIO_SECRET_KEY", "minioadmin"),
		MinIOBucket:         getEnv("MINIO_BUCKET", "farm-trace"),
		MinIOUseSSL:         getEnvBool("MINIO_USE_SSL", false),
		ServerPort:          getEnv("SERVER_PORT", "8080"),
		GeoRegionMap:        getEnv("GEO_REGION_MAP", ""),
	}
	return cfg
}

func (c *Config) DSN() string {
	return "host=" + c.DBHost + " port=" + strconv.Itoa(c.DBPort) +
		" user=" + c.DBUser + " password=" + c.DBPassword +
		" dbname=" + c.DBName + " sslmode=disable"
}

func (c *Config) RedisAddr() string {
	return c.RedisHost + ":" + strconv.Itoa(c.RedisPort)
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	if v := os.Getenv(key); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return fallback
}
