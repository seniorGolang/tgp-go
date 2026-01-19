// Copyright (c) 2020 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package parser

// GetPackage возвращает пакет из кэша.
func (l *AutonomousPackageLoader) GetPackage(pkgPath string) (info *PackageInfo, ok bool) {

	l.mu.RLock()
	defer l.mu.RUnlock()
	info, ok = l.cache[pkgPath]
	return
}

// GetAllPackages возвращает все загруженные пакеты (для внутреннего использования).
func (l *AutonomousPackageLoader) GetAllPackages() (result map[string]*PackageInfo) {

	l.mu.RLock()
	defer l.mu.RUnlock()
	result = make(map[string]*PackageInfo, len(l.cache))
	for k, v := range l.cache {
		result[k] = v
	}
	return
}

// GetLoadPackageStats возвращает статистику загрузки пакетов.
func (l *AutonomousPackageLoader) GetLoadPackageStats() (result map[string]*loadPackageStat) {

	l.loadPackageStatsMu.RLock()
	defer l.loadPackageStatsMu.RUnlock()
	result = make(map[string]*loadPackageStat, len(l.loadPackageStats))
	for k, v := range l.loadPackageStats {
		statCopy := &loadPackageStat{
			count:     v.count,
			totalTime: v.totalTime,
			maxTime:   v.maxTime,
		}
		result[k] = statCopy
	}
	return
}
