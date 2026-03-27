package quadtree

import (
	"fmt"
	"testing"
	"time"
)

func BenchmarkInsertByDatasetSize(b *testing.B) {
	datasetSizes := []int{10_000, 25_000, 50_000, 100_000, 250_000, 500_000, 1_000_000}

	for _, dbSize := range datasetSizes {
		points := makePoints(dbSize, int64(dbSize))
		b.Run(caseName(dbSize, "insert"), func(b *testing.B) {
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

func BenchmarkSearchInRectByPrecision(b *testing.B) {
	datasetSizes := []int{100_000, 250_000, 500_000, 750_000, 1_000_000}
	precisions := []precisionCase{
		{name: "p005", side: 5},
		{name: "p010", side: 10},
		{name: "p025", side: 25},
		{name: "p060", side: 60},
		{name: "p120", side: 120},
	}

	for _, dbSize := range datasetSizes {
		points := makePoints(dbSize, int64(dbSize*3))
		tree := buildTreeFromPoints(b, points, 16)

		for _, p := range precisions {
			name := fmt.Sprintf("%s/precision=%s", caseName(dbSize, "rect"), p.name)
			b.Run(name, func(b *testing.B) {
				b.ReportAllocs()

				started := time.Now()
				totalHits := 0
				for i := 0; i < b.N; i++ {
					q := points[i%len(points)]
					minX, minY, maxX, maxY := queryRectAroundPoint(q, p.side)

					res, err := tree.SearchInRect(minX, minY, maxX, maxY)
					if err != nil {
						b.Fatalf("SearchInRect returned error: %v", err)
					}
					totalHits += len(res)
				}
				reportQueryMetrics(b, started, totalHits)
			})
		}
	}
}

func BenchmarkSearchNearestNByRequestedKeys(b *testing.B) {
	datasetSizes := []int{100_000, 250_000, 500_000, 750_000, 1_000_000}
	requestedKeys := []int{1, 5, 10, 25, 50, 100, 250, 500, 1000}

	for _, dbSize := range datasetSizes {
		points := makePoints(dbSize, int64(dbSize*7))
		tree := buildTreeFromPoints(b, points, 16)

		for _, keyCount := range requestedKeys {
			name := fmt.Sprintf("%s/keys=%d", caseName(dbSize, "nearest"), keyCount)
			b.Run(name, func(b *testing.B) {
				b.ReportAllocs()

				started := time.Now()
				totalHits := 0
				for i := 0; i < b.N; i++ {
					q := points[i%len(points)]
					res := tree.SearchNearestN(q.x, q.y, keyCount)
					totalHits += len(res)
				}
				reportQueryMetrics(b, started, totalHits)
			})
		}
	}
}
