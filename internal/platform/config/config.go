package config

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	HTTPAddress     string
	GRPCAddress     string
	APIKey          string
	MaxBodyBytes    int64
	MaxAssetBytes   int64
	RequestTimeout  time.Duration
	WorkerCount     int
	LeaseDuration   time.Duration
	StorageDir      string
	WatermarkSecret string
}

func Default() Config {
	return Config{HTTPAddress: ":8090", GRPCAddress: ":9090", APIKey: "development-key", MaxBodyBytes: 4 << 20, MaxAssetBytes: 32 << 20, RequestTimeout: 15 * time.Second, WorkerCount: 2, LeaseDuration: 20 * time.Second, StorageDir: "./data", WatermarkSecret: "local-signing-key"}
}

func Load(path string) (Config, error) {
	c := Default()
	if path != "" {
		f, err := os.Open(path)
		if err != nil && !os.IsNotExist(err) {
			return c, fmt.Errorf("open config: %w", err)
		}
		if err == nil {
			defer f.Close()
			s := bufio.NewScanner(f)
			for s.Scan() {
				line := strings.TrimSpace(strings.SplitN(s.Text(), "#", 2)[0])
				if line == "" {
					continue
				}
				parts := strings.SplitN(line, ":", 2)
				if len(parts) != 2 {
					return c, fmt.Errorf("invalid config line %q", line)
				}
				if err := assign(&c, strings.TrimSpace(parts[0]), strings.Trim(strings.TrimSpace(parts[1]), "\"'")); err != nil {
					return c, err
				}
			}
			if err := s.Err(); err != nil {
				return c, fmt.Errorf("scan config: %w", err)
			}
		}
	}
	for k, v := range map[string]*string{"MWF_HTTP_ADDR": &c.HTTPAddress, "MWF_GRPC_ADDR": &c.GRPCAddress, "MWF_API_KEY": &c.APIKey, "MWF_STORAGE_DIR": &c.StorageDir, "MWF_WATERMARK_SECRET": &c.WatermarkSecret} {
		if x, ok := os.LookupEnv(k); ok {
			*v = x
		}
	}
	if v := os.Getenv("MWF_WORKERS"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 {
			return c, fmt.Errorf("MWF_WORKERS must be positive")
		}
		c.WorkerCount = n
	}
	if v := os.Getenv("MWF_MAX_ASSET_BYTES"); v != "" {
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil || n < 1 {
			return c, fmt.Errorf("MWF_MAX_ASSET_BYTES must be positive")
		}
		c.MaxAssetBytes = n
	}
	return c, c.Validate()
}

func assign(c *Config, key, value string) error {
	switch key {
	case "http_address":
		c.HTTPAddress = value
	case "grpc_address":
		c.GRPCAddress = value
	case "api_key":
		c.APIKey = value
	case "storage_dir":
		c.StorageDir = value
	case "watermark_secret":
		c.WatermarkSecret = value
	case "worker_count":
		n, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("worker_count: %w", err)
		}
		c.WorkerCount = n
	case "max_body_bytes", "max_asset_bytes":
		n, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return fmt.Errorf("%s: %w", key, err)
		}
		if key == "max_body_bytes" {
			c.MaxBodyBytes = n
		} else {
			c.MaxAssetBytes = n
		}
	case "request_timeout", "lease_duration":
		d, err := time.ParseDuration(value)
		if err != nil {
			return fmt.Errorf("%s: %w", key, err)
		}
		if key == "request_timeout" {
			c.RequestTimeout = d
		} else {
			c.LeaseDuration = d
		}
	default:
		return fmt.Errorf("unknown config key %q", key)
	}
	return nil
}

func (c Config) Validate() error {
	if c.HTTPAddress == "" || c.GRPCAddress == "" {
		return fmt.Errorf("listener addresses are required")
	}
	if c.WorkerCount < 1 || c.MaxBodyBytes < 1 || c.MaxAssetBytes < c.MaxBodyBytes {
		return fmt.Errorf("invalid worker or size limits")
	}
	if c.RequestTimeout <= 0 || c.LeaseDuration <= 0 {
		return fmt.Errorf("durations must be positive")
	}
	if len(c.WatermarkSecret) < 8 {
		return fmt.Errorf("watermark secret is too short")
	}
	return nil
}
