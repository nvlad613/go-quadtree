package main

import (
	"bytes"
	"context"
	"encoding/csv"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

var (
	benchLineRE = regexp.MustCompile(`^(Benchmark\S+)-\d+\s+(\d+)\s+([0-9.]+)\s+ns/op(?:\s+([0-9.]+)\s+B/op\s+([0-9.]+)\s+allocs/op)?(.*)$`)
	metricRE    = regexp.MustCompile(`([0-9.]+)\s+([A-Za-z0-9_/]+)`)
)

const (
	defaultOutDir       = "docs/perf/out"
	defaultPkg          = "./pkg/quadtree"
	defaultCount        = 3
	defaultBenchTime    = "1s"
	insertRawFileName   = "insert-raw.txt"
	searchRawFileName   = "search-raw.txt"
	insertCSVFileName   = "insert.csv"
	searchCSVFileName   = "search.csv"
	insertMetricKey     = "insert_ops/s"
	queryMetricKey      = "query_ops/s"
	throughputMetricKey = "throughput_ops/s"
	hitsMetricKey       = "hits/query"
	p50MetricKey        = "p50_ms"
	p95MetricKey        = "p95_ms"
)

var csvHeader = []string{
	"timestamp",
	"count",
	"benchtime",
	"benchmark",
	"series",
	"operation",
	"dataset_size",
	"keys",
	"precision",
	"iterations",
	"ns_per_op",
	"bytes_per_op",
	"allocs_per_op",
	"insert_ops_per_s",
	"query_ops_per_s",
	"throughput_ops_s",
	"hits_per_query",
	"p50_ms",
	"p95_ms",
}

type benchRecord struct {
	Timestamp      string
	Count          string
	Benchtime      string
	Benchmark      string
	Series         string
	Operation      string
	DatasetSize    string
	DatasetSizeNum int
	Keys           string
	KeysNum        int
	Precision      string
	Iterations     string
	NSPerOp        string
	BytesPerOp     string
	AllocsPerOp    string
	InsertOpsPerS  string
	QueryOpsPerS   string
	ThroughputOpsS string
	HitsPerQuery   string
	P50MS          string
	P95MS          string
}

type runConfig struct {
	outDir      string
	pkg         string
	count       int
	benchtime   string
	insertBench string
	searchBench string
}

type runMeta struct {
	timestamp string
	countStr  string
	benchTime string
}

type rawOutputs struct {
	insert string
	search string
}

type parsedBenchLine struct {
	benchmark   string
	iterations  string
	nsPerOp     string
	bytesPerOp  string
	allocsPerOp string
	tail        string
}

func main() {
	cfg := parseFlags()
	if err := run(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "benchcsv: %v\n", err)
		os.Exit(1)
	}
}

func parseFlags() runConfig {
	var cfg runConfig
	flag.StringVar(&cfg.outDir, "out", filepath.FromSlash(defaultOutDir), "Output directory for CSV and raw benchmark logs")
	flag.StringVar(&cfg.pkg, "pkg", defaultPkg, "Package with benchmarks")
	flag.IntVar(&cfg.count, "count", defaultCount, "go test -count value")
	flag.StringVar(&cfg.benchtime, "benchtime", defaultBenchTime, "go test -benchtime value")
	flag.StringVar(&cfg.insertBench, "insert-bench", `Benchmark(InsertByDatasetSize|PerfLargeDatasetInsert)$`, "Regex for insert benchmarks")
	flag.StringVar(&cfg.searchBench, "search-bench", `Benchmark(SearchInRectByPrecision|SearchNearestNByRequestedKeys|PerfLargeDatasetSearchInRect|PerfLargeDatasetSearchNearestN)$`, "Regex for search benchmarks")
	flag.Parse()
	return cfg
}

func run(cfg runConfig) error {
	if cfg.count <= 0 {
		return errors.New("count must be greater than zero")
	}

	if err := os.MkdirAll(cfg.outDir, 0o755); err != nil {
		return fmt.Errorf("create out directory: %w", err)
	}

	meta := runMeta{
		timestamp: time.Now().Format(time.RFC3339),
		countStr:  strconv.Itoa(cfg.count),
		benchTime: cfg.benchtime,
	}

	raw, err := runBenchmarks(context.Background(), cfg, meta)
	if err != nil {
		return err
	}

	insertRows := parseBenchmarkOutput(raw.insert, meta)
	searchRows := parseBenchmarkOutput(raw.search, meta)
	sortRecords(insertRows)
	sortRecords(searchRows)

	insertRawPath, searchRawPath, err := writeRawOutputs(cfg.outDir, raw)
	if err != nil {
		return err
	}

	insertCSVPath, searchCSVPath, err := writeCSVOutputs(cfg.outDir, insertRows, searchRows)
	if err != nil {
		return err
	}

	printSummary(insertCSVPath, searchCSVPath, insertRawPath, searchRawPath)
	return nil
}

func runBenchmarks(ctx context.Context, cfg runConfig, meta runMeta) (rawOutputs, error) {
	insertOut, err := executeBench(ctx, cfg.pkg, cfg.insertBench, meta.countStr, meta.benchTime)
	if err != nil {
		return rawOutputs{}, err
	}
	searchOut, err := executeBench(ctx, cfg.pkg, cfg.searchBench, meta.countStr, meta.benchTime)
	if err != nil {
		return rawOutputs{}, err
	}
	return rawOutputs{insert: insertOut, search: searchOut}, nil
}

func writeRawOutputs(outDir string, raw rawOutputs) (insertPath, searchPath string, err error) {
	insertPath = filepath.Join(outDir, insertRawFileName)
	searchPath = filepath.Join(outDir, searchRawFileName)

	if err = os.WriteFile(insertPath, []byte(raw.insert), 0o644); err != nil {
		return "", "", fmt.Errorf("write insert raw output: %w", err)
	}
	if err = os.WriteFile(searchPath, []byte(raw.search), 0o644); err != nil {
		return "", "", fmt.Errorf("write search raw output: %w", err)
	}
	return insertPath, searchPath, nil
}

func writeCSVOutputs(outDir string, insertRows, searchRows []benchRecord) (insertPath, searchPath string, err error) {
	insertPath = filepath.Join(outDir, insertCSVFileName)
	searchPath = filepath.Join(outDir, searchCSVFileName)

	if err = writeCSV(insertPath, insertRows); err != nil {
		return "", "", err
	}
	if err = writeCSV(searchPath, searchRows); err != nil {
		return "", "", err
	}
	return insertPath, searchPath, nil
}

func printSummary(insertCSVPath, searchCSVPath, insertRawPath, searchRawPath string) {
	fmt.Printf("Insert CSV: %s\n", insertCSVPath)
	fmt.Printf("Search CSV: %s\n", searchCSVPath)
	fmt.Printf("Insert RAW: %s\n", insertRawPath)
	fmt.Printf("Search RAW: %s\n", searchRawPath)
}

func executeBench(ctx context.Context, pkg, benchRegex, count, benchtime string) (string, error) {
	args := []string{"test", pkg, "-run", "^$", "-bench", benchRegex, "-benchmem", "-count", count, "-benchtime", benchtime}
	cmd := exec.CommandContext(ctx, "go", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("go %s failed: %w\n%s", strings.Join(args, " "), err, string(out))
	}
	return string(out), nil
}

func parseBenchmarkOutput(output string, meta runMeta) []benchRecord {
	lines := strings.Split(output, "\n")
	rows := make([]benchRecord, 0, len(lines))

	for _, line := range lines {
		parsed, ok := parseBenchmarkLine(line)
		if !ok {
			continue
		}
		rows = append(rows, buildRecord(parsed, meta))
	}

	return rows
}

func parseBenchmarkLine(line string) (parsedBenchLine, bool) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return parsedBenchLine{}, false
	}

	m := benchLineRE.FindStringSubmatch(trimmed)
	if len(m) == 0 {
		return parsedBenchLine{}, false
	}

	return parsedBenchLine{
		benchmark:   m[1],
		iterations:  m[2],
		nsPerOp:     m[3],
		bytesPerOp:  m[4],
		allocsPerOp: m[5],
		tail:        m[6],
	}, true
}

func buildRecord(parsed parsedBenchLine, meta runMeta) benchRecord {
	params := parseParams(parsed.benchmark)
	metrics := parseCustomMetrics(parsed.tail)

	dataset := params["db"]
	keys := params["keys"]

	return benchRecord{
		Timestamp:      meta.timestamp,
		Count:          meta.countStr,
		Benchtime:      meta.benchTime,
		Benchmark:      parsed.benchmark,
		Series:         seriesFromName(parsed.benchmark),
		Operation:      operationFromName(parsed.benchmark),
		DatasetSize:    dataset,
		DatasetSizeNum: parseIntOrZero(dataset),
		Keys:           keys,
		KeysNum:        parseIntOrZero(keys),
		Precision:      params["precision"],
		Iterations:     parsed.iterations,
		NSPerOp:        parsed.nsPerOp,
		BytesPerOp:     parsed.bytesPerOp,
		AllocsPerOp:    parsed.allocsPerOp,
		InsertOpsPerS:  metrics[insertMetricKey],
		QueryOpsPerS:   metrics[queryMetricKey],
		ThroughputOpsS: metrics[throughputMetricKey],
		HitsPerQuery:   metrics[hitsMetricKey],
		P50MS:          metrics[p50MetricKey],
		P95MS:          metrics[p95MetricKey],
	}
}

func parseParams(benchmark string) map[string]string {
	params := map[string]string{}
	parts := strings.Split(benchmark, "/")
	if len(parts) < 2 {
		return params
	}
	for _, part := range parts[1:] {
		if !strings.Contains(part, "=") {
			continue
		}
		kv := strings.SplitN(part, "=", 2)
		params[kv[0]] = kv[1]
	}
	return params
}

func parseCustomMetrics(tail string) map[string]string {
	metrics := map[string]string{}
	matches := metricRE.FindAllStringSubmatch(tail, -1)
	for _, m := range matches {
		if len(m) < 3 {
			continue
		}
		metrics[m[2]] = m[1]
	}
	return metrics
}

func operationFromName(benchmark string) string {
	switch {
	case strings.Contains(benchmark, "Insert"):
		return "insert"
	case strings.Contains(benchmark, "SearchInRect"):
		return "search_rect"
	case strings.Contains(benchmark, "SearchNearestN"):
		return "search_nearest"
	default:
		return "unknown"
	}
}

func seriesFromName(benchmark string) string {
	switch {
	case strings.Contains(benchmark, "PerfLargeDataset"):
		return "perf_large"
	case strings.Contains(benchmark, "Profile"):
		return "profile"
	default:
		return "trend"
	}
}

func parseIntOrZero(s string) int {
	if s == "" {
		return 0
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return v
}

func precisionRank(p string) int {
	if p == "" {
		return 0
	}
	digits := strings.TrimLeft(p, "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ_")
	v, err := strconv.Atoi(digits)
	if err != nil {
		return 0
	}
	return v
}

func sortRecords(rows []benchRecord) {
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Series != rows[j].Series {
			return rows[i].Series < rows[j].Series
		}
		if rows[i].Operation != rows[j].Operation {
			return rows[i].Operation < rows[j].Operation
		}
		if rows[i].DatasetSizeNum != rows[j].DatasetSizeNum {
			return rows[i].DatasetSizeNum < rows[j].DatasetSizeNum
		}
		if rows[i].KeysNum != rows[j].KeysNum {
			return rows[i].KeysNum < rows[j].KeysNum
		}
		if precisionRank(rows[i].Precision) != precisionRank(rows[j].Precision) {
			return precisionRank(rows[i].Precision) < precisionRank(rows[j].Precision)
		}
		return rows[i].Benchmark < rows[j].Benchmark
	})
}

func writeCSV(path string, rows []benchRecord) error {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)

	if err := w.Write(csvHeader); err != nil {
		return fmt.Errorf("write csv header: %w", err)
	}

	for _, row := range rows {
		if err := w.Write(row.csvRow()); err != nil {
			return fmt.Errorf("write csv row: %w", err)
		}
	}

	w.Flush()
	if err := w.Error(); err != nil {
		return fmt.Errorf("flush csv writer: %w", err)
	}

	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		return fmt.Errorf("write csv file %s: %w", path, err)
	}
	return nil
}

func (r benchRecord) csvRow() []string {
	return []string{
		r.Timestamp,
		r.Count,
		r.Benchtime,
		r.Benchmark,
		r.Series,
		r.Operation,
		r.DatasetSize,
		r.Keys,
		r.Precision,
		r.Iterations,
		r.NSPerOp,
		r.BytesPerOp,
		r.AllocsPerOp,
		r.InsertOpsPerS,
		r.QueryOpsPerS,
		r.ThroughputOpsS,
		r.HitsPerQuery,
		r.P50MS,
		r.P95MS,
	}
}
