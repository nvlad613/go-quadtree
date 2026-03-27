package quadtree

import (
	"fmt"
	"math"
	"math/rand"
	"sort"
	"testing"
	"time"
)

type benchPoint struct {
	x float64
	y float64
}

type precisionCase struct {
	name string
	side float64
}

func makePoints(n int, seed int64) []benchPoint {
	r := rand.New(rand.NewSource(seed))
	points := make([]benchPoint, n)
	for i := 0; i < n; i++ {
		points[i] = benchPoint{
			x: r.Float64() * 999.999,
			y: r.Float64() * 999.999,
		}
	}
	return points
}

func buildTreeFromPoints(tb testing.TB, points []benchPoint, maxPerNode int) *Tree[int] {
	tb.Helper()

	tree, err := New[int](0, 0, 1000, 1000, maxPerNode)
	if err != nil {
		tb.Fatalf("New returned error: %v", err)
	}

	for i, p := range points {
		if err = tree.Insert(p.x, p.y, i); err != nil {
			tb.Fatalf("Insert(%d) returned error: %v", i, err)
		}
	}

	return tree
}

func queryRectAroundPoint(p benchPoint, side float64) (minX, minY, maxX, maxY float64) {
	half := side / 2
	minX = clamp(p.x-half, 0, 1000-side)
	minY = clamp(p.y-half, 0, 1000-side)
	maxX = minX + side
	maxY = minY + side
	return minX, minY, maxX, maxY
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func reportQueryMetrics(b *testing.B, started time.Time, totalHits int) {
	elapsed := time.Since(started)
	if elapsed <= 0 {
		return
	}

	b.ReportMetric(float64(b.N)/elapsed.Seconds(), "query_ops/s")
	b.ReportMetric(float64(totalHits)/float64(b.N), "hits/query")
}

func reportBulkInsertMetrics(b *testing.B, started time.Time, inserts int) {
	elapsed := time.Since(started)
	if elapsed <= 0 {
		return
	}
	b.ReportMetric(float64(inserts)/elapsed.Seconds(), "insert_ops/s")
}

func reportLatencyMetrics(b *testing.B, durs []time.Duration, elapsed time.Duration) {
	if len(durs) == 0 || elapsed <= 0 {
		return
	}

	copyDurs := append([]time.Duration(nil), durs...)
	sort.Slice(copyDurs, func(i, j int) bool { return copyDurs[i] < copyDurs[j] })

	p50 := copyDurs[len(copyDurs)/2]
	p95Idx := int(math.Ceil(float64(len(copyDurs))*0.95)) - 1
	if p95Idx < 0 {
		p95Idx = 0
	}
	if p95Idx >= len(copyDurs) {
		p95Idx = len(copyDurs) - 1
	}
	p95 := copyDurs[p95Idx]

	b.ReportMetric(float64(len(copyDurs))/elapsed.Seconds(), "throughput_ops/s")
	b.ReportMetric(float64(p50.Nanoseconds())/1e6, "p50_ms")
	b.ReportMetric(float64(p95.Nanoseconds())/1e6, "p95_ms")
}

func caseName(dbSize int, suffix string) string {
	return fmt.Sprintf("db=%d/%s", dbSize, suffix)
}
