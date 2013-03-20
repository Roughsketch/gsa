@echo off
for %%f in (*.go) do (
	go build "%%~nf.go"
)