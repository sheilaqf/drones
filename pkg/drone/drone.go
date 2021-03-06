// Implements routines for manipulating drone objects.
package drone

import (
	"drones/pkg/medication"
	"fmt"
	"sync"

	"github.com/pkg/errors"
)

const (
	maxSerialNumberCharacters = 100
	maxWeightLimit            = 500
	maxBatteryCapacity        = 100
	//allowed models
	ModelLightweight   = "Lightweight"
	ModelMiddleweight  = "Middleweight"
	ModelCruiserweight = "Cruiserweight"
	ModelHeavyweight   = "Heavyweight"
	//allowed states
	StateIdle       = "IDLE"
	StateLoading    = "LOADING"
	StateLoaded     = "LOADED"
	StateDelivering = "DELIVERING"
	StateDelivered  = "DELIVERED"
	StateReturning  = "RETURNING"

	forbiddenBatteryLevelForStateLoading = 25
)

type (
	//define what is a drone within the system
	Drone struct {
		serialNumber    string // (100 characters max);
		model           string // (Lightweight, Middleweight, Cruiserweight, Heavyweight);
		weightLimit     uint16 // (500gr max);
		batteryCapacity uint8  // (percentage);
		state           string // (IDLE, LOADING, LOADED, DELIVERING, DELIVERED, RETURNING).
		medications     []medication.Medication
		sync.Mutex
	}

	//define a data transfer object for a drone
	DroneDTO struct {
		SerialNumber    string                     `json:"serial_number"`              // (100 characters max);
		Model           string                     `json:"model,omitempty"`            // (Lightweight, Middleweight, Cruiserweight, Heavyweight);
		WeightLimit     uint16                     `json:"weight_limit,omitempty"`     // (500gr max);
		BatteryCapacity uint8                      `json:"battery_capacity,omitempty"` // (percentage);
		State           string                     `json:"state,omitempty"`            // (IDLE, LOADING, LOADED, DELIVERING, DELIVERED, RETURNING).
		Medications     []medication.MedicationDTO `json:"medications,omitempty"`
	}
)

//get a pointer to a drone object from a drone DTO
func NewDrone(dto DroneDTO) (*Drone, error) {

	if !validSerialNumber(dto.SerialNumber) {
		return nil, errors.New(dto.SerialNumber + "is not a valid serial number")
	}

	if !validModel(dto.Model) {
		return nil, errors.New(dto.Model + "is not a valid model")
	}

	if !validWeightLimit(dto.WeightLimit) {
		return nil, fmt.Errorf("%d is not a valid weight limit", dto.WeightLimit)
	}

	if !validBatteryCapacity(dto.BatteryCapacity) {
		return nil, fmt.Errorf("%d is not a valid battery capacity", dto.BatteryCapacity)
	}

	if !validState(dto.State) {
		return nil, errors.New(dto.State + "is not a valid state")
	}

	if thereIsLoadingStateAndBatteryLevelUnderPercentage(dto.State, dto.BatteryCapacity) {
		return nil, fmt.Errorf("drone should not be %s when the battery level is below %d %%", StateLoading, forbiddenBatteryLevelForStateLoading)
	}

	drone := &Drone{
		serialNumber:    dto.SerialNumber,
		model:           dto.Model,
		weightLimit:     dto.WeightLimit,
		batteryCapacity: dto.BatteryCapacity,
		state:           dto.State,
	}

	err := drone.LoadSetOfMedications(dto.Medications)
	if err != nil {
		return drone, err
	}

	return drone, nil
}

//get the total weight of all medications on the drone
func (d *Drone) CurrentWeight() uint16 {

	currentWeight := uint16(0)

	for _, v := range d.medications {
		currentWeight += uint16(v.GetWeight())
	}

	return currentWeight
}

//check whether the addition of a new medication does not exceed the maximun load capacity of the drone
func (d *Drone) IsAcceptableLoad(medication medication.Medication) bool {
	return medication.GetWeight()+uint(d.CurrentWeight()) <= uint(d.weightLimit)
}

//load a new medication on the drone
func (d *Drone) LoadNewMedication(medication medication.Medication) error {

	d.Lock()
	defer d.Unlock()

	if d.batteryCapacity < forbiddenBatteryLevelForStateLoading {
		return fmt.Errorf("drone should not be %s when the battery level is below %d %%", StateLoading, forbiddenBatteryLevelForStateLoading)
	}

	if d.IsAcceptableLoad(medication) {
		d.state = StateLoading
		d.medications = append(d.medications, medication)
		d.state = StateLoaded
		return nil
	}

	return errors.New("the drone must not being loaded with more weight that it can carry")
}

//load new medications on the drone
func (d *Drone) LoadNewMedications(medications []medication.Medication) error {

	successfullyLoaded := 0
	for _, v := range medications {
		err := d.LoadNewMedication(v)
		if err != nil {
			return errors.Wrapf(err, "successfully loaded medications: %d of %d", successfullyLoaded, len(medications))
		}
		successfullyLoaded++
	}

	return nil
}

//load new medications on the drone (using a list of DTOs)
func (d *Drone) LoadSetOfMedications(medications []medication.MedicationDTO) error {

	successfullyLoaded := 0
	for _, v := range medications {
		medication, err := medication.NewMedication(v)
		if err != nil {
			return errors.Wrapf(err, "successfully loaded medications: %d of %d", successfullyLoaded, len(medications))
		}
		err = d.LoadNewMedication(*medication)
		if err != nil {
			return errors.Wrapf(err, "successfully loaded medications: %d of %d", successfullyLoaded, len(medications))
		}
		successfullyLoaded++
	}

	return nil
}

//get DTO that represents a drone
func (d *Drone) GetDTO() DroneDTO {

	dto := DroneDTO{
		SerialNumber:    d.serialNumber,
		Model:           d.model,
		WeightLimit:     d.weightLimit,
		BatteryCapacity: d.batteryCapacity,
		State:           d.state,
		Medications:     make([]medication.MedicationDTO, 0),
	}

	for _, v := range d.medications {
		dto.Medications = append(dto.Medications, v.GetDTO())
	}

	return dto
}

//get serial number of drone
func (d *Drone) GetSerialNumber() string {
	return d.serialNumber
}

//get model of drone
func (d *Drone) GetModel() string {
	return d.model
}

//get state of drone
func (d *Drone) GetState() string {
	return d.state
}

//get battery capacityo of drone
func (d *Drone) GetBatteryCapacity() uint8 {
	return d.batteryCapacity
}

//get drone DTO with only the information of the serial number
func (d *Drone) GetDTOWithSerialNumber() DroneDTO {

	dto := DroneDTO{
		SerialNumber: d.serialNumber,
	}

	return dto
}

//get drone DTO with only the information of the serial number and battery capacity
func (d *Drone) GetDTOWithSerialNumberAndBatteryCapacity() DroneDTO {

	dto := DroneDTO{
		SerialNumber:    d.serialNumber,
		BatteryCapacity: d.batteryCapacity,
	}

	return dto
}

//get drone DTO with only the information of the serial number and medications (without medication's image)
func (d *Drone) GetDTOWithSerialNumberAndMedications() DroneDTO {

	dto := DroneDTO{
		SerialNumber: d.serialNumber,
		Medications:  make([]medication.MedicationDTO, 0),
	}

	for _, v := range d.medications {
		dto.Medications = append(dto.Medications, v.GetDTO())
	}

	return dto
}

//check whether the drone has loaded medications
func (d *Drone) HasMedications() bool {
	return len(d.medications) > 0
}

//check whether a drone is available for loading
func (d *Drone) IsAvailableForLoading() bool {
	return d.state == StateIdle && d.batteryCapacity >= forbiddenBatteryLevelForStateLoading
}

//check whether a serial number of drone is valid
func validSerialNumber(serialNumber string) bool {
	return len(serialNumber) > 0 && len(serialNumber) <= maxSerialNumberCharacters
}

//check whether a weight limit of drone is valid
func validWeightLimit(weightLimit uint16) bool {
	return weightLimit <= maxWeightLimit
}

//check whether a battery capacity of drone is valid
func validBatteryCapacity(batteryCapacity uint8) bool {
	return batteryCapacity > 0 && batteryCapacity <= maxBatteryCapacity
}

//check whether a model of drone is valid
func validModel(model string) bool {

	switch model {
	case ModelLightweight, ModelMiddleweight, ModelCruiserweight, ModelHeavyweight:
		return true
	}

	return false
}

//check whether a state of drone is valid
func validState(state string) bool {

	switch state {
	case StateIdle, StateLoading, StateLoaded, StateDelivering, StateDelivered, StateReturning:
		return true
	}

	return false
}

//prevent the drone from being in LOADING state if the battery level is **below 25%**
func thereIsLoadingStateAndBatteryLevelUnderPercentage(batteryLevel string, percentage uint8) bool {
	return batteryLevel == StateLoading && percentage < forbiddenBatteryLevelForStateLoading
}
