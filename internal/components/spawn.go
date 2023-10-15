package components

type SpawnType int

const (
	SpawnTypeNPC SpawnType = iota
	SpawnTypePlayer
)

type SpawnTable interface {
	MaxCount() int
	MinCount() int
	SpawnType() SpawnType
}

type Spawn interface {
	GetSpawn(SpawnType) Spawn
	AddSpawn(SpawnType, Spawn) error
	RemoveSpawn(SpawnType) error
	ListSpawns() []SpawnTable
}
