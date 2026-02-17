// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.

package er

type config struct{}

func newConfig() (c *config) {
	return &config{}
}

type Option func(*config)
