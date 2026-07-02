package worker

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hibiken/asynq"

	"image_sync/config"
	"image_sync/syncer"
)

const TaskTypeImageSync = "image:sync"

type ImageSyncPayload struct {
	Src   string `json:"src"`
	Dst   string `json:"dst"`
	Image string `json:"image"`
	Tag   string `json:"tag"`
}

func QueueName(priority string) string {
	switch priority {
	case "high":
		return "critical"
	case "low":
		return "low"
	default:
		return "default"
	}
}

func NewServer(cfg config.Config) *asynq.Server {
	return asynq.NewServer(
		asynq.RedisClientOpt{Addr: cfg.Redis.Addr, Password: cfg.Redis.Password},
		asynq.Config{Concurrency: cfg.Asynq.Concurrency},
	)
}

func NewClient(cfg config.Config) *asynq.Client {
	return asynq.NewClient(
		asynq.RedisClientOpt{Addr: cfg.Redis.Addr, Password: cfg.Redis.Password},
	)
}

func NewHandler(cfg config.Config) asynq.Handler {
	mux := asynq.NewServeMux()
	mux.HandleFunc(TaskTypeImageSync, makeHandleImageSync(cfg))
	return mux
}

func makeHandleImageSync(cfg config.Config) func(ctx context.Context, t *asynq.Task) error {
	return func(ctx context.Context, t *asynq.Task) error {
		var p ImageSyncPayload
		if err := json.Unmarshal(t.Payload(), &p); err != nil {
			return fmt.Errorf("unmarshal payload: %w", err)
		}

		srcReg, ok := cfg.Registries[p.Src]
		if !ok {
			return fmt.Errorf("source registry %q not configured", p.Src)
		}
		dstReg, ok := cfg.Registries[p.Dst]
		if !ok {
			return fmt.Errorf("destination registry %q not configured", p.Dst)
		}

		srcAuth := srcReg.BasicAuth()
		dstAuth := dstReg.BasicAuth()

		srcImage := fmt.Sprintf("%s/%s:%s", srcReg.URL, p.Image, p.Tag)
		dstImage := fmt.Sprintf("%s/%s:%s", dstReg.URL, p.Image, p.Tag)

		return syncer.Copy(srcImage, dstImage, srcAuth, dstAuth)
	}
}
