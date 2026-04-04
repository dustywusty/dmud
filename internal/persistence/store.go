package persistence

import (
	"context"
	"time"

	"dmud/internal/components"
)

type Store interface {
	LoadPlayer(ctx context.Context, key string) (*PlayerState, error)
	SavePlayer(ctx context.Context, key string, state *PlayerState) error
	DeletePlayer(ctx context.Context, key string) error
	LoadWorld(ctx context.Context) (*WorldState, error)
	SaveWorld(ctx context.Context, state *WorldState) error
	IsAdmin(ctx context.Context, key string) (bool, error)
	SetAdmin(ctx context.Context, key string, isAdmin bool) error
}

type StoreConfig struct {
	Driver    string
	RedisURL  string
	KeyPrefix string
	TTL       time.Duration
}

type PlayerState struct {
	Version       int                          `json:"version"`
	Name          string                       `json:"name"`
	AreaID        string                       `json:"area_id"`
	Health        HealthState                  `json:"health"`
	Experience    ExperienceState              `json:"experience"`
	Inventory     InventoryState               `json:"inventory"`
	Quests        map[string]QuestStatusRecord `json:"quests"`
	StatusEffects []StatusEffectState          `json:"status_effects"`
	UpdatedAt     time.Time                    `json:"updated_at"`
}

type HealthState struct {
	Current int                     `json:"current"`
	Max     int                     `json:"max"`
	Status  components.HealthStatus `json:"status"`
}

type ExperienceState struct {
	Current int `json:"current"`
	Level   int `json:"level"`
}

type InventoryState struct {
	MaxSlots int         `json:"max_slots"`
	Items    []ItemState `json:"items"`
}

type ItemState struct {
	ID          string              `json:"id"`
	Name        string              `json:"name"`
	Description string              `json:"description"`
	Type        components.ItemType `json:"type"`
	Value       int                 `json:"value"`
	Stackable   bool                `json:"stackable"`
	Quantity    int                 `json:"quantity"`
}

type QuestStatusRecord struct {
	Status components.QuestStatus `json:"status"`
}

type StatusEffectState struct {
	Type                components.StatusEffectType `json:"type"`
	Name                string                      `json:"name"`
	AppliedAtUnix       int64                       `json:"applied_at_unix"`
	DurationSecond      int64                       `json:"duration_seconds"`
	HPBonus             int                         `json:"hp_bonus"`
	Applied             bool                        `json:"applied"`
	SourceEntityID      string                      `json:"source_entity_id,omitempty"`
	SuppressAggro       bool                        `json:"suppress_aggro,omitempty"`
	SuppressRetaliation bool                        `json:"suppress_retaliation,omitempty"`
}

type WorldState struct {
	Version   int                  `json:"version"`
	Areas     map[string]AreaState `json:"areas"`
	NPCs      []NPCState           `json:"npcs"`
	Corpses   []CorpseState        `json:"corpses"`
	DayCycle  DayCycleState        `json:"day_cycle"`
	UpdatedAt time.Time            `json:"updated_at"`
}

type AreaState struct {
	Items []ItemState `json:"items"`
}

type NPCState struct {
	TemplateID       string                 `json:"template_id"`
	AreaID           string                 `json:"area_id"`
	Name             string                 `json:"name"`
	Description      string                 `json:"description"`
	Behavior         components.NPCBehavior `json:"behavior"`
	Health           HealthState            `json:"health"`
	Inventory        InventoryState         `json:"inventory"`
	LastActionUnix   int64                  `json:"last_action_unix"`
	LastMovementUnix int64                  `json:"last_movement_unix"`
}

type CorpseState struct {
	VictimName   string         `json:"victim_name"`
	WasPlayer    bool           `json:"was_player"`
	AreaID       string         `json:"area_id"`
	TimeOfDeath  int64          `json:"time_of_death"`
	DecaySeconds int64          `json:"decay_seconds"`
	LootedAtUnix int64          `json:"looted_at_unix"`
	Inventory    InventoryState `json:"inventory"`
}

type DayCycleState struct {
	CurrentTime    components.TimeOfDay `json:"current_time"`
	ElapsedSeconds int64                `json:"elapsed_seconds"`
	CycleStartUnix int64                `json:"cycle_start_unix"`
	DayNumber      int                  `json:"day_number"`
}
