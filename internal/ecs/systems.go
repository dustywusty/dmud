package ecs

type System interface {
	Update(entities []Entity, deltaTime float64)
}
