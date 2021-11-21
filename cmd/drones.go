package main

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/Pallinder/go-randomdata"
	"github.com/gorilla/mux"

	"drones/pkg/config"
	"drones/pkg/drone"
	"drones/pkg/medication"
)

type (
	environment struct {
		HttpServer       *http.Server
		Router           *mux.Router
		Config           *config.Config
		registeredDrones map[string]*drone.Drone // use SerialNumber as key
	}
)

var samplMedicationCaseBase64 string

func main() {

	log.Println("Initializing Drones Management API.")
	configFlag := flag.String("config", "config.dev.json", "path to config json file")
	flag.Parse()

	if *configFlag == "" {
		flag.Usage()
		log.Fatalln("config file is missing")
	}

	log.Println("parsing config file...")
	cfg, err := config.Parse(*configFlag)
	if err != nil {
		log.Fatal(err)
	}

	env := environment{
		Config: cfg,
	}

	err = env.preloadData()
	if err != nil {
		log.Printf("preload of data failed: %v", err)
	}
	log.Println("preload of data successfully completed...")

	go env.checkDronesBatteryLevelsPeriodically()

	var wg sync.WaitGroup
	wg.Add(1)
	//capturing signal of closing of the application
	go func(wg *sync.WaitGroup) {
		defer wg.Done()
		c := make(chan os.Signal, 1) // we need to reserve to buffer size 1, so the notifier are not blocked
		signal.Notify(c, os.Interrupt, syscall.SIGTERM)
		<-c
	}(&wg)

	go env.startServer(cfg)
	wg.Wait()

	log.Println("Drones Management API is now closed")
}

func (env *environment) startServer(cfg *config.Config) {

	env.Router = mux.NewRouter()

	env.Router.HandleFunc("/drone/register", env.registerDrone).Methods("POST")
	env.Router.HandleFunc("/drone/load", env.loadMedications).Methods("POST")
	env.Router.HandleFunc("/drone/medications", env.getMedicationsFromDrone).Methods("GET")
	env.Router.HandleFunc("/drone/available", env.getDronesAvailablesForLoading).Methods("GET")
	env.Router.HandleFunc("/drone/all", env.getAllDrones).Methods("GET")

	env.HttpServer = &http.Server{
		Handler:           env.Router,
		Addr:              fmt.Sprintf(":%s", env.Config.ApiPort),
		WriteTimeout:      60 * time.Second,
		ReadTimeout:       5 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Println("listening on:", env.HttpServer.Addr)
	log.Fatal(env.HttpServer.ListenAndServe())
}

func (env *environment) registerDrone(w http.ResponseWriter, r *http.Request) {
	d := drone.DroneDTO{}

	err := json.NewDecoder(r.Body).Decode(&d)
	if err != nil {
		log.Println("could not decode drone json object:", err)
		writeError(w, http.StatusBadRequest)
		return
	}

	droneObj, err := drone.NewDrone(d)
	if err != nil {
		log.Println("could not obtain drone object from dto:", err)
		writeError(w, http.StatusBadRequest)
		return
	}

	err = env.addNewDrone(droneObj)
	if err != nil {
		log.Println("could not add new drone:", err)
		writeError(w, http.StatusBadRequest)
		return
	}

	log.Printf("new drone added: %+v", droneObj)
}

func (env *environment) loadMedications(w http.ResponseWriter, r *http.Request) {
	d := drone.DroneDTO{}

	err := json.NewDecoder(r.Body).Decode(&d)
	if err != nil {
		log.Println("could not decode drone json object:", err)
		writeError(w, http.StatusBadRequest)
		return
	}

	err = env.setLoadForDrone(d)
	if err != nil {
		log.Println("error while trying to load medications on drone:", err)
		writeError(w, http.StatusBadRequest)
		return
	}
}

func (env *environment) getMedicationsFromDrone(w http.ResponseWriter, r *http.Request) {

	parameter := "serial_number"
	k, ok := r.URL.Query()[parameter]

	if !ok || len(k) < 1 {
		log.Printf("request lacks of parameter '%s'", parameter)
		writeError(w, http.StatusBadRequest)
		return
	}

	serialNumber := k[0]
	droneObj := env.registeredDrones[serialNumber]
	if droneObj == nil {
		log.Printf("drone with serial number '%s' was not found", serialNumber)
		writeError(w, http.StatusNotFound)
		return
	}

	err := json.NewEncoder(w).Encode(droneObj.GetDTO().Medications)
	if err != nil {
		log.Println("could not encode response:", err)
		writeError(w, http.StatusInternalServerError)
		return
	}

	log.Printf("%v", droneObj.GetDTO().Medications)
}

func (env *environment) getDronesAvailablesForLoading(w http.ResponseWriter, r *http.Request) {

	drones := make([]drone.DroneDTO, 0)

	for _, v := range env.registeredDrones {
		if v.IsAvailableForLoading() {
			drones = append(drones, v.GetDTO())
		} /* else {
			log.Printf("not available S/N: %s State: %s Battery: %d", v.GetSerialNumber(), v.GetState(), v.GetBatteryCapacity())
		} */
	}

	err := json.NewEncoder(w).Encode(drones)
	if err != nil {
		log.Println("could not encode response:", err)
		writeError(w, http.StatusInternalServerError)
		return
	}

	log.Printf("%v", drones)
}

func (env *environment) getAllDrones(w http.ResponseWriter, r *http.Request) {

	drones := make([]drone.DroneDTO, 0)

	for _, v := range env.registeredDrones {
		drones = append(drones, v.GetDTO())
	}

	err := json.NewEncoder(w).Encode(drones)
	if err != nil {
		log.Println("could not encode response:", err)
		writeError(w, http.StatusInternalServerError)
		return
	}

	log.Printf("%v", drones)
}

func (env *environment) addNewDrone(droneObj *drone.Drone) error {

	if env.registeredDrones == nil {
		env.registeredDrones = make(map[string]*drone.Drone)
	}

	if env.registeredDrones[droneObj.GetSerialNumber()] != nil {
		return fmt.Errorf("drone with serial number %s already exists", droneObj.GetSerialNumber())
	}

	env.registeredDrones[droneObj.GetSerialNumber()] = droneObj

	return nil
}

func (env *environment) setLoadForDrone(load drone.DroneDTO) error {

	if env.registeredDrones[load.SerialNumber] == nil {
		return fmt.Errorf("there is not a drone with serial number %s", load.SerialNumber)
	}

	return env.registeredDrones[load.SerialNumber].LoadSetOfMedications(load.Medications)
}

func (env *environment) preloadData() error {

	//loadSamplMedicationCaseBase64()

	env.registeredDrones = make(map[string]*drone.Drone)

	droneDTO1 := drone.DroneDTO{
		SerialNumber:    randomdata.Alphanumeric(100),
		Model:           drone.ModelLightweight,
		WeightLimit:     150,
		BatteryCapacity: 100,
		State:           drone.StateIdle,
		Medications: []medication.MedicationDTO{
			{
				Name:   "Medication-A",
				Code:   strings.ToUpper(randomdata.Alphanumeric(32)),
				Weight: 20,
				Image:  samplMedicationCaseBase64,
			},
			{
				Name:   "Medication-B",
				Code:   strings.ToUpper(randomdata.Alphanumeric(32)),
				Weight: 40,
				Image:  samplMedicationCaseBase64,
			},
			{
				Name:   "Medication-C",
				Code:   strings.ToUpper(randomdata.Alphanumeric(32)),
				Weight: 25,
				Image:  samplMedicationCaseBase64,
			},
			{
				Name:   "Medication-D",
				Code:   strings.ToUpper(randomdata.Alphanumeric(32)),
				Weight: 10,
				Image:  samplMedicationCaseBase64,
			},
		},
	}
	drone1, err := drone.NewDrone(droneDTO1)
	if err != nil {
		return fmt.Errorf("error while creating preloaded drone with serial number %s:%v", droneDTO1.SerialNumber, err)
	}
	env.registeredDrones[droneDTO1.SerialNumber] = drone1

	droneDTO2 := drone.DroneDTO{
		SerialNumber:    randomdata.Alphanumeric(100),
		Model:           drone.ModelHeavyweight,
		WeightLimit:     500,
		BatteryCapacity: 100,
		State:           drone.StateIdle,
		Medications: []medication.MedicationDTO{
			{
				Name:   "Medication-A",
				Code:   strings.ToUpper(randomdata.Alphanumeric(32)),
				Weight: 200,
				Image:  samplMedicationCaseBase64,
			},
			{
				Name:   "Medication-B",
				Code:   strings.ToUpper(randomdata.Alphanumeric(32)),
				Weight: 80,
				Image:  samplMedicationCaseBase64,
			},
			{
				Name:   "Medication-C",
				Code:   strings.ToUpper(randomdata.Alphanumeric(32)),
				Weight: 50,
				Image:  samplMedicationCaseBase64,
			},
			{
				Name:   "Medication-D",
				Code:   strings.ToUpper(randomdata.Alphanumeric(32)),
				Weight: 60,
				Image:  samplMedicationCaseBase64,
			},
		},
	}
	drone2, err := drone.NewDrone(droneDTO2)
	if err != nil {
		return fmt.Errorf("error while creating preloaded drone with serial number %s:%v", droneDTO2.SerialNumber, err)
	}
	env.registeredDrones[droneDTO2.SerialNumber] = drone2

	droneDTO3 := drone.DroneDTO{
		SerialNumber:    randomdata.Alphanumeric(100),
		Model:           drone.ModelMiddleweight,
		WeightLimit:     300,
		BatteryCapacity: 100,
		State:           drone.StateIdle,
	}
	drone3, err := drone.NewDrone(droneDTO3)
	if err != nil {
		return fmt.Errorf("error while creating preloaded drone with serial number %s:%v", droneDTO3.SerialNumber, err)
	}
	env.registeredDrones[droneDTO3.SerialNumber] = drone3

	droneDTO4 := drone.DroneDTO{
		SerialNumber:    randomdata.Alphanumeric(100),
		Model:           drone.ModelCruiserweight,
		WeightLimit:     400,
		BatteryCapacity: 100,
		State:           drone.StateIdle,
		Medications: []medication.MedicationDTO{
			{
				Name:   "Medication-C",
				Code:   strings.ToUpper(randomdata.Alphanumeric(32)),
				Weight: 300,
				Image:  samplMedicationCaseBase64,
			},
			{
				Name:   "Medication-D",
				Code:   strings.ToUpper(randomdata.Alphanumeric(32)),
				Weight: 90,
				Image:  samplMedicationCaseBase64,
			},
		},
	}
	drone4, err := drone.NewDrone(droneDTO4)
	if err != nil {
		return fmt.Errorf("error while creating preloaded drone with serial number %s:%v", droneDTO4.SerialNumber, err)
	}
	env.registeredDrones[droneDTO4.SerialNumber] = drone4

	droneDTO5 := drone.DroneDTO{
		SerialNumber:    randomdata.Alphanumeric(100),
		Model:           drone.ModelLightweight,
		WeightLimit:     125,
		BatteryCapacity: 100,
		State:           drone.StateIdle,
	}
	drone5, err := drone.NewDrone(droneDTO5)
	if err != nil {
		return fmt.Errorf("error while creating preloaded drone with serial number %s:%v", droneDTO5.SerialNumber, err)
	}
	env.registeredDrones[droneDTO5.SerialNumber] = drone5

	env.printDataOfDrones()

	return nil
}

func (env *environment) printDataOfDrones() {

	log.Printf("data of the %d registered drones:", len(env.registeredDrones))

	for k, v := range env.registeredDrones {
		log.Printf("drone with serial number %s:", k)
		log.Printf("%+v:", v)
	}
}

func (env *environment) checkDronesBatteryLevelsPeriodically() {

	ticker := time.NewTicker(time.Duration(env.Config.LogPeriodMinutes) * time.Minute)
	defer ticker.Stop()
	for {
		select {
		/* 		case <-stop:
		log.Println("external command: periodic check of drones battery levels is stopped, due to restart signal")
		return */
		case <-ticker.C:
			log.Print("check of battery levels:")
			for k, v := range env.registeredDrones {
				log.Printf("drone serial number: %s has a battery level of %d %%", k, v.GetBatteryCapacity())
			}
		}
	}
}

func writeError(w http.ResponseWriter, statusCode int) {
	w.WriteHeader(statusCode)
	_, _ = fmt.Fprintln(w, statusCode, http.StatusText(statusCode))
}

func loadSamplMedicationCaseBase64() {
	// Open file on disk.
	imagePath := "sample_medication_case_base64.jpg"
	f, err := os.Open(imagePath)
	if err != nil {
		log.Printf("it was not possible to open %s: %v", imagePath, err)
	}

	// Read entire JPG into byte slice.
	reader := bufio.NewReader(f)
	content, err := ioutil.ReadAll(reader)
	if err != nil {
		log.Printf("error reading content of file %s: %v", imagePath, err)
	}
	// Encode as base64.
	samplMedicationCaseBase64 = base64.StdEncoding.EncodeToString(content)
}
