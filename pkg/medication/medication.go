package medication

import (
	"errors"
	"regexp"
)

type (
	//define what is a medication within the system
	Medication struct {
		name   string // (allowed only letters, numbers, ‘-‘, ‘_’);
		weight uint   //
		code   string // (allowed only upper case letters, underscore and numbers);
		image  string // (picture of the medication case). // *base64
	}

	//define a data transfer object for a medication
	MedicationDTO struct {
		Name   string `json:"name"`   // (allowed only letters, numbers, ‘-‘, ‘_’);
		Weight uint   `json:"weight"` //
		Code   string `json:"code"`   // (allowed only upper case letters, underscore and numbers);
		Image  string `json:"image"`  // (picture of the medication case). // *base64
	}
)

func NewMedication(dto MedicationDTO) (*Medication, error) {

	//TODO: Check conditions here
	if !isValidName(dto.Name) {
		return nil, errors.New(dto.Name + "is not a valid name")
	}

	if !isValidCode(dto.Code) {
		return nil, errors.New(dto.Code + "is not a valid code")
	}

	return &Medication{
		name:   dto.Name,
		weight: dto.Weight,
		code:   dto.Code,
		image:  dto.Image,
	}, nil
}

func (m *Medication) GetWeight() uint {
	return m.weight
}

func (m *Medication) GetDTO() MedicationDTO {
	return MedicationDTO{
		Name:   m.name,
		Weight: m.weight,
		Code:   m.code,
		Image:  m.image,
	}
}

func isValidName(name string) bool {
	match, err := regexp.MatchString("^[A-Za-z0-9?_-]+$", name)
	if err != nil {
		return false
	}
	return match
}

func isValidCode(name string) bool {
	match, err := regexp.MatchString("^[A-Z0-9?_]+$", name)
	if err != nil {
		return false
	}
	return match
}
