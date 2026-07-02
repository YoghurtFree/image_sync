package config

import (
	"os"
	"testing"
)

func TestLoad(t *testing.T) {
	yaml := `
server:
  port: 9090
redis:
  addr: localhost:6379
  password: testpass
asynq:
  concurrency: 5
registries:
  harbor-prod:
    url: registry.example.com
    username: admin
    password: secret123
  acr-cn:
    url: xxx.azurecr.io
    username: user1
    password: pass1
`
	tmp, err := os.CreateTemp("", "config-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmp.Name())
	tmp.Write([]byte(yaml))
	tmp.Close()

	cfg, err := Load(tmp.Name())
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.Server.Port != 9090 {
		t.Errorf("Server.Port = %d, want 9090", cfg.Server.Port)
	}
	if cfg.Redis.Addr != "localhost:6379" {
		t.Errorf("Redis.Addr = %s, want localhost:6379", cfg.Redis.Addr)
	}
	if cfg.Redis.Password != "testpass" {
		t.Errorf("Redis.Password = %s, want testpass", cfg.Redis.Password)
	}
	if cfg.Asynq.Concurrency != 5 {
		t.Errorf("Asynq.Concurrency = %d, want 5", cfg.Asynq.Concurrency)
	}
	if len(cfg.Registries) != 2 {
		t.Fatalf("len(Registries) = %d, want 2", len(cfg.Registries))
	}
	harbor := cfg.Registries["harbor-prod"]
	if harbor.URL != "registry.example.com" {
		t.Errorf("harbor URL = %s, want registry.example.com", harbor.URL)
	}
	if harbor.Username != "admin" {
		t.Errorf("harbor Username = %s, want admin", harbor.Username)
	}
	if harbor.Password != "secret123" {
		t.Errorf("harbor Password = %s, want secret123", harbor.Password)
	}
}

func TestLoadFileNotFound(t *testing.T) {
	_, err := Load("/nonexistent/config.yaml")
	if err == nil {
		t.Error("Load should fail for nonexistent file")
	}
}
