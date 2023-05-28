package components

type Exit struct {
	Direction string
	Room      *RoomComponent
}

type RoomComponent struct {
	Description string
	Exits       []Exit
}
