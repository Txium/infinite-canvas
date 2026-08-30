package service

import (
	"log"
	"strings"
	"sync"
	"time"

	"github.com/tigerowo/infinite-canvas/repository"
)

const canvasTaskMaxAge = 10 * time.Minute
const canvasTaskCleanupInterval = time.Minute

var canvasTaskCleanupOnce sync.Once

func StartCanvasTaskCleanupScheduler() {
	canvasTaskCleanupOnce.Do(func() {
		go func() {
			cleanupStaleCanvasTasks()
			ticker := time.NewTicker(canvasTaskCleanupInterval)
			defer ticker.Stop()
			for range ticker.C {
				cleanupStaleCanvasTasks()
			}
		}()
	})
}

func cleanupStaleCanvasTasks() {
	before := time.Now().Add(-canvasTaskMaxAge).UTC().Format(time.RFC3339Nano)
	images, err := repository.ListStaleCanvasImageTasks(before, 200)
	if err != nil {
		log.Printf("list stale canvas image tasks failed err=%v", err)
	}
	for _, task := range images {
		if staleCanvasTaskNeedsReconciliation(task.Status, task.StartedAt) {
			task.Status = "reconciling"
			task.Error = ""
			task.ErrorDetail = "任务已提交上游但长时间未同步结果，正在对账；请勿重复提交"
			task.UpdatedAt = now()
			if _, err := repository.UpdateCanvasImageTask(task); err != nil {
				log.Printf("reconcile stale canvas image task failed id=%s err=%v", task.ID, err)
			}
			continue
		}
		if err := ReleaseTaskFrozenCredits(task.UserID, task.Model, task.Endpoint, task.ID); err != nil {
			log.Printf("release stale canvas image task failed id=%s err=%v", task.ID, err)
			continue
		}
		task.Status = "failed"
		task.Progress = 0
		task.Error = "任务处理超时，已自动解冻"
		task.ErrorDetail = "任务超过 10 分钟仍未完成"
		task.CompletedAt = now()
		task.UpdatedAt = task.CompletedAt
		if _, err := repository.UpdateCanvasImageTask(task); err != nil {
			log.Printf("expire canvas image task failed id=%s err=%v", task.ID, err)
		}
	}
	audios, err := repository.ListStaleCanvasAudioTasks(before, 200)
	if err != nil {
		log.Printf("list stale canvas audio tasks failed err=%v", err)
	}
	for _, task := range audios {
		if staleCanvasTaskNeedsReconciliation(task.Status, task.StartedAt) {
			task.Status = "reconciling"
			task.Error = ""
			task.ErrorDetail = "任务已提交上游但长时间未同步结果，正在对账；请勿重复提交"
			task.UpdatedAt = now()
			if _, err := repository.UpdateCanvasAudioTask(task); err != nil {
				log.Printf("reconcile stale canvas audio task failed id=%s err=%v", task.ID, err)
			}
			continue
		}
		if err := ReleaseTaskFrozenCredits(task.UserID, task.Model, task.Endpoint, task.ID); err != nil {
			log.Printf("release stale canvas audio task failed id=%s err=%v", task.ID, err)
			continue
		}
		task.Status = "failed"
		task.Progress = 0
		task.Error = "任务处理超时，已自动解冻"
		task.ErrorDetail = "任务超过 10 分钟仍未完成"
		task.CompletedAt = now()
		task.UpdatedAt = task.CompletedAt
		if _, err := repository.UpdateCanvasAudioTask(task); err != nil {
			log.Printf("expire canvas audio task failed id=%s err=%v", task.ID, err)
		}
	}
}

func staleCanvasTaskNeedsReconciliation(status string, startedAt string) bool {
	status = strings.ToLower(strings.TrimSpace(status))
	return strings.TrimSpace(startedAt) != "" || status == "processing" || status == "running" || status == "in_progress"
}
