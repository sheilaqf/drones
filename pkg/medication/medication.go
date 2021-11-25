// Implements routines for manipulating medication objects.
package medication

import (
	"regexp"

	"github.com/pkg/errors"
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

//get a pointer to a medication object from a medication DTO
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

//get a list medication object from a list of medication DTOs
func NewMedications(dtos []MedicationDTO) ([]Medication, error) {

	medications := make([]Medication, 0)
	for _, v := range dtos {
		medication, err := NewMedication(v)
		if err != nil {
			return medications, errors.Wrapf(err, "successfully loaded medications: %d of %d", len(medications), len(dtos))
		}
		medications = append(medications, *medication)
	}

	return medications, nil
}

//get weight of medication
func (m *Medication) GetWeight() uint {
	return m.weight
}

//get DTO of medication object excluding the image information in base64
func (m *Medication) GetDTO() MedicationDTO {
	return MedicationDTO{
		Name:   m.name,
		Weight: m.weight,
		Code:   m.code,
	}
}

//get DTO of medication object including the image information in base64
func (m *Medication) GetDTOWithImage() MedicationDTO {
	return MedicationDTO{
		Name:   m.name,
		Weight: m.weight,
		Code:   m.code,
		Image:  m.image,
	}
}

//check whether a name of medication is valid
func isValidName(name string) bool {
	match, err := regexp.MatchString("^[A-Za-z0-9?_-]+$", name)
	if err != nil {
		return false
	}
	return match
}

//check whether a code of medication is valid
func isValidCode(name string) bool {
	match, err := regexp.MatchString("^[A-Z0-9?_]+$", name)
	if err != nil {
		return false
	}
	return match
}
