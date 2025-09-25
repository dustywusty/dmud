package storage

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

type EntityModel struct {
	ID        string `gorm:"primaryKey"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

type AreaModel struct {
	EntityID    string `gorm:"primaryKey"`
	Region      string
	Description string
	X           int
	Y           int
	Z           int
}

type AreaExitModel struct {
	ID           uint   `gorm:"primaryKey"`
	AreaID       string `gorm:"index"`
	Direction    string
	TargetAreaID string
}

type PlayerModel struct {
	EntityID         string `gorm:"primaryKey"`
	Name             string `gorm:"uniqueIndex"`
	AreaID           string
	CommandHistory   JSONStringSlice `gorm:"type:jsonb"`
	LastKnownAddress string
}

type HealthModel struct {
	EntityID string `gorm:"primaryKey"`
	Current  int
	Max      int
	Status   int
}

type MovementModel struct {
	EntityID  string `gorm:"primaryKey"`
	Status    int
	Direction string
}

type CombatModel struct {
	EntityID  string `gorm:"primaryKey"`
	TargetID  string
	MinDamage int
	MaxDamage int
}

type NPCModel struct {
	EntityID     string `gorm:"primaryKey"`
	Name         string
	Description  string
	AreaID       string
	TemplateID   string
	Behavior     int
	Dialogue     JSONStringSlice `gorm:"type:jsonb"`
	LastAction   time.Time
	LastMovement time.Time
	TargetID     string
}

type SpawnModel struct {
	EntityID     string `gorm:"primaryKey"`
	AreaID       string
	Configs      JSONSpawnConfigList `gorm:"type:jsonb"`
	ActiveSpawns JSONStringMap       `gorm:"type:jsonb"`
	LastSpawn    time.Time
}

type JSONStringSlice []string

type JSONStringMap map[string]string

type SpawnConfigRecord struct {
	Type           int     `json:"type"`
	TemplateID     string  `json:"template_id"`
	MinCount       int     `json:"min_count"`
	MaxCount       int     `json:"max_count"`
	RespawnSeconds int64   `json:"respawn_seconds"`
	Chance         float64 `json:"chance"`
}

type JSONSpawnConfigList []SpawnConfigRecord

func (s JSONStringSlice) Value() (driver.Value, error) {
	if len(s) == 0 {
		return "[]", nil
	}
	b, err := json.Marshal(s)
	if err != nil {
		return nil, err
	}
	return string(b), nil
}

func (s *JSONStringSlice) Scan(value interface{}) error {
	if value == nil {
		*s = []string{}
		return nil
	}
	var bytes []byte
	switch v := value.(type) {
	case string:
		bytes = []byte(v)
	case []byte:
		bytes = v
	default:
		return fmt.Errorf("unsupported type for JSONStringSlice: %T", value)
	}
	if len(bytes) == 0 {
		*s = []string{}
		return nil
	}
	return json.Unmarshal(bytes, s)
}

func (m JSONStringMap) Value() (driver.Value, error) {
	if len(m) == 0 {
		return "{}", nil
	}
	b, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	return string(b), nil
}

func (m *JSONStringMap) Scan(value interface{}) error {
	if value == nil {
		*m = make(map[string]string)
		return nil
	}
	var bytes []byte
	switch v := value.(type) {
	case string:
		bytes = []byte(v)
	case []byte:
		bytes = v
	default:
		return fmt.Errorf("unsupported type for JSONStringMap: %T", value)
	}
	if len(bytes) == 0 {
		*m = make(map[string]string)
		return nil
	}
	var temp map[string]string
	if err := json.Unmarshal(bytes, &temp); err != nil {
		return err
	}
	*m = temp
	return nil
}

func (s JSONSpawnConfigList) Value() (driver.Value, error) {
	if len(s) == 0 {
		return "[]", nil
	}
	b, err := json.Marshal(s)
	if err != nil {
		return nil, err
	}
	return string(b), nil
}

func (s *JSONSpawnConfigList) Scan(value interface{}) error {
	if value == nil {
		*s = JSONSpawnConfigList{}
		return nil
	}
	var bytes []byte
	switch v := value.(type) {
	case string:
		bytes = []byte(v)
	case []byte:
		bytes = v
	default:
		return fmt.Errorf("unsupported type for JSONSpawnConfigList: %T", value)
	}
	if len(bytes) == 0 {
		*s = JSONSpawnConfigList{}
		return nil
	}
	return json.Unmarshal(bytes, s)
}
