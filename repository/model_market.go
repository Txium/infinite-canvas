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

func ListAllMarketModels() ([]model.MarketModel, error) {
	db, err := DB(); if err != nil { return nil, err }
	var items []model.MarketModel
	err = db.Order("sort asc, name asc").Find(&items).Error
	return items, err
}

func ListModelVariants(modelIDs []string, enabledOnly bool) ([]model.ModelVariant, error) {
	if len(modelIDs) == 0 { return []model.ModelVariant{}, nil }
	db, err := DB(); if err != nil { return nil, err }
	var items []model.ModelVariant
	tx := db.Where("model_id IN ?", modelIDs)
	if enabledOnly { tx = tx.Where("enabled = ?", true) }
	err = tx.Order("model_id asc, sort asc, name asc").Find(&items).Error
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
	err = db.Order("model_id asc, variant_id asc, priority asc").Find(&items).Error
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
func SaveModelVariant(item model.ModelVariant) error { db, err := DB(); if err != nil { return err }; return db.Save(&item).Error }

func SyncDefaultModelCatalog(version int, models []model.MarketModel, variants []model.ModelVariant, providers []model.ModelProvider, routes []model.ModelRoute, updatedAt string) error {
	db, err := DB()
	if err != nil { return err }
	return db.Transaction(func(tx *gorm.DB) error {
		var current model.ModelCatalogVersion
		if err := tx.Where("key = ?", "default").First(&current).Error; err == nil && current.Version >= version { return nil }
		modelIDs := make([]string, 0, len(models)); for _, item := range models { modelIDs = append(modelIDs, item.ID) }
		if err := tx.Model(&model.MarketModel{}).Where("id NOT IN ?", modelIDs).Update("enabled", false).Error; err != nil { return err }
		for _, item := range models { if err := tx.Save(&item).Error; err != nil { return err } }
		for _, item := range variants { if err := tx.Save(&item).Error; err != nil { return err } }
		for _, item := range providers { if err := tx.Where("id = ?", item.ID).FirstOrCreate(&item).Error; err != nil { return err } }
		if err := tx.Model(&model.ModelProvider{}).Where("code IN ?", []string{"302", "wavespeed", "lec", "seedance_nz"}).Update("api_key", "").Error; err != nil { return err }
		for _, item := range routes { if err := tx.Where("id = ?", item.ID).FirstOrCreate(&item).Error; err != nil { return err } }
		return tx.Save(&model.ModelCatalogVersion{Key:"default",Version:version,UpdatedAt:updatedAt}).Error
	})
}

func EnabledRoutesForVariant(variantID string) ([]model.ModelRoute, error) {
	db, err := DB(); if err != nil { return nil, err }
	var items []model.ModelRoute
	err = db.Where("variant_id = ? AND enabled = ?", variantID, true).Order("priority asc").Find(&items).Error
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

func SavedModelVariantByID(id string) (model.ModelVariant, error) {
	db, err := DB(); if err != nil { return model.ModelVariant{}, err }
	var item model.ModelVariant
	err = db.Where("id = ?", id).First(&item).Error
	return item, err
}

func SavedMarketModelByID(id string) (model.MarketModel, error) {
	db, err := DB(); if err != nil { return model.MarketModel{}, err }
	var item model.MarketModel
	err = db.Where("id = ?", id).First(&item).Error
	return item, err
}

func MarketVariantByID(variantID string) (model.ModelVariant, error) {
	db, err := DB(); if err != nil { return model.ModelVariant{}, err }
	var item model.ModelVariant
	err = db.Where("id = ?", variantID).First(&item).Error
	return item, err
}
