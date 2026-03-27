package quadtree

import "testing"

func TestSearchInRectReturnsOnlyPointsInside(t *testing.T) {
	tree, err := New[string](0, 0, 10, 10, 1)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	points := []struct {
		x float64
		y float64
		v string
	}{
		{x: 1, y: 1, v: "a"},
		{x: 5, y: 5, v: "b"},
		{x: 7, y: 2, v: "c"},
		{x: 9.5, y: 9.5, v: "d"},
	}

	for _, p := range points {
		if err := tree.Insert(p.x, p.y, p.v); err != nil {
			t.Fatalf("Insert(%v,%v) returned error: %v", p.x, p.y, err)
		}
	}

	got, err := tree.SearchInRect(0, 0, 6, 6)
	if err != nil {
		t.Fatalf("SearchInRect returned error: %v", err)
	}

	if len(got) != 2 {
		t.Fatalf("expected 2 results, got %d", len(got))
	}

	seen := map[string]bool{}
	for _, it := range got {
		seen[it.Payload] = true
	}

	if !seen["a"] || !seen["b"] {
		t.Fatalf("expected results to contain a and b, got %#v", got)
	}

	for _, it := range got {
		if it.Payload == "a" && (it.Pos.X != 1 || it.Pos.Y != 1) {
			t.Fatalf("unexpected coordinates for a: %+v", it.Pos)
		}
		if it.Payload == "b" && (it.Pos.X != 5 || it.Pos.Y != 5) {
			t.Fatalf("unexpected coordinates for b: %+v", it.Pos)
		}
	}
}

func TestSearchInRectOutsideTreeReturnsEmpty(t *testing.T) {
	tree, err := New[int](0, 0, 10, 10, 2)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	if err := tree.Insert(2, 2, 42); err != nil {
		t.Fatalf("Insert returned error: %v", err)
	}

	got, err := tree.SearchInRect(20, 20, 30, 30)
	if err != nil {
		t.Fatalf("SearchInRect returned error: %v", err)
	}

	if len(got) != 0 {
		t.Fatalf("expected empty result, got %d entries", len(got))
	}
}

func TestSearchInRectInvalidBounds(t *testing.T) {
	tree, err := New[int](0, 0, 10, 10, 2)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	if _, err := tree.SearchInRect(1, 1, 1, 5); err == nil {
		t.Fatal("expected error for zero-width rectangle")
	}

	if _, err := tree.SearchInRect(1, 1, 5, 1); err == nil {
		t.Fatal("expected error for zero-height rectangle")
	}
}

func TestSearchNearestNReturnsClosestInOrder(t *testing.T) {
	tree, err := New[string](0, 0, 10, 10, 2)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	data := []struct {
		x float64
		y float64
		v string
	}{
		{x: 1, y: 1, v: "a"},
		{x: 3, y: 1, v: "b"},
		{x: 5, y: 5, v: "c"},
		{x: 2, y: 2, v: "d"},
	}

	for _, p := range data {
		if err := tree.Insert(p.x, p.y, p.v); err != nil {
			t.Fatalf("Insert(%v,%v) returned error: %v", p.x, p.y, err)
		}
	}

	got := tree.SearchNearestN(0, 0, 3)
	if len(got) != 3 {
		t.Fatalf("expected 3 results, got %d", len(got))
	}

	if got[0].Payload != "a" || got[1].Payload != "d" || got[2].Payload != "b" {
		t.Fatalf("unexpected order: %#v", got)
	}
}

func TestSearchNearestNHandlesNonPositiveN(t *testing.T) {
	tree, err := New[int](0, 0, 10, 10, 2)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	if err := tree.Insert(2, 2, 1); err != nil {
		t.Fatalf("Insert returned error: %v", err)
	}

	got := tree.SearchNearestN(0, 0, 0)
	if len(got) != 0 {
		t.Fatalf("expected empty result for n=0, got %d", len(got))
	}

	got = tree.SearchNearestN(0, 0, -3)
	if len(got) != 0 {
		t.Fatalf("expected empty result for negative n, got %d", len(got))
	}
}

func TestSearchNearestNReturnsAllWhenNTooLarge(t *testing.T) {
	tree, err := New[int](0, 0, 10, 10, 2)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	if err := tree.Insert(1, 1, 10); err != nil {
		t.Fatalf("Insert returned error: %v", err)
	}

	if err := tree.Insert(2, 2, 20); err != nil {
		t.Fatalf("Insert returned error: %v", err)
	}

	got := tree.SearchNearestN(0, 0, 10)
	if len(got) != 2 {
		t.Fatalf("expected 2 results, got %d", len(got))
	}
}
