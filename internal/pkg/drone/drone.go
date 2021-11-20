package drone

type (
	//define what is a drone
	Drone struct {
		SerialNumber    string `json:"serial_number"`    // (100 characters max);
		Model           string `json:"model"`            // (Lightweight, Middleweight, Cruiserweight, Heavyweight);
		WeightLimit     uint16 `json:"weight_limit"`     // (500gr max);
		BatteryCapacity uint8  `json:"battery_capacity"` // (percentage);
		State           string `json:"state"`            // (IDLE, LOADING, LOADED, DELIVERING, DELIVERED, RETURNING).
	}
)
