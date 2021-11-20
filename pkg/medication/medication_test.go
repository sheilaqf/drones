package medication

import (
	"testing"
)

func Test_isValidName(t *testing.T) {

	validStrings := []string{
		"ThisAValidName",
		"ThisAValidName0",
		"ThisAValidName-0",
		"123",
		"0-A-B-C_D_872_D_This_must_be_valid",
		"This-A-Valid-Name",
		"This_A_Valid_Name",
	}

	for _, v := range validStrings {
		if !isValidName(v) {
			t.Errorf("%s must be a valid name and was asserted as a wrong one", v)
		}
	}

	noValidStrings := []string{
		"This,Is,Not,A,Valid,Name",
		"0-A-B-C_+D_872_D_This_must_not_be_valid",
		"0-A-B-C_D_872_D_This_must_be_not&valid",
	}

	for _, v := range noValidStrings {
		if isValidName(v) {
			t.Errorf("%s must be a no valid name and was asserted as a wrong one", v)
		}
	}
}

func Test_isValidCode(t *testing.T) {

	validStrings := []string{
		"THIS_IS_A_VALID_CODE",
		"THIS_IS_A_VALID_CODE_T00",
		"VALID",
		"VALID_",
		"0123456789",
		"ABCDE_12345",
		"_1Q",
	}

	for _, v := range validStrings {
		if !isValidCode(v) {
			t.Errorf("%s must be a valid code and was asserted as a wrong one", v)
		}
	}

	noValidStrings := []string{
		"ThisIsNotAValidCode",
		"THIS_WAS_A_VALID_CODE_until_it_reached_this_part_of_the_string",
		"NOT-A-VALID_CODE",
		"ThisIsNotAValidCode",
		"This,Is,Not,A,Valid,Code",
		"0-A-B-C_+D_872_D_This_must_not_be_valid",
		"0-A-B-C_D_872_D_This_must_be_not&valid",
	}

	for _, v := range noValidStrings {
		if isValidCode(v) {
			t.Errorf("%s must be a no valid name and was asserted as a wrong one", v)
		}
	}
}
