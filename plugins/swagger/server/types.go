// Copyright (c) 2020 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package server

// server управляет HTTP сервером для Swagger UI.
type server struct {
	serverID  uint64
	specBytes []byte
}
