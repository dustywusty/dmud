package main

// https://github.com/kelindar/ecs

import (
  "fmt"
	"github.com/EngoEngine/ecs"
)

func main() {
	world := ecs.World{}
	world.Update(0.1)

  fmt.Println("Hello, World!")
}