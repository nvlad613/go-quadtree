package quadtree

type Node[T any] struct {
	Min vec2
	Max vec2

	items   []item[T]
	root    *Tree[T]
	notLeaf bool

	ul, ur *Node[T]
	bl, br *Node[T]
}

type item[T any] struct {
	Pos     vec2
	Payload T
}

type vec2 struct {
	X, Y float64
}

func (n *Node[T]) Width() float64 {
	return n.Max.X - n.Min.X
}

func (n *Node[T]) Height() float64 {
	return n.Max.Y - n.Min.Y
}

func newNode[T any](min, max vec2, root *Tree[T], points []item[T]) *Node[T] {
	n := &Node[T]{
		Min:   min,
		Max:   max,
		items: make([]item[T], 0, len(points)/2),
		root:  root,
	}

	for _, point := range points {
		if !n.contains(point.Pos) {
			continue
		}
		n.items = append(n.items, point)
	}

	return n
}

func (n *Node[T]) hasCap() bool {
	return !n.notLeaf && n.root.maxPerNode > len(n.items)
}

func (n *Node[T]) contains(pos vec2) bool {
	return inRect(pos, n.Min, n.Max)
}

func inRect(pos, min, max vec2) bool {
	return (pos.X >= min.X && pos.Y >= min.Y) && (pos.X < max.X && pos.Y < max.Y)
}

func (n *Node[T]) intersects(min, max vec2) bool {
	return n.Min.X < max.X && n.Max.X > min.X && n.Min.Y < max.Y && n.Max.Y > min.Y
}

func (n *Node[T]) collectInRect(min, max vec2, out *[]item[T]) {
	if n == nil || !n.intersects(min, max) {
		return
	}

	for _, obj := range n.items {
		if inRect(obj.Pos, min, max) {
			*out = append(*out, obj)
		}
	}

	if !n.notLeaf {
		return
	}

	n.ul.collectInRect(min, max, out)
	n.ur.collectInRect(min, max, out)
	n.bl.collectInRect(min, max, out)
	n.br.collectInRect(min, max, out)
}

func (n *Node[T]) insert(p item[T]) {
	if n.contains(p.Pos) && n.hasCap() {
		n.items = append(n.items, p)
		return
	}

	if !n.notLeaf {
		n.split4()
	}

	switch {
	case n.ul.contains(p.Pos):
		n.ul.insert(p)
	case n.ur.contains(p.Pos):
		n.ur.insert(p)
	case n.bl.contains(p.Pos):
		n.bl.insert(p)
	case n.br.contains(p.Pos):
		n.br.insert(p)
	default:
		panic("invalid quadtree")
	}
}

func (n *Node[T]) split4() {
	center := vec2{
		n.Min.X + n.Width()/2,
		n.Min.Y + n.Height()/2,
	}

	// Верхний-левый квадрант
	n.ul = newNode(
		n.Min,
		center,
		n.root,
		n.items,
	)
	// Верхний-правый квадрант
	n.ur = newNode(
		vec2{center.X, n.Min.Y},
		vec2{n.Max.X, center.Y},
		n.root,
		n.items,
	)
	// Нижний-левый квадрант
	n.bl = newNode(
		vec2{n.Min.X, center.Y},
		vec2{center.X, n.Max.Y},
		n.root,
		n.items,
	)
	// Нижний-правый квадрант
	n.br = newNode(
		center,
		n.Max,
		n.root,
		n.items,
	)

	n.items = nil
	n.notLeaf = true
}
