package handler

import (
	"github.com/google/uuid"
	"github.com/tigerowo/infinite-canvas/model"
)

func canRetryFailedVideoTask(task model.VideoTask) bool {
	return task.Status == "failed" && (task.BillingStatus == "released" || (task.Credits == 0 && task.BillingStatus == ""))
}

func videoRetryTaskID(userID, previousID string) string {
	return "client_video_retry_" + uuid.NewSHA1(uuid.NameSpaceOID, []byte(userID+":"+previousID)).String()
}
