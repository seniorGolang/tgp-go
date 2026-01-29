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
	filesGenerated int64
	linesGenerated int64
	cacheHits      int64
	cacheMisses    int64
}

func incrementFilesGenerated() {

	atomic.AddInt64(&stats.filesGenerated, 1)
}

func addLinesGenerated(lines int64) {

	atomic.AddInt64(&stats.linesGenerated, lines)
}

func onFileSaved(filepath string, lines int64) {

	incrementFilesGenerated()
	addLinesGenerated(lines)
}

func incrementCacheHits() {

	atomic.AddInt64(&stats.cacheHits, 1)
}

func incrementCacheMisses() {

	atomic.AddInt64(&stats.cacheMisses, 1)
}

func resetStats() {

	atomic.StoreInt64(&stats.filesGenerated, 0)
	atomic.StoreInt64(&stats.linesGenerated, 0)
	atomic.StoreInt64(&stats.cacheHits, 0)
	atomic.StoreInt64(&stats.cacheMisses, 0)
}

func logStats(contractID ...string) {

	files := atomic.LoadInt64(&stats.filesGenerated)
	lines := atomic.LoadInt64(&stats.linesGenerated)
	hits := atomic.LoadInt64(&stats.cacheHits)
	misses := atomic.LoadInt64(&stats.cacheMisses)

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
