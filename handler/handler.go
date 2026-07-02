package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/hibiken/asynq"

	"image_sync/config"
	"image_sync/worker"
)

type SyncRequest struct {
	Src      string `json:"src" binding:"required"`
	Dst      string `json:"dst" binding:"required"`
	Image    string `json:"image" binding:"required"`
	Tag      string `json:"tag" binding:"required"`
	Priority string `json:"priority"`
}

type SyncResponse struct {
	TaskID string `json:"task_id"`
}

type TaskStatusResponse struct {
	TaskID      string `json:"task_id"`
	Status      string `json:"status"`
	CompletedAt string `json:"completed_at,omitempty"`
	Error       string `json:"error,omitempty"`
}

func RegisterRoutes(r *gin.Engine, client *asynq.Client, cfg config.Config) {
	r.POST("/api/v1/sync", handleSync(client, cfg))
	r.GET("/api/v1/tasks/:id", handleTaskStatus(cfg))
}

func handleSync(client *asynq.Client, cfg config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req SyncRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if _, ok := cfg.Registries[req.Src]; !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("source registry %q not configured", req.Src)})
			return
		}
		if _, ok := cfg.Registries[req.Dst]; !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("destination registry %q not configured", req.Dst)})
			return
		}

		payload, _ := json.Marshal(worker.ImageSyncPayload{
			Src:   req.Src,
			Dst:   req.Dst,
			Image: req.Image,
			Tag:   req.Tag,
		})

		queue := worker.QueueName(req.Priority)
		task := asynq.NewTask(worker.TaskTypeImageSync, payload)
		info, err := client.Enqueue(task, asynq.Queue(queue), asynq.MaxRetry(3))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("enqueue failed: %v", err)})
			return
		}

		c.JSON(http.StatusAccepted, SyncResponse{TaskID: info.ID})
	}
}

func handleTaskStatus(cfg config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		inspector := asynq.NewInspector(
			asynq.RedisClientOpt{Addr: cfg.Redis.Addr, Password: cfg.Redis.Password},
		)

		var info *asynq.TaskInfo
		var err error
		for _, q := range []string{"critical", "default", "low"} {
			info, err = inspector.GetTaskInfo(q, id)
			if err == nil {
				break
			}
		}
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
			return
		}

		resp := TaskStatusResponse{TaskID: info.ID}
		switch info.State {
		case asynq.TaskStateActive:
			resp.Status = "processing"
		case asynq.TaskStatePending:
			resp.Status = "pending"
		case asynq.TaskStateCompleted:
			resp.Status = "completed"
			resp.CompletedAt = info.CompletedAt.Format("2006-01-02T15:04:05Z")
		case asynq.TaskStateRetry, asynq.TaskStateArchived:
			resp.Status = "failed"
			resp.Error = info.LastErr
		default:
			resp.Status = fmt.Sprintf("unknown(%d)", int(info.State))
		}

		c.JSON(http.StatusOK, resp)
	}
}
