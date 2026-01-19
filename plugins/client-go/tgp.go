package main

//go:generate go run -tags pluginInfo . ../../dist/clientGo.json
//go:generate env GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -o ../../dist/clientGo.tgp .
