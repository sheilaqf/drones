package drone

import (
	"drones/pkg/medication"
	"errors"
	"fmt"
)

const (
	maxSerialNumberCharacters = 100
	maxWeightLimit            = 500
	maxBatteryCapacity        = 100
)

type (
	//define what is a drone within the system
	Drone struct {
		serialNumber     string // (100 characters max);
		model            string // (Lightweight, Middleweight, Cruiserweight, Heavyweight);
		weightLimit      uint16 // (500gr max);
		batteryCapacity  uint8  // (percentage);
		state            string // (IDLE, LOADING, LOADED, DELIVERING, DELIVERED, RETURNING).
		loadedMedication []medication.Medication
	}

	//define a data transfer object for a drone
	DroneDTO struct {
		SerialNumber     string                  `json:"serial_number"`    // (100 characters max);
		Model            string                  `json:"model"`            // (Lightweight, Middleweight, Cruiserweight, Heavyweight);
		WeightLimit      uint16                  `json:"weight_limit"`     // (500gr max);
		BatteryCapacity  uint8                   `json:"battery_capacity"` // (percentage);
		State            string                  `json:"state"`            // (IDLE, LOADING, LOADED, DELIVERING, DELIVERED, RETURNING).
		LoadedMedication []medication.Medication `json:"loaded_medication"`
	}
)

func NewDrone(dto DroneDTO) (*Drone, error) {

	if !validSerialNumber(dto.SerialNumber) {
		return nil, errors.New(dto.SerialNumber + "is not a valid serial number")
	}

	if !validWeightLimit(dto.WeightLimit) {
		return nil, fmt.Errorf("%d is not a valid weight limit", dto.WeightLimit)
	}

	if !validBatteryCapacity(dto.BatteryCapacity) {
		return nil, fmt.Errorf("%d is not a valid battery capacity", dto.BatteryCapacity)
	}

	return &Drone{
		serialNumber:    dto.SerialNumber,
		model:           dto.Model,
		weightLimit:     dto.WeightLimit,
		batteryCapacity: dto.BatteryCapacity,
		state:           dto.State,
	}, nil
}

func (d *Drone) CurrentWeight() uint16 {

	currentWeight := uint16(0)

	for _, v := range d.loadedMedication {
		currentWeight += uint16(v.GetWeight())
	}

	return currentWeight
}

func (d *Drone) IsAcceptableLoad(medication medication.Medication) bool {
	return medication.GetWeight()+uint(d.CurrentWeight()) <= uint(d.weightLimit)
}

func (d *Drone) LoadNewMedication(medication medication.Medication) error {

	if d.IsAcceptableLoad(medication) {
		d.loadedMedication = append(d.loadedMedication, medication)
		return nil
	}

	return errors.New("the drone must not being loaded with more weight that it can carry")
}

func validSerialNumber(serialNumber string) bool {
	return len(serialNumber) > 0 && len(serialNumber) <= maxSerialNumberCharacters
}

func validWeightLimit(weightLimit uint16) bool {
	return weightLimit <= maxWeightLimit
}

func validBatteryCapacity(batteryCapacity uint8) bool {
	return batteryCapacity > 0 && batteryCapacity <= maxBatteryCapacity
}
