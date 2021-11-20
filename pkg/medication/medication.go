package medication

type (
	//define what is a medication
	Medication struct {
		Name   string `json:"name"`   // (allowed only letters, numbers, ‘-‘, ‘_’);
		Weight uint   `json:"weight"` //
		Code   string `json:"code"`   // (allowed only upper case letters, underscore and numbers);
		Image  string `json:"image"`  // (picture of the medication case). // *base64
	}
)
