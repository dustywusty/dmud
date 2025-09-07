package ecs

import (
	"dmud/internal/common"
	"fmt"
)

func GetTypedComponent[T any](w *World, entityID common.EntityID, componentType string) (T, error) {
    var zero T

    component, err := w.GetComponent(entityID, componentType)
    if err != nil {
        return zero, err
    }

    typed, ok := component.(T)
    if !ok {
        return zero, fmt.Errorf("component %s is not of expected type", componentType)
    }

    return typed, nil
}
