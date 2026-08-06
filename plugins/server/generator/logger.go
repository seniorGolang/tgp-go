// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package generator

import (
	"fmt"
	"log/slog"
	"sync/atomic"

	"tgp/core/i18n"
)

var (
	stats = &generationStats{}
)

type generationStats struct {
	filesGenerated atomic.Int64
	linesGenerated atomic.Int64
	cacheHits      atomic.Int64
	cacheMisses    atomic.Int64
}

func incrementFilesGenerated() {

	stats.filesGenerated.Add(1)
}

func addLinesGenerated(lines int64) {

	stats.linesGenerated.Add(lines)
}

func onFileSaved(_filepath string, lines int64) {

	incrementFilesGenerated()
	addLinesGenerated(lines)
}

func incrementCacheHits() {

	stats.cacheHits.Add(1)
}

func incrementCacheMisses() {

	stats.cacheMisses.Add(1)
}

func resetStats() {

	stats.filesGenerated.Store(0)
	stats.linesGenerated.Store(0)
	stats.cacheHits.Store(0)
	stats.cacheMisses.Store(0)
}

func logStats(contractID ...string) {

	files := stats.filesGenerated.Load()
	lines := stats.linesGenerated.Load()
	hits := stats.cacheHits.Load()
	misses := stats.cacheMisses.Load()

	totalCacheRequests := hits + misses
	cacheHitRate := float64(0)
	if totalCacheRequests > 0 {
		cacheHitRate = float64(hits) / float64(totalCacheRequests) * 100
	}

	args := []any{
		slog.Int64("files", files),
		slog.Int64("lines", lines),
		slog.Int64("cache_hits", hits),
		slog.Int64("cache_misses", misses),
		slog.String("cache_hit_rate", fmt.Sprintf("%.2f%%", cacheHitRate)),
	}
	if len(contractID) > 0 && contractID[0] != "" {
		args = append(args, slog.String("contract", contractID[0]))
	}

	slog.Debug(i18n.Msg("generation statistics"), args...)
}
