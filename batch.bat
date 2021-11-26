@echo off
set GOARCH=amd64
set GOOS=linux
for /f %%i in ('git rev-parse HEAD') do set git_command=%%i
echo %git_command%

go build cmd/drones.go

scp -r drones config.dev.json my_linux:/service/go-playground
