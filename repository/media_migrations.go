package repository

import (
	_ "embed"
	"gorm.io/gorm"
	"log"
)

//go:embed migrations/001_media_profiles.sql
var mediaProfilesSQL string

//go:embed migrations/002_generation_audit.sql
var generationAuditSQL string

// Apply only to the configured PostgreSQL connection; never open another database.
func applyMediaMigrations(db *gorm.DB) error {
	if db.Dialector.Name() != "postgres" {
		return nil
	}
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SELECT pg_advisory_xact_lock(814152001)").Error; err != nil {
			return err
		}
		if err := tx.Exec("CREATE TABLE IF NOT EXISTS canvas_schema_migrations (version text PRIMARY KEY, applied_at timestamptz NOT NULL DEFAULT now())").Error; err != nil {
			return err
		}
		var count int64
		if err := tx.Table("canvas_schema_migrations").Where("version = ?", "001_media_profiles").Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return nil
		}
		if err := tx.Exec(mediaProfilesSQL).Error; err != nil {
			return err
		}
		if err := tx.Exec("INSERT INTO canvas_schema_migrations(version) VALUES (?)", "001_media_profiles").Error; err != nil {
			return err
		}
		log.Print("MEDIA_SCHEMA_MIGRATION_APPLIED version=001_media_profiles driver=postgres")
		return nil
	})
}

func applyGenerationAuditMigration(db *gorm.DB) error {
	if db.Dialector.Name() != "postgres" {
		return nil
	}
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SELECT pg_advisory_xact_lock(814152002)").Error; err != nil {
			return err
		}
		var count int64
		if err := tx.Table("canvas_schema_migrations").Where("version = ?", "002_generation_audit").Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return nil
		}
		if err := tx.Exec(generationAuditSQL).Error; err != nil {
			return err
		}
		if err := tx.Exec("INSERT INTO canvas_schema_migrations(version) VALUES (?)", "002_generation_audit").Error; err != nil {
			return err
		}
		log.Print("GENERATION_SCHEMA_MIGRATION_APPLIED version=002_generation_audit driver=postgres")
		return nil
	})
}
