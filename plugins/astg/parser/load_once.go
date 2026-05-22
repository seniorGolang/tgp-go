// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package parser

import (
	"log/slog"
	"sync"
)

type packageLoadFlight struct {
	wg   sync.WaitGroup
	info *PackageInfo
	err  error
}

func (l *AutonomousPackageLoader) cachedPackage(pkgPath string) (info *PackageInfo, ok bool) {

	l.mu.RLock()
	defer l.mu.RUnlock()
	info, found := l.cache[pkgPath]
	return info, found && info != nil
}

func (l *AutonomousPackageLoader) loadPackageOnce(pkgPath string, load func() (*PackageInfo, error)) (info *PackageInfo, err error) {

	l.mu.Lock()
	var flight *packageLoadFlight
	var waiting bool
	if flight, waiting = l.inflight[pkgPath]; waiting {
		l.mu.Unlock()
		flight.wg.Wait()
		return flight.info, flight.err
	}
	flight = &packageLoadFlight{}
	flight.wg.Add(1)
	l.inflight[pkgPath] = flight
	l.mu.Unlock()

	traceStep("loadPackageOnce leader", slog.String("pkgPath", pkgPath))
	flight.info, flight.err = load()
	if flight.err != nil {
		traceStep("loadPackageOnce failed", slog.String("pkgPath", pkgPath), slog.String("error", flight.err.Error()))
	} else {
		traceStep("loadPackageOnce ok", slog.String("pkgPath", pkgPath))
	}

	l.mu.Lock()
	delete(l.inflight, pkgPath)
	if flight.info != nil {
		l.cache[pkgPath] = flight.info
	}
	l.mu.Unlock()

	flight.wg.Done()

	return flight.info, flight.err
}
