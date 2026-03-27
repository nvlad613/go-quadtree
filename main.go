package main

import (
	"fmt"

	"map-search/pkg/quadtree"
)

func main() {
	tree, err := quadtree.New[string](0, 0, 100, 100, 2)
	if err != nil {
		panic(err)
	}

	points := []struct {
		x, y float64
		name string
	}{
		{x: 10, y: 10, name: "A"},
		{x: 20, y: 20, name: "B"},
		{x: 12, y: 15, name: "C"},
		{x: 90, y: 90, name: "D"},
	}

	for _, p := range points {
		if err = tree.Insert(p.x, p.y, p.name); err != nil {
			panic(err)
		}
	}

	const (
		lang float64 = 11
		lat  float64 = 11
		n            = 3
	)
	nearest := tree.SearchNearestN(11, 11, 3)
	fmt.Printf("ближайшие %d точек к (%f,%f):\n", n, lang, lat)
	for i, it := range nearest {
		fmt.Printf("%d) %s at (%.1f, %.1f)\n", i+1, it.Payload, it.Pos.X, it.Pos.Y)
	}
}
