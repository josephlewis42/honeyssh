// Package logger is a standardized event logging framework for the honeypot.
package logger

//go:generate protoc --go_out=. --go_opt=paths=source_relative  log.proto
//go:generate protoc --go-json_out=paths=source_relative:. log.proto
