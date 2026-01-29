// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package parser

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
	for k, v := range l.cache {
		result[k] = v
	}
	return
}
