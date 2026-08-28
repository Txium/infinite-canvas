package service

import (
	"sort"
	"strings"

	"github.com/tigerowo/infinite-canvas/model"
	"github.com/tigerowo/infinite-canvas/repository"
)

func AdminGenerationTasks(limit int) ([]model.AdminGenerationTask, error) {
	videos, err := repository.ListRecentVideoTasks(limit)
	if err != nil { return nil, err }
	images, err := repository.ListRecentCanvasImageTasks(limit)
	if err != nil { return nil, err }
	audios, err := repository.ListRecentCanvasAudioTasks(limit)
	if err != nil { return nil, err }
	items := make([]model.AdminGenerationTask, 0, len(videos)+len(images)+len(audios))
	ids := make([]string, 0, cap(items))
	for _, task := range videos {
		items = append(items, model.AdminGenerationTask{ID:task.ID,UserID:task.UserID,UserDisplayName:task.UserDisplayName,Kind:"video",Model:task.Model,Status:task.Status,BillingStatus:task.BillingStatus,PriceCents:task.Credits,Source:task.Source,ResultURL:publicTaskResultURL(task.VideoURL),Error:firstVideoTaskValue(task.Error,task.ErrorDetail),CreatedAt:task.CreatedAt,CompletedAt:task.CompletedAt})
		ids = append(ids, task.ID)
	}
	for _, task := range images {
		items = append(items, model.AdminGenerationTask{ID:task.ID,UserID:task.UserID,UserDisplayName:task.UserDisplayName,Kind:"image",Model:task.Model,Status:task.Status,Source:task.Source,ResultURL:publicTaskResultURL(task.ImageURL),Error:firstVideoTaskValue(task.Error,task.ErrorDetail),CreatedAt:task.CreatedAt,CompletedAt:task.CompletedAt})
		ids = append(ids, task.ID)
	}
	for _, task := range audios {
		items = append(items, model.AdminGenerationTask{ID:task.ID,UserID:task.UserID,UserDisplayName:task.UserDisplayName,Kind:"audio",Model:task.Model,Status:task.Status,Source:task.Source,ResultURL:publicTaskResultURL(task.AudioURL),Error:firstVideoTaskValue(task.Error,task.ErrorDetail),CreatedAt:task.CreatedAt,CompletedAt:task.CompletedAt})
		ids = append(ids, task.ID)
	}
	logs, err := repository.ListTaskCreditLogs(ids)
	if err != nil { return nil, err }
	byTask := map[string][]model.CreditLog{}
	for _, item := range logs { byTask[item.RelatedID] = append(byTask[item.RelatedID], item) }
	for i := range items {
		for _, entry := range byTask[items[i].ID] {
			if entry.Type == model.CreditLogTypeAIFreeze { items[i].PriceCents = -entry.Amount; items[i].BillingStatus = "frozen" }
			if entry.Type == model.CreditLogTypeAISettle { items[i].BillingStatus = "settled" }
			if entry.Type == model.CreditLogTypeAIRelease { items[i].BillingStatus = "released" }
		}
		items[i].BillingStatus = strings.TrimSpace(items[i].BillingStatus)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt > items[j].CreatedAt })
	if limit > 0 && len(items) > limit { items = items[:limit] }
	return items, nil
}

func publicTaskResultURL(value string) string {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, "https://") || strings.HasPrefix(value, "http://") { return value }
	return ""
}
