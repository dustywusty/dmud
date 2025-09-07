package models

// Room represents a location in the game world.
type Room struct {
	ID          string `gorm:"primaryKey"`
	X           int    `gorm:"column:x"`
	Y           int    `gorm:"column:y"`
	Z           int    `gorm:"column:z"`
	Description string `gorm:"column:description"`
	Exits       []Exit `gorm:"foreignKey:SourceRoomID"`
}

// Exit represents a directional connection from one room to another.
type Exit struct {
	ID           uint   `gorm:"primaryKey"`
	SourceRoomID string `gorm:"column:source_room_id"`
	Direction    string `gorm:"column:direction"`
	RoomID       string `gorm:"column:room_id"`
}

// Item represents an object that can exist in the world or inventory.
type Item struct {
	ID          string `gorm:"primaryKey"`
	Name        string `gorm:"column:name"`
	Description string `gorm:"column:description"`
}

// Character represents a player or NPC in the world.
type Character struct {
	ID            string `gorm:"primaryKey"`
	Name          string `gorm:"column:name"`
	RoomID        string `gorm:"column:room_id"`
	HealthCurrent int    `gorm:"column:health_current"`
	HealthMax     int    `gorm:"column:health_max"`
}

// InventoryItem links characters to items they possess.
type InventoryItem struct {
	ID          uint   `gorm:"primaryKey"`
	CharacterID string `gorm:"column:character_id"`
	ItemID      string `gorm:"column:item_id"`
	Quantity    int    `gorm:"column:quantity"`
}
