package drone

import (
	"strings"
	"testing"

	"github.com/Pallinder/go-randomdata"

	"drones/pkg/medication"
)

func Test_thereIsLoadingStateAndBatteryLevelUnderPercentage(t *testing.T) {

	if forbiddenBatteryLevelForStateLoading > 0 {
		if !thereIsLoadingStateAndBatteryLevelUnderPercentage(StateLoading, forbiddenBatteryLevelForStateLoading-1) {
			t.Errorf(
				"thereIsLoadingStateAndBatteryLevelUnderPercentage must return true for state: %s and battery level: %d",
				StateLoading, forbiddenBatteryLevelForStateLoading)
		}
	}

	if thereIsLoadingStateAndBatteryLevelUnderPercentage(StateLoading, forbiddenBatteryLevelForStateLoading) {
		t.Errorf(
			"thereIsLoadingStateAndBatteryLevelUnderPercentage must return false for state: %s and battery level: %d",
			StateLoading, forbiddenBatteryLevelForStateLoading)
	}

	if thereIsLoadingStateAndBatteryLevelUnderPercentage(StateIdle, forbiddenBatteryLevelForStateLoading) {
		t.Errorf(
			"thereIsLoadingStateAndBatteryLevelUnderPercentage must return false for state: %s and battery level: %d",
			StateIdle, forbiddenBatteryLevelForStateLoading)
	}

	if thereIsLoadingStateAndBatteryLevelUnderPercentage(StateIdle, 10) {
		t.Errorf(
			"thereIsLoadingStateAndBatteryLevelUnderPercentage must return false for state: %s and battery level: %d",
			StateIdle, 10)
	}
}

func Test_NewDrone(t *testing.T) {

	droneDTO := DroneDTO{
		SerialNumber:    randomdata.Alphanumeric(50),
		Model:           ModelHeavyweight,
		WeightLimit:     500,
		BatteryCapacity: 100,
		State:           StateIdle,
		Medications: []medication.MedicationDTO{
			{
				Name:   "Medication-A",
				Code:   strings.ToUpper(randomdata.Alphanumeric(32)),
				Weight: 20,
			},
			{
				Name:   "Medication-B",
				Code:   strings.ToUpper(randomdata.Alphanumeric(32)),
				Weight: 40,
			},
			{
				Name:   "Medication-C",
				Code:   strings.ToUpper(randomdata.Alphanumeric(32)),
				Weight: 25,
			},
			{
				Name:   "Medication-D",
				Code:   strings.ToUpper(randomdata.Alphanumeric(32)),
				Weight: 10,
			},
		},
	}

	droneObj, err := NewDrone(droneDTO)
	if err != nil {
		t.Errorf("error while creating drone for test:%v", err)
	}

	if droneDTO.SerialNumber != droneObj.GetSerialNumber() {
		t.Errorf("serial numbers of droneDTO and droneObj must be the same but droneDTO's serial number was %s and droneObj's serial number was %s", droneDTO.SerialNumber, droneObj.GetSerialNumber())
	}

	if droneDTO.Model != droneObj.GetModel() {
		t.Errorf("models of droneDTO and droneObj must be the same but droneDTO's model was %s and droneObj's model was %s", droneDTO.Model, droneObj.GetModel())
	}

	if droneDTO.Medications[0].Code != droneObj.GetDTO().Medications[0].Code {
		t.Errorf("code of first medications of droneDTO and droneObj must be the same but code of first medication on droneDTO's was %s and code of first medication on droneObj's was %s", droneDTO.Medications[0].Code, droneObj.GetDTO().Medications[0].Code)
	}

	droneDTOB := DroneDTO{
		SerialNumber:    randomdata.Alphanumeric(50) + "B",
		Model:           ModelLightweight,
		WeightLimit:     100,
		BatteryCapacity: 100,
		State:           StateIdle,
		Medications: []medication.MedicationDTO{
			{
				Name:   "Medication-B",
				Code:   "CODE_B",
				Weight: 40,
			},
		},
	}

	if droneDTO.SerialNumber == droneDTOB.SerialNumber {
		t.Errorf("serial numbers of droneDTO and droneDTOB must not be the same but droneDTO's serial number was %s and droneDTOB's serial number was %s", droneDTO.SerialNumber, droneDTOB.SerialNumber)
	}

	if droneDTO.Model == droneDTOB.Model {
		t.Errorf("models of droneDTO and droneDTOB must not be the same but droneDTO's model was %s and droneDTOB's model was %s", droneDTO.Model, droneDTOB.Model)
	}

	if droneDTO.Medications[0].Code == droneDTOB.Medications[0].Code {
		t.Errorf("code of first medications of droneDTO and droneDTOB must not be the same but code of first medication on droneDTO's was %s and code of first medication on droneDTOB's was %s", droneDTO.Medications[0].Code, droneDTOB.Medications[0].Code)
	}
}
