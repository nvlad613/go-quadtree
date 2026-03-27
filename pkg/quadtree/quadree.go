package quadtree

import (
	"errors"
	"sort"
)

const minRectSide = 1e-9

type Tree[T any] struct {
	root       *Node[T]
	maxPerNode int
}

func New[T any](rectMinX, rectMinY, rectMaxX, rectMaxY float64, maxPerNode int) (*Tree[T], error) {
	if maxPerNode <= 0 {
		return nil, errors.New("maxPerNode must be greater than zero")
	}

	if rectMaxX-rectMinX < minRectSide || rectMaxY-rectMinY < minRectSide {
		return nil, errors.New("quadtree bounds are too small")
	}

	var tree Tree[T]
	tree.maxPerNode = maxPerNode
	tree.root = newNode[T](vec2{rectMinX, rectMinY}, vec2{rectMaxX, rectMaxY}, &tree, nil)
	return &tree, nil
}

func (t *Tree[T]) Insert(lang, lat float64, data T) error {
	if !t.root.contains(vec2{lang, lat}) {
		return errors.New("point is out of bounds")
	}
	t.root.insert(item[T]{
		Payload: data,
		Pos:     vec2{lang, lat},
	})
	return nil
}

func (t *Tree[T]) SearchInRect(minX, minY, maxX, maxY float64) ([]item[T], error) {
	if maxX-minX < minRectSide || maxY-minY < minRectSide {
		return nil, errors.New("search bounds are too small")
	}

	queryMin := vec2{X: minX, Y: minY}
	queryMax := vec2{X: maxX, Y: maxY}
	result := make([]item[T], 0)
	t.root.collectInRect(queryMin, queryMax, &result)
	return result, nil
}

func (t *Tree[T]) SearchNearestN(lang, lat float64, n int) []item[T] {
	if n <= 0 {
		return []item[T]{}
	}

	target := vec2{X: lang, Y: lat}
	all := make([]item[T], 0)
	t.root.collectInRect(t.root.Min, t.root.Max, &all)

	sort.Slice(all, func(i, j int) bool {
		di := distanceSq(all[i].Pos, target)
		dj := distanceSq(all[j].Pos, target)
		if di != dj {
			return di < dj
		}

		if all[i].Pos.X != all[j].Pos.X {
			return all[i].Pos.X < all[j].Pos.X
		}

		return all[i].Pos.Y < all[j].Pos.Y
	})

	if n >= len(all) {
		return all
	}

	return all[:n]
}

func distanceSq(a, b vec2) float64 {
	dx := a.X - b.X
	dy := a.Y - b.Y
	return dx*dx + dy*dy
}
