package drone

import (
	"testing"
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
