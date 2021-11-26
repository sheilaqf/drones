# Drones

## How to test using go commands:

go test ./...

## How to test using curl commands:
### Registering a drone

curl -H "Content-Type: application/json" -v -X POST "http://localhost:8099/drone/register" --data '{"serial_number":"SQF-831030_1400","model":"Cruiserweight","weight_limit":350,"battery_capacity":100,"state":"IDLE"}'

### To load medications on a drone
curl -H "Content-Type: application/json" -v -X POST "http://localhost:8099/drone/load" --data '{
    "serial_number":"SQF-831030_1400",
    "medications":[
    {
        "name":"dipirona",
        "weight":110,
		"code":"DIP_10"
    },
    {
        "name":"condoms",
        "weight":50,
		"code":"COMDOMS_27"
    },
    {
        "name":"oxigened-water",
        "weight":100,
		"code":"COMDOMS_27"
    }
]
}'

### To get all drones availables for loading
curl -v "http://localhost:8099/drone/all/availables"
### Checking a drone battery capacity
curl -v "http://localhost:8099/drone/battery?serial_number=SQF-831030_1400"

### Checking medications loaded on a drone
curl -v "http://localhost:8099/drone/medications?serial_number=SQF-831030_1400"

## How to build for current SO:

go build cmd/drones.go

## To build for Linux, please run the batch.bat file (you need to edit it in case of build for other SO):

batch.bat

## How to run:

go run cmd/drones.go