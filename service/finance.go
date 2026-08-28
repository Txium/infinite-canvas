package service

import (
	"time"

	"github.com/tigerowo/infinite-canvas/model"
	"github.com/tigerowo/infinite-canvas/repository"
)

func AdminFinanceSummary() (model.AdminFinanceSummary, error) {
	current := time.Now()
	start := time.Date(current.Year(), current.Month(), current.Day(), 0, 0, 0, 0, current.Location()).Format(time.RFC3339)
	return repository.AdminFinanceSummary(start)
}
