package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"

	"drones/pkg/config"
	"drones/pkg/drone"
)

type (
	environment struct {
		HttpServer       *http.Server
		Router           *mux.Router
		Config           *config.Config
		registeredDrones map[string]*drone.Drone // use SerialNumber as key
	}
)

func main() {

	log.Println("Initializing Drones Management API.")
	configFlag := flag.String("config", "config.json", "path to config json file")
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

	startServer(cfg)
}

func startServer(cfg *config.Config) {

	env := environment{
		Config: cfg,
	}

	env.Router = mux.NewRouter()

	env.Router.HandleFunc("/drone/register", env.RegisterDrone).Methods("POST")
	/*	env.Router.HandleFunc("/user/create", env.CreateUser).Methods("POST")
		env.Router.HandleFunc("/user/update", env.UpdateUser).Methods("POST")
		env.Router.HandleFunc("/user/delete", env.DeleteUser).Methods("POST")
		env.Router.HandleFunc("/user/auth", env.AuthenticateUser).Methods("POST")
		env.Router.HandleFunc("/user/out", env.LogOutUser).Methods("GET")
		env.Router.HandleFunc("/user/password/update", env.UpdatePassword).Methods("POST")
		env.Router.HandleFunc("/user/password/reset", env.ResetPassword).Methods("POST") */

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

func (env *environment) RegisterDrone(w http.ResponseWriter, r *http.Request) {
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

	env.addNewDrone(droneObj)
}

func (env *environment) addNewDrone(droneObj *drone.Drone) {

	if env.registeredDrones == nil {
		env.registeredDrones = make(map[string]*drone.Drone)
	}

	env.registeredDrones[droneObj.GetSerialNumber()] = droneObj
}

func (env *environment) setLoadForDrone(load drone.DroneDTO) error {

	if env.registeredDrones[load.SerialNumber] == nil {
		return fmt.Errorf("there is not a drone with serial number %s", load.SerialNumber)
	}

	return nil
}

func writeError(w http.ResponseWriter, statusCode int) {
	w.WriteHeader(statusCode)
	_, _ = fmt.Fprintln(w, statusCode, http.StatusText(statusCode))
}
