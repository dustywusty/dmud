package ecs

type System interface {
	SetWorld(w *World)
	Update(dt float64, w *World)
}
