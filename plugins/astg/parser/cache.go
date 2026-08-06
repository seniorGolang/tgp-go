// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package parser

import "maps"

func (l *AutonomousPackageLoader) GetPackage(pkgPath string) (info *PackageInfo, ok bool) {

	l.mu.RLock()
	defer l.mu.RUnlock()
	info, ok = l.cache[pkgPath]
	return
}

func (l *AutonomousPackageLoader) GetAllPackages() (result map[string]*PackageInfo) {

	l.mu.RLock()
	defer l.mu.RUnlock()
	result = make(map[string]*PackageInfo, len(l.cache))
	maps.Copy(result, l.cache)
	return
}
