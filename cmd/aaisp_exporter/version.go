package main

// version is the default build version. It is overridden at build time via:
//
//	go build -ldflags "-X main.version=x.y.z"
var version = "dev"
