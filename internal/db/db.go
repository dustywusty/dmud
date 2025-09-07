package db

import (
	"dmud/internal/db/models"
	"encoding/json"
	"os"

	"gorm.io/gorm"
)

// AutoMigrate migrates the database schema for all models.
func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&models.Room{},
		&models.Exit{},
		&models.Item{},
		&models.Character{},
		&models.InventoryItem{},
	)
}

// SeedRooms loads initial room data from a JSON file if no rooms exist.
func SeedRooms(db *gorm.DB, jsonPath string) error {
	var count int64
	if err := db.Model(&models.Room{}).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	f, err := os.Open(jsonPath)
	if err != nil {
		return err
	}
	defer f.Close()

	var rooms []struct {
		ID          string            `json:"id"`
		Exits       map[string]string `json:"exits"`
		Description string            `json:"description"`
	}
	if err := json.NewDecoder(f).Decode(&rooms); err != nil {
		return err
	}

	for _, r := range rooms {
		room := models.Room{
			ID:          r.ID,
			Description: r.Description,
		}
		if err := db.Create(&room).Error; err != nil {
			return err
		}

		for dir, dest := range r.Exits {
			exit := models.Exit{
				SourceRoomID: r.ID,
				Direction:    dir,
				RoomID:       dest,
			}
			if err := db.Create(&exit).Error; err != nil {
				return err
			}
		}
	}
	return nil
}
