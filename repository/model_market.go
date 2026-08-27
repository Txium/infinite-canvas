package repository

import (
	"github.com/tigerowo/infinite-canvas/model"
	"gorm.io/gorm"
)

func ListMarketModels(category string, featured bool) ([]model.MarketModel, error) {
	db, err := DB()
	if err != nil { return nil, err }
	tx := db.Where("enabled = ?", true)
	if category != "" && category != "all" { tx = tx.Where("category = ?", category) }
	if featured { tx = tx.Where("featured = ?", true) }
	var items []model.MarketModel
	err = tx.Order("sort asc, name asc").Find(&items).Error
	return items, err
}

func ListModelPrices(modelIDs []string) ([]model.ModelPrice, error) {
	if len(modelIDs) == 0 { return []model.ModelPrice{}, nil }
	db, err := DB(); if err != nil { return nil, err }
	var items []model.ModelPrice
	err = db.Where("model_id IN ? AND enabled = ?", modelIDs, true).Find(&items).Error
	return items, err
}

func ListModelProviders() ([]model.ModelProvider, error) {
	db, err := DB(); if err != nil { return nil, err }
	var items []model.ModelProvider
	err = db.Order("priority asc, name asc").Find(&items).Error
	return items, err
}

func ListModelRoutes() ([]model.ModelRoute, error) {
	db, err := DB(); if err != nil { return nil, err }
	var items []model.ModelRoute
	err = db.Order("model_id asc, priority asc").Find(&items).Error
	return items, err
}

func ListEnabledModelRoutes(modelIDs []string) ([]model.ModelRoute, error) {
	if len(modelIDs) == 0 { return []model.ModelRoute{}, nil }
	db, err := DB(); if err != nil { return nil, err }
	var items []model.ModelRoute
	err = db.Where("model_id IN ? AND enabled = ?", modelIDs, true).Find(&items).Error
	return items, err
}

func SaveModelProvider(item model.ModelProvider) error { db, err := DB(); if err != nil { return err }; return db.Save(&item).Error }
func SaveMarketModel(item model.MarketModel) error { db, err := DB(); if err != nil { return err }; return db.Save(&item).Error }
func SaveModelRoute(item model.ModelRoute) error { db, err := DB(); if err != nil { return err }; return db.Save(&item).Error }
func SaveModelPrice(item model.ModelPrice) error { db, err := DB(); if err != nil { return err }; return db.Save(&item).Error }

func SeedMarketModels(items []model.MarketModel) error {
	db, err := DB(); if err != nil { return err }
	return db.Transaction(func(tx *gorm.DB) error {
		for _, item := range items {
			if err := tx.Where("id = ?", item.ID).FirstOrCreate(&item).Error; err != nil { return err }
		}
		return nil
	})
}

func EnabledRoutesForModel(modelID string) ([]model.ModelRoute, error) {
	db, err := DB(); if err != nil { return nil, err }
	var items []model.ModelRoute
	err = db.Where("model_id = ? AND enabled = ?", modelID, true).Order("priority asc").Find(&items).Error
	return items, err
}

func ModelProviderByID(id string) (model.ModelProvider, error) {
	db, err := DB(); if err != nil { return model.ModelProvider{}, err }
	var item model.ModelProvider
	err = db.Where("id = ? AND enabled = ?", id, true).First(&item).Error
	return item, err
}

func SavedModelProviderByID(id string) (model.ModelProvider, error) {
	db, err := DB(); if err != nil { return model.ModelProvider{}, err }
	var item model.ModelProvider
	err = db.Where("id = ?", id).First(&item).Error
	return item, err
}

func MarketModelPrice(modelID string) (model.ModelPrice, error) {
	db, err := DB(); if err != nil { return model.ModelPrice{}, err }
	var item model.ModelPrice
	err = db.Where("model_id = ? AND enabled = ?", modelID, true).Order("created_at asc").First(&item).Error
	return item, err
}
