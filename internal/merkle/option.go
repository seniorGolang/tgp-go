// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.

package merkle

type Option func(*config)

type config struct {
	excludePrefixes []string
}

// ExcludePrefixes: пути, начинающиеся с одного из префиксов (с учётом нормализации слэшей), не включаются в дерево.
func ExcludePrefixes(prefixes ...string) (o Option) {

	return func(c *config) {
		c.excludePrefixes = prefixes
	}
}
