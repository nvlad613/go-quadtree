package quadtree

import (
	"fmt"
	"testing"
	"time"
)

func BenchmarkPerfLargeDatasetInsert(b *testing.B) {
	largeDatasetSizes := []int{250_000, 500_000, 750_000, 1_000_000}

	for _, dbSize := range largeDatasetSizes {
		points := makePoints(dbSize, int64(dbSize*11))
		b.Run(caseName(dbSize, "bulk_insert"), func(b *testing.B) {
			b.ReportAllocs()
			started := time.Now()
			totalInserts := 0

			for i := 0; i < b.N; i++ {
				tree, err := New[int](0, 0, 1000, 1000, 16)
				if err != nil {
					b.Fatalf("New returned error: %v", err)
				}
				for idx, p := range points {
					if err = tree.Insert(p.x, p.y, idx); err != nil {
						b.Fatalf("Insert(%d) returned error: %v", idx, err)
					}
				}
				totalInserts += len(points)
			}

			reportBulkInsertMetrics(b, started, totalInserts)
		})
	}
}

func BenchmarkPerfLargeDatasetSearchNearestN(b *testing.B) {
	largeDatasetSizes := []int{250_000, 500_000, 750_000, 1_000_000}
	keyCounts := []int{10, 25, 50, 100, 250, 500, 1000}

	for _, dbSize := range largeDatasetSizes {
		points := makePoints(dbSize, int64(dbSize*13))
		tree := buildTreeFromPoints(b, points, 16)

		for _, keyCount := range keyCounts {
			name := fmt.Sprintf("%s/keys=%d", caseName(dbSize, "nearest_latency"), keyCount)
			b.Run(name, func(b *testing.B) {
				b.ReportAllocs()
				latencySamples := make([]time.Duration, 0, b.N)
				started := time.Now()

				for i := 0; i < b.N; i++ {
					q := points[i%len(points)]
					t0 := time.Now()
					_ = tree.SearchNearestN(q.x, q.y, keyCount)
					latencySamples = append(latencySamples, time.Since(t0))
				}

				reportLatencyMetrics(b, latencySamples, time.Since(started))
			})
		}
	}
}

func BenchmarkPerfLargeDatasetSearchInRect(b *testing.B) {
	largeDatasetSizes := []int{250_000, 500_000, 750_000, 1_000_000}
	precisions := []precisionCase{
		{name: "p005", side: 5},
		{name: "p010", side: 10},
		{name: "p025", side: 25},
		{name: "p060", side: 60},
		{name: "p120", side: 120},
	}

	for _, dbSize := range largeDatasetSizes {
		points := makePoints(dbSize, int64(dbSize*17))
		tree := buildTreeFromPoints(b, points, 16)

		for _, p := range precisions {
			name := fmt.Sprintf("%s/precision=%s", caseName(dbSize, "rect_latency"), p.name)
			b.Run(name, func(b *testing.B) {
				b.ReportAllocs()
				latencySamples := make([]time.Duration, 0, b.N)
				started := time.Now()

				for i := 0; i < b.N; i++ {
					q := points[i%len(points)]
					minX, minY, maxX, maxY := queryRectAroundPoint(q, p.side)

					t0 := time.Now()
					_, err := tree.SearchInRect(minX, minY, maxX, maxY)
					if err != nil {
						b.Fatalf("SearchInRect returned error: %v", err)
					}
					latencySamples = append(latencySamples, time.Since(t0))
				}

				reportLatencyMetrics(b, latencySamples, time.Since(started))
			})
		}
	}
}

func BenchmarkProfileInsert(b *testing.B) {
	points := makePoints(500_000, 21)
	tree, err := New[int](0, 0, 1000, 1000, 16)
	if err != nil {
		b.Fatalf("New returned error: %v", err)
	}

	idx := 0
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p := points[idx]
		if err = tree.Insert(p.x, p.y, idx); err != nil {
			b.Fatalf("Insert(%d) returned error: %v", idx, err)
		}
		idx++
		if idx == len(points) {
			tree, err = New[int](0, 0, 1000, 1000, 16)
			if err != nil {
				b.Fatalf("New returned error: %v", err)
			}
			idx = 0
		}
	}
}

func BenchmarkProfileSearchInRect(b *testing.B) {
	points := makePoints(500_000, 22)
	tree := buildTreeFromPoints(b, points, 16)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		q := points[i%len(points)]
		minX, minY, maxX, maxY := queryRectAroundPoint(q, 25)
		_, err := tree.SearchInRect(minX, minY, maxX, maxY)
		if err != nil {
			b.Fatalf("SearchInRect returned error: %v", err)
		}
	}
}

func BenchmarkProfileSearchNearestN(b *testing.B) {
	points := makePoints(500_000, 23)
	tree := buildTreeFromPoints(b, points, 16)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		q := points[i%len(points)]
		_ = tree.SearchNearestN(q.x, q.y, 100)
	}
}
