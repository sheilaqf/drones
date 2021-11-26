# Drones

##How to test:

go test ./...

### Registering a drone using curl command

curl -H "Content-Type: application/json" -v -X POST "http://localhost:8099/drone/register" --dat
a '{"serial_number":"SQF-831030_1400","model":"Cruiserweight","weight_limit":350,"battery_capacity":100,"state":"IDLE"}'

##How to build for current SO:

go build cmd/drones.go

##How to build for Linux:

batch.bat

##How to run:

go run cmd/drones.go