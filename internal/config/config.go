package config

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ctrlaltcloud008-hub/prj-apex-core-modules/pkg/config"
	"github.com/spf13/viper"
)

type UploadConfig struct {
	appEnv        string
	port          string
	service       string
	region        string
	projectID     string
	spannerDB     string
	buckets       map[string]string
	redisAddr     string
	redisPassword string
}

func LoadUploadConfig() (*UploadConfig, error) {

	v := viper.New()
	v.SetDefault("PORT", "8080")
	v.SetDefault("APP_ENV", "local")
	v.SetDefault("SERVICE", "upload")
	v.SetDefault("REGION", "asia-south1")
	v.SetDefault("PROJECT_ID", "apex-494315")
	v.SetDefault("BUCKETS", map[string]string{"asia-south1": "prj-apex-upload-bucket"})
	v.SetDefault("BUCKETS_JSON", "")
	v.SetDefault("SPANNER_DATABASE", "projects/test-project/instances/test-instance/databases/test-database")
	v.SetDefault("REDIS_ADDR", "localhost:6379")
	v.SetDefault("REDIS_PASSWORD", "")

	if err := config.LoadConfig(v, "config"); err != nil {
		return nil, err
	}

	buckets := v.GetStringMapString("BUCKETS")
	if bucketsJSON := strings.TrimSpace(v.GetString("BUCKETS_JSON")); bucketsJSON != "" {
		if err := json.Unmarshal([]byte(bucketsJSON), &buckets); err != nil {
			return nil, fmt.Errorf("parse BUCKETS_JSON: %w", err)
		}
	}

	cfg := &UploadConfig{
		appEnv:        v.GetString("APP_ENV"),
		port:          normalizePort(v.GetString("PORT")),
		service:       v.GetString("SERVICE"),
		region:        v.GetString("REGION"),
		projectID:     v.GetString("PROJECT_ID"),
		buckets:       buckets,
		spannerDB:     v.GetString("SPANNER_DATABASE"),
		redisAddr:     v.GetString("REDIS_ADDR"),
		redisPassword: v.GetString("REDIS_PASSWORD"),
	}

	return cfg, nil
}

func normalizePort(port string) string {
	port = strings.TrimSpace(port)
	if port == "" {
		return ":8000"
	}
	if strings.HasPrefix(port, ":") {
		return port
	}
	return ":" + port
}

func (c *UploadConfig) AppEnv() string                { return c.appEnv }
func (c *UploadConfig) Port() string                  { return c.port }
func (c *UploadConfig) Service() string               { return c.service }
func (c *UploadConfig) Region() string                { return c.region }
func (c *UploadConfig) ProjectID() string             { return c.projectID }
func (c *UploadConfig) Buckets() map[string]string    { return c.buckets }
func (c *UploadConfig) SpannerDatabase() string       { return c.spannerDB }
func (c *UploadConfig) RedisAddr() string             { return c.redisAddr }
func (c *UploadConfig) RedisPassword() string         { return c.redisPassword }
