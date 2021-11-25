// Implements a service via REST API that allows clients to communicate with the drones (i.e. **dispatch controller**).
// The specific communication with the drone is outside the scope of this app.
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
	//contains all the global variables to use in the app
	environment struct {
		HttpServer                *http.Server
		Router                    *mux.Router
		Config                    *config.Config
		registeredDrones          map[string]*drone.Drone // use SerialNumber as key
		samplMedicationCaseBase64 string
	}

	//a http response body
	Response struct {
		OK      bool             `json:"ok"`
		Details string           `json:"details,omitempty"`
		Drones  []drone.DroneDTO `json:"drones,omitempty"`
	}
)

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

//start http client
func (env *environment) startServer(cfg *config.Config) {

	env.Router = mux.NewRouter()

	env.Router.HandleFunc("/drone/register", env.registerDrone).Methods("POST")
	env.Router.HandleFunc("/drone/load", env.loadMedications).Methods("POST")
	env.Router.HandleFunc("/drone/medications", env.getMedicationsFromDrone).Methods("GET")
	env.Router.HandleFunc("/drone/battery", env.getBatteryLevelFromDrone).Methods("GET")
	env.Router.HandleFunc("/drone/all/availables", env.getDronesAvailablesForLoading).Methods("GET")
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

//http handler to register a new drone
func (env *environment) registerDrone(w http.ResponseWriter, r *http.Request) {
	d := drone.DroneDTO{}

	err := json.NewDecoder(r.Body).Decode(&d)
	if err != nil {
		errMessage := "could not decode drone json object"
		log.Println(errMessage, ":", err)
		writeError(w, http.StatusBadRequest, errMessage)
		return
	}

	droneObj, err := drone.NewDrone(d)
	if err != nil {
		errMessage := fmt.Sprintf("could not obtain drone object from dto: %s", err.Error())
		log.Println(errMessage)
		writeError(w, http.StatusBadRequest, errMessage)
		return
	}

	err = env.addNewDrone(droneObj)
	if err != nil {
		errMessage := fmt.Sprintf("could not add new drone: %s", err.Error())
		log.Println(errMessage)
		writeError(w, http.StatusBadRequest, errMessage)
		return
	}

	err = json.NewEncoder(w).Encode(Response{
		OK:      true,
		Details: fmt.Sprintf("new drone with serial number %s added", droneObj.GetSerialNumber()),
	})
	if err != nil {
		errMessage := "could not encode response"
		log.Println(errMessage, ":", err)
		writeError(w, http.StatusInternalServerError, errMessage)
		return
	}

	jsonBytes, err := json.MarshalIndent(droneObj.GetDTO(), "", "  ")
	if err != nil {
		log.Printf("new drone added: %+v", droneObj)
	} else {
		log.Printf("new drone added: %+v", string(jsonBytes))
	}

}

//http handler to load medications on a drone
func (env *environment) loadMedications(w http.ResponseWriter, r *http.Request) {

	dto := drone.DroneDTO{}

	err := json.NewDecoder(r.Body).Decode(&dto)
	if err != nil {
		errMessage := "could not decode drone json object"
		log.Println(errMessage, ":", err)
		writeError(w, http.StatusBadRequest, errMessage)
		return
	}

	err = env.setLoadForDrone(dto)
	if err != nil {
		errMessage := fmt.Sprintf("error while trying to load medications on drone: %s", err.Error())
		log.Println(errMessage)
		writeError(w, http.StatusBadRequest, errMessage)
		return
	}

	err = json.NewEncoder(w).Encode(Response{
		OK:      true,
		Details: fmt.Sprintf("medications loaded in drone with serial number %s added", dto.SerialNumber),
	})
	if err != nil {
		errMessage := "could not encode response"
		log.Println(errMessage, ":", err)
		writeError(w, http.StatusInternalServerError, errMessage)
		return
	}

	log.Printf("medications loaded in drone %s", dto.SerialNumber)
}

//http handler to get the medication loaded on a drone
func (env *environment) getMedicationsFromDrone(w http.ResponseWriter, r *http.Request) {

	parameter := "serial_number"
	k, ok := r.URL.Query()[parameter]

	if !ok || len(k) < 1 {
		errMessage := fmt.Sprintf("request lacks of parameter '%s'", parameter)
		log.Println(errMessage)
		writeError(w, http.StatusBadRequest, errMessage)
		return
	}

	serialNumber := k[0]
	droneObj := env.registeredDrones[serialNumber]
	if droneObj == nil {
		errMessage := fmt.Sprintf("drone with serial number '%s' was not found", serialNumber)
		log.Println(errMessage)
		writeError(w, http.StatusNotFound, errMessage)
		return
	}

	if !droneObj.HasMedications() {
		errMessage := fmt.Sprintf("drone with serial number '%s' has not loaded medications", serialNumber)
		log.Println(errMessage)
		writeError(w, http.StatusNotFound, errMessage)
		return
	}

	err := json.NewEncoder(w).Encode(Response{
		OK:      true,
		Details: fmt.Sprintf("this are the medications loaded in drone with serial number %s", droneObj.GetSerialNumber()),
		Drones:  []drone.DroneDTO{droneObj.GetDTOWithSerialNumberAndMedications()},
	})
	if err != nil {
		errMessage := "could not encode response"
		log.Println(errMessage, ":", err)
		writeError(w, http.StatusInternalServerError, errMessage)
		return
	}

	jsonBytes, err := json.MarshalIndent(droneObj.GetDTO().Medications, "", "  ")
	if err != nil {
		log.Printf("medications in drone %s are: \n %v", droneObj.GetSerialNumber(), droneObj.GetDTO().Medications)
	} else {
		log.Printf("medications in drone %s are: \n %v", droneObj.GetSerialNumber(), string(jsonBytes))
	}

}

//http handler to get the battery level of a drone
func (env *environment) getBatteryLevelFromDrone(w http.ResponseWriter, r *http.Request) {

	parameter := "serial_number"
	k, ok := r.URL.Query()[parameter]

	if !ok || len(k) < 1 {
		errMessage := fmt.Sprintf("request lacks of parameter '%s'", parameter)
		log.Println(errMessage)
		writeError(w, http.StatusBadRequest, errMessage)
		return
	}

	serialNumber := k[0]
	droneObj := env.registeredDrones[serialNumber]
	if droneObj == nil {
		errMessage := fmt.Sprintf("drone with serial number '%s' was not found", serialNumber)
		log.Println(errMessage)
		writeError(w, http.StatusNotFound, errMessage)
		return
	}

	err := json.NewEncoder(w).Encode(Response{
		OK:      true,
		Details: fmt.Sprintf("this is the battery capacity of drone with serial number %s", droneObj.GetSerialNumber()),
		Drones:  []drone.DroneDTO{droneObj.GetDTOWithSerialNumberAndBatteryCapacity()},
	})
	if err != nil {
		errMessage := "could not encode response"
		log.Println(errMessage, ":", err)
		writeError(w, http.StatusInternalServerError, errMessage)
		return
	}

	jsonBytes, err := json.MarshalIndent(droneObj.GetDTOWithSerialNumberAndBatteryCapacity(), "", "  ")
	if err != nil {
		log.Printf("battery capacity of drone: %v", droneObj.GetDTOWithSerialNumberAndBatteryCapacity())
	} else {
		log.Printf("battery capacity of drone: %s", string(jsonBytes))
	}

}

//http handler to get all drones availables for loading
func (env *environment) getDronesAvailablesForLoading(w http.ResponseWriter, r *http.Request) {

	drones := make([]drone.DroneDTO, 0)

	for _, v := range env.registeredDrones {
		if v.IsAvailableForLoading() {
			drones = append(drones, v.GetDTOWithSerialNumber())
		} /* else {
			log.Printf("not available S/N: %s State: %s Battery: %d", v.GetSerialNumber(), v.GetState(), v.GetBatteryCapacity())
		} */
	}

	if len(drones) == 0 {
		errMessage := "there is not available drones for loading"
		log.Println(errMessage)
		writeError(w, http.StatusNotFound, errMessage)
		return
	}

	err := json.NewEncoder(w).Encode(Response{
		OK:      true,
		Details: fmt.Sprintf("this are the %d available drones for loading", len(drones)),
		Drones:  drones,
	})
	if err != nil {
		errMessage := "could not encode response"
		log.Println(errMessage, ":", err)
		writeError(w, http.StatusInternalServerError, errMessage)
		return
	}

	jsonBytes, err := json.MarshalIndent(drones, "", "  ")
	if err != nil {
		log.Printf("drones availables for loading: %v", drones)
	} else {
		log.Printf("drones availables for loading: %s", string(jsonBytes))
	}

}

//http handler to get all registered drones
func (env *environment) getAllDrones(w http.ResponseWriter, r *http.Request) {

	drones := make([]drone.DroneDTO, 0)

	for _, v := range env.registeredDrones {
		drones = append(drones, v.GetDTO())
	}

	err := json.NewEncoder(w).Encode(drones)
	if err != nil {
		errMessage := "could not encode response"
		log.Println(errMessage, ":", err)
		writeError(w, http.StatusInternalServerError, errMessage)
		return
	}

	env.printDataOfDrones(drones)
}

//add a new drone to the list of registered drones
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

//set load for a drone using a DTO with the information of the serial number of the drone and the medications to load
func (env *environment) setLoadForDrone(load drone.DroneDTO) error {

	if env.registeredDrones[load.SerialNumber] == nil {
		return fmt.Errorf("there is not a drone with serial number %s", load.SerialNumber)
	}

	return env.registeredDrones[load.SerialNumber].LoadSetOfMedications(load.Medications)
}

//preload during the start of the app, a list with some drones
func (env *environment) preloadData() error {

	env.loadSamplMedicationCaseBase64()

	env.registeredDrones = make(map[string]*drone.Drone)

	droneDTO1 := drone.DroneDTO{
		SerialNumber:    randomdata.Alphanumeric(50),
		Model:           drone.ModelLightweight,
		WeightLimit:     150,
		BatteryCapacity: 100,
		State:           drone.StateIdle,
		Medications: []medication.MedicationDTO{
			{
				Name:   "Medication-A",
				Code:   strings.ToUpper(randomdata.Alphanumeric(32)),
				Weight: 20,
				Image:  env.samplMedicationCaseBase64,
			},
			{
				Name:   "Medication-B",
				Code:   strings.ToUpper(randomdata.Alphanumeric(32)),
				Weight: 40,
				Image:  env.samplMedicationCaseBase64,
			},
			{
				Name:   "Medication-C",
				Code:   strings.ToUpper(randomdata.Alphanumeric(32)),
				Weight: 25,
				Image:  env.samplMedicationCaseBase64,
			},
			{
				Name:   "Medication-D",
				Code:   strings.ToUpper(randomdata.Alphanumeric(32)),
				Weight: 10,
				Image:  env.samplMedicationCaseBase64,
			},
		},
	}
	drone1, err := drone.NewDrone(droneDTO1)
	if err != nil {
		return fmt.Errorf("error while creating preloaded drone with serial number %s:%v", droneDTO1.SerialNumber, err)
	}
	env.registeredDrones[droneDTO1.SerialNumber] = drone1

	droneDTO2 := drone.DroneDTO{
		SerialNumber:    randomdata.Alphanumeric(50),
		Model:           drone.ModelHeavyweight,
		WeightLimit:     500,
		BatteryCapacity: 100,
		State:           drone.StateIdle,
		Medications: []medication.MedicationDTO{
			{
				Name:   "Medication-A",
				Code:   strings.ToUpper(randomdata.Alphanumeric(32)),
				Weight: 200,
				Image:  env.samplMedicationCaseBase64,
			},
			{
				Name:   "Medication-B",
				Code:   strings.ToUpper(randomdata.Alphanumeric(32)),
				Weight: 80,
				Image:  env.samplMedicationCaseBase64,
			},
			{
				Name:   "Medication-C",
				Code:   strings.ToUpper(randomdata.Alphanumeric(32)),
				Weight: 50,
				Image:  env.samplMedicationCaseBase64,
			},
			{
				Name:   "Medication-D",
				Code:   strings.ToUpper(randomdata.Alphanumeric(32)),
				Weight: 60,
				Image:  env.samplMedicationCaseBase64,
			},
		},
	}
	drone2, err := drone.NewDrone(droneDTO2)
	if err != nil {
		return fmt.Errorf("error while creating preloaded drone with serial number %s:%v", droneDTO2.SerialNumber, err)
	}
	env.registeredDrones[droneDTO2.SerialNumber] = drone2

	droneDTO3 := drone.DroneDTO{
		SerialNumber:    randomdata.Alphanumeric(50),
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
		SerialNumber:    randomdata.Alphanumeric(50),
		Model:           drone.ModelCruiserweight,
		WeightLimit:     400,
		BatteryCapacity: 100,
		State:           drone.StateIdle,
		Medications: []medication.MedicationDTO{
			{
				Name:   "Medication-C",
				Code:   strings.ToUpper(randomdata.Alphanumeric(32)),
				Weight: 300,
				Image:  env.samplMedicationCaseBase64,
			},
			{
				Name:   "Medication-D",
				Code:   strings.ToUpper(randomdata.Alphanumeric(32)),
				Weight: 90,
				Image:  env.samplMedicationCaseBase64,
			},
		},
	}
	drone4, err := drone.NewDrone(droneDTO4)
	if err != nil {
		return fmt.Errorf("error while creating preloaded drone with serial number %s:%v", droneDTO4.SerialNumber, err)
	}
	env.registeredDrones[droneDTO4.SerialNumber] = drone4

	droneDTO5 := drone.DroneDTO{
		SerialNumber:    randomdata.Alphanumeric(50),
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

	//env.printDataOfDrones()

	return nil
}

//print data of registered drones in log
func (env *environment) printDataOfDrones(drones []drone.DroneDTO) {

	if drones == nil {
		drones = make([]drone.DroneDTO, 0)
	}

	for _, v := range env.registeredDrones {
		drones = append(drones, v.GetDTO())
	}

	log.Printf("data of the %d registered drones:", len(drones))

	jsonBytes, err := json.MarshalIndent(drones, "", "  ")
	if err != nil {
		log.Printf("%v", drones)
	} else {
		log.Printf("%s", string(jsonBytes))
	}

}

//periodic check on log of battery levels
func (env *environment) checkDronesBatteryLevelsPeriodically() {

	ticker := time.NewTicker(time.Duration(env.Config.LogPeriodMinutes) * time.Minute)
	defer ticker.Stop()
	for {
		select {
		/* 		case <-stop:
		log.Println("external command: periodic check of drones battery levels is stopped, due to restart signal")
		return */
		case <-ticker.C:
			drones := make([]drone.DroneDTO, 0)
			log.Print("check of drones's battery levels:")
			for _, v := range env.registeredDrones {
				drones = append(drones, v.GetDTOWithSerialNumberAndBatteryCapacity())
				//log.Printf("drone serial number: %s has a battery level of %d %%", k, v.GetBatteryCapacity())
			}

			jsonBytes, err := json.MarshalIndent(drones, "", "  ")
			if err != nil {
				log.Printf("list of drones's battery levels could not be marshaled: %v", err)
			} else {
				log.Println(string(jsonBytes))
			}
		}
	}
}

//load of sample of medication case image (convert the image to base64)
func (env *environment) loadSamplMedicationCaseBase64() {
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
	env.samplMedicationCaseBase64 = base64.StdEncoding.EncodeToString(content)
}

//print error responses
func writeError(w http.ResponseWriter, statusCode int, errMessage string) {

	response := Response{
		OK:      false,
		Details: errMessage,
	}

	responseBytes, err := json.Marshal(response)
	if err != nil {
		log.Printf("response object could not be marshaled: %v", err)
	} else {
		errMessage = string(responseBytes)
		log.Println(errMessage)
	}

	w.WriteHeader(statusCode)

	if errMessage == "" {
		_, _ = fmt.Fprintln(w, statusCode, http.StatusText(statusCode))
	} else {
		_, _ = fmt.Fprintln(w, errMessage)
	}
}
