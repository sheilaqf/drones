package drone

import (
	"drones/pkg/medication"
	"errors"
)

type (
	//define what is a drone
	Drone struct {
		SerialNumber     string                  `json:"serial_number"`    // (100 characters max);
		Model            string                  `json:"model"`            // (Lightweight, Middleweight, Cruiserweight, Heavyweight);
		WeightLimit      uint16                  `json:"weight_limit"`     // (500gr max);
		BatteryCapacity  uint8                   `json:"battery_capacity"` // (percentage);
		State            string                  `json:"state"`            // (IDLE, LOADING, LOADED, DELIVERING, DELIVERED, RETURNING).
		LoadedMedication []medication.Medication `json:"loaded_medication"`
	}
)

func (d *Drone) CurrentWeight() uint16 {

	currentWeight := uint16(0)

	for _, v := range d.LoadedMedication {
		currentWeight += uint16(v.Weight)
	}

	return currentWeight
}

func (d *Drone) IsAcceptableLoad(medication medication.Medication) bool {
	return medication.Weight+uint(d.CurrentWeight()) <= uint(d.WeightLimit)
}

func (d *Drone) LoadNewMedication(medication medication.Medication) error {

	if d.IsAcceptableLoad(medication) {
		d.LoadedMedication = append(d.LoadedMedication, medication)
		return nil
	}

	return errors.New("the drone must not being loaded with more weight that it can carry")
}
