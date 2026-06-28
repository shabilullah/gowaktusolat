package config

import (
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	DBPath      string
	Port        string
	BasePath    string
	CORSOrigins string
	Prefork     bool
	APIKey      string
	SeederSched string
	Years       []int
}

func Load() *Config {
	_ = godotenv.Load()
	return &Config{
		DBPath:      envOrDefault("DB_PATH", "data/waktusolat.db"),
		Port:        envOrDefault("PORT", "8080"),
		BasePath:    envOrDefault("BASE_PATH", ""),
		CORSOrigins: envOrDefault("CORS_ORIGINS", "*"),
		Prefork:     envBool("PREFORK", false),
		APIKey:      os.Getenv("API_KEY"),
		SeederSched: os.Getenv("SEEDER_SCHED"),
		Years:       parseYears(os.Getenv("YEAR")),
	}
}

// CORSOriginsSlice returns CORS origins as a string slice for gofiber v3.
func (c *Config) CORSOriginsSlice() []string {
	if c.CORSOrigins == "" || c.CORSOrigins == "*" {
		return []string{"*"}
	}
	origins := strings.Split(c.CORSOrigins, ",")
	for i, o := range origins {
		origins[i] = strings.TrimSpace(o)
	}
	return origins
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envBool(key string, def bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	return v == "true" || v == "1"
}

func parseYears(raw string) []int {
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	var years []int
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		y, err := strconv.Atoi(p)
		if err != nil {
			continue
		}
		years = append(years, y)
	}
	return years
}
