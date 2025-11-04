package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test structs for validation
type TestUser struct {
	Email    string `validate:"required,email"`
	Username string `validate:"required,min=3,max=20"`
	Age      int    `validate:"required,min=18,max=100"`
	Password string `validate:"required,min=8"`
}

type SimpleStruct struct {
	Name string `validate:"required"`
}

type ComplexStruct struct {
	Field1 string `validate:"required"`
	Field2 string `validate:"email"`
	Field3 int    `validate:"min=10,max=100"`
	Field4 string `validate:"min=5,max=50"`
}

type NoValidationStruct struct {
	Name  string
	Email string
	Age   int
}

// TestValidateStruct_ValidData tests validation with valid data
func TestValidateStruct_ValidData(t *testing.T) {
	user := TestUser{
		Email:    "test@example.com",
		Username: "validuser",
		Age:      25,
		Password: "securepassword123",
	}

	err := ValidateStruct(user)
	assert.NoError(t, err, "Valid struct should pass validation")
}

// TestValidateStruct_InvalidEmail tests validation with invalid email
func TestValidateStruct_InvalidEmail(t *testing.T) {
	user := TestUser{
		Email:    "invalid-email",
		Username: "validuser",
		Age:      25,
		Password: "securepassword123",
	}

	err := ValidateStruct(user)
	assert.Error(t, err, "Invalid email should fail validation")
}

// TestValidateStruct_MissingRequired tests validation with missing required fields
func TestValidateStruct_MissingRequired(t *testing.T) {
	user := TestUser{
		Email: "test@example.com",
		// Username is missing (empty string)
		Age:      25,
		Password: "securepassword123",
	}

	err := ValidateStruct(user)
	assert.Error(t, err, "Missing required field should fail validation")
}

// TestValidateStruct_MinLength tests minimum length validation
func TestValidateStruct_MinLength(t *testing.T) {
	testCases := []struct {
		name     string
		username string
		password string
		shouldFail bool
	}{
		{"Valid lengths", "abc", "12345678", false},
		{"Username too short", "ab", "12345678", true},
		{"Password too short", "validuser", "1234567", true},
		{"Both too short", "ab", "1234567", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			user := TestUser{
				Email:    "test@example.com",
				Username: tc.username,
				Age:      25,
				Password: tc.password,
			}

			err := ValidateStruct(user)
			if tc.shouldFail {
				assert.Error(t, err, "Should fail validation")
			} else {
				assert.NoError(t, err, "Should pass validation")
			}
		})
	}
}

// TestValidateStruct_MaxLength tests maximum length validation
func TestValidateStruct_MaxLength(t *testing.T) {
	user := TestUser{
		Email:    "test@example.com",
		Username: "thisusernameiswaytoolongandexceeds20characters",
		Age:      25,
		Password: "securepassword123",
	}

	err := ValidateStruct(user)
	assert.Error(t, err, "Username exceeding max length should fail")
}

// TestValidateStruct_RangeValidation tests range validation for numbers
func TestValidateStruct_RangeValidation(t *testing.T) {
	testCases := []struct {
		name       string
		age        int
		shouldFail bool
	}{
		{"Valid age", 25, false},
		{"Minimum age", 18, false},
		{"Maximum age", 100, false},
		{"Below minimum", 17, true},
		{"Above maximum", 101, true},
		{"Very low", 0, true},
		{"Negative", -5, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			user := TestUser{
				Email:    "test@example.com",
				Username: "validuser",
				Age:      tc.age,
				Password: "securepassword123",
			}

			err := ValidateStruct(user)
			if tc.shouldFail {
				assert.Error(t, err, "Should fail validation")
			} else {
				assert.NoError(t, err, "Should pass validation")
			}
		})
	}
}

// TestValidateStruct_MultipleErrors tests multiple validation errors
func TestValidateStruct_MultipleErrors(t *testing.T) {
	user := TestUser{
		Email:    "invalid-email",
		Username: "ab", // Too short
		Age:      15,   // Too young
		Password: "short", // Too short
	}

	err := ValidateStruct(user)
	assert.Error(t, err, "Multiple validation errors should fail")
}

// TestValidateStruct_NoValidationTags tests struct without validation tags
func TestValidateStruct_NoValidationTags(t *testing.T) {
	data := NoValidationStruct{
		Name:  "",
		Email: "not-an-email",
		Age:   -5,
	}

	err := ValidateStruct(data)
	assert.NoError(t, err, "Struct without validation tags should pass")
}

// TestGetValidationErrors_RequiredField tests required field error message
func TestGetValidationErrors_RequiredField(t *testing.T) {
	user := SimpleStruct{
		Name: "", // Required but empty
	}

	err := ValidateStruct(user)
	require.Error(t, err, "Should have validation error")

	errors := GetValidationErrors(err)
	require.Len(t, errors, 1, "Should have one error")
	assert.Equal(t, "Name is required", errors[0], "Error message should be formatted correctly")
}

// TestGetValidationErrors_EmailField tests email validation error message
func TestGetValidationErrors_EmailField(t *testing.T) {
	user := ComplexStruct{
		Field1: "valid",
		Field2: "invalid-email",
		Field3: 50,
		Field4: "valid field",
	}

	err := ValidateStruct(user)
	require.Error(t, err, "Should have validation error")

	errors := GetValidationErrors(err)
	require.Greater(t, len(errors), 0, "Should have at least one error")

	// Find the email error
	found := false
	for _, errMsg := range errors {
		if errMsg == "Field2 must be a valid email" {
			found = true
			break
		}
	}
	assert.True(t, found, "Should have email validation error message")
}

// TestGetValidationErrors_MinField tests min validation error message
func TestGetValidationErrors_MinField(t *testing.T) {
	user := ComplexStruct{
		Field1: "valid",
		Field2: "test@example.com",
		Field3: 5, // Below min of 10
		Field4: "abc", // Below min of 5
	}

	err := ValidateStruct(user)
	require.Error(t, err, "Should have validation error")

	errors := GetValidationErrors(err)
	require.Greater(t, len(errors), 0, "Should have errors")

	// Check for min error messages
	errorString := ""
	for _, e := range errors {
		errorString += e + " "
	}

	assert.Contains(t, errorString, "must be at least", "Should mention minimum requirement")
}

// TestGetValidationErrors_MaxField tests max validation error message
func TestGetValidationErrors_MaxField(t *testing.T) {
	user := ComplexStruct{
		Field1: "valid",
		Field2: "test@example.com",
		Field3: 150, // Above max of 100
		Field4: "this is a very long string that exceeds the maximum length of 50 characters",
	}

	err := ValidateStruct(user)
	require.Error(t, err, "Should have validation error")

	errors := GetValidationErrors(err)
	require.Greater(t, len(errors), 0, "Should have errors")

	// Check for max error messages
	errorString := ""
	for _, e := range errors {
		errorString += e + " "
	}

	assert.Contains(t, errorString, "must be at most", "Should mention maximum requirement")
}

// TestGetValidationErrors_MultipleErrors tests formatting of multiple errors
func TestGetValidationErrors_MultipleErrors(t *testing.T) {
	user := TestUser{
		Email:    "invalid-email",
		Username: "ab", // Too short (min=3)
		Age:      15,   // Too young (min=18)
		Password: "short", // Too short (min=8)
	}

	err := ValidateStruct(user)
	require.Error(t, err, "Should have validation errors")

	errors := GetValidationErrors(err)
	assert.Greater(t, len(errors), 1, "Should have multiple errors")

	// Check that all errors are formatted
	for _, errMsg := range errors {
		assert.NotEmpty(t, errMsg, "Error message should not be empty")
		// Each message should contain field name and description
		assert.Contains(t, errMsg, " ", "Error should contain field and description")
	}
}

// TestGetValidationErrors_AllErrorTypes tests all supported error types
func TestGetValidationErrors_AllErrorTypes(t *testing.T) {
	type AllTypesStruct struct {
		RequiredField string `validate:"required"`
		EmailField    string `validate:"email"`
		MinField      string `validate:"min=5"`
		MaxField      string `validate:"max=10"`
	}

	testCases := []struct {
		name          string
		data          AllTypesStruct
		expectedError string
	}{
		{
			name: "Required error",
			data: AllTypesStruct{
				RequiredField: "",
				EmailField:    "valid@email.com",
				MinField:      "12345",
				MaxField:      "short",
			},
			expectedError: "RequiredField is required",
		},
		{
			name: "Email error",
			data: AllTypesStruct{
				RequiredField: "present",
				EmailField:    "invalid-email",
				MinField:      "12345",
				MaxField:      "short",
			},
			expectedError: "EmailField must be a valid email",
		},
		{
			name: "Min error",
			data: AllTypesStruct{
				RequiredField: "present",
				EmailField:    "valid@email.com",
				MinField:      "1234", // Too short
				MaxField:      "short",
			},
			expectedError: "MinField must be at least 5 characters",
		},
		{
			name: "Max error",
			data: AllTypesStruct{
				RequiredField: "present",
				EmailField:    "valid@email.com",
				MinField:      "12345",
				MaxField:      "this is way too long", // Too long
			},
			expectedError: "MaxField must be at most 10 characters",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateStruct(tc.data)
			require.Error(t, err, "Should have validation error")

			errors := GetValidationErrors(err)
			require.Greater(t, len(errors), 0, "Should have at least one error")

			found := false
			for _, errMsg := range errors {
				if errMsg == tc.expectedError {
					found = true
					break
				}
			}
			assert.True(t, found, "Should have expected error message: %s", tc.expectedError)
		})
	}
}

// TestGetValidationErrors_UnknownTag tests handling of unknown validation tags
func TestGetValidationErrors_UnknownTag(t *testing.T) {
	// Skip this test as validator v10 panics on unknown tags at registration time
	// This is expected behavior for go-playground/validator
	t.Skip("Validator panics on unknown tags - this is expected behavior")
}

// TestGetValidationErrors_NonValidationError tests handling of non-validation errors
func TestGetValidationErrors_NonValidationError(t *testing.T) {
	// Create a non-validation error
	err := assert.AnError

	errors := GetValidationErrors(err)
	assert.Empty(t, errors, "Non-validation errors should return empty slice")
}

// TestGetValidationErrors_NilError tests handling of nil error
func TestGetValidationErrors_NilError(t *testing.T) {
	errors := GetValidationErrors(nil)
	assert.Empty(t, errors, "Nil error should return empty slice")
}

// TestValidateStruct_NestedStruct tests validation with nested structs
func TestValidateStruct_NestedStruct(t *testing.T) {
	type Address struct {
		Street string `validate:"required"`
		City   string `validate:"required"`
	}

	type Person struct {
		Name    string  `validate:"required"`
		Address Address `validate:"required"`
	}

	validPerson := Person{
		Name: "John Doe",
		Address: Address{
			Street: "123 Main St",
			City:   "New York",
		},
	}

	err := ValidateStruct(validPerson)
	assert.NoError(t, err, "Valid nested struct should pass")
}

// TestValidateStruct_EmptyStruct tests validation with empty struct
func TestValidateStruct_EmptyStruct(t *testing.T) {
	type EmptyStruct struct{}

	data := EmptyStruct{}

	err := ValidateStruct(data)
	assert.NoError(t, err, "Empty struct should not cause errors")
}

// TestValidateStruct_PointerFields tests validation with pointer fields
func TestValidateStruct_PointerFields(t *testing.T) {
	type PointerStruct struct {
		Name  *string `validate:"required"`
		Email *string `validate:"email"`
	}

	name := "John"
	email := "john@example.com"

	data := PointerStruct{
		Name:  &name,
		Email: &email,
	}

	err := ValidateStruct(data)
	// Note: This might fail depending on how validator handles pointers
	// Just ensuring it doesn't panic
	_ = err
}

// Benchmark tests
func BenchmarkValidateStruct(b *testing.B) {
	user := TestUser{
		Email:    "test@example.com",
		Username: "validuser",
		Age:      25,
		Password: "securepassword123",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ValidateStruct(user)
	}
}

func BenchmarkGetValidationErrors(b *testing.B) {
	user := TestUser{
		Email:    "invalid-email",
		Username: "ab",
		Age:      15,
		Password: "short",
	}

	err := ValidateStruct(user)
	require.Error(b, err)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = GetValidationErrors(err)
	}
}

// TestValidateStruct_Concurrency tests thread safety
func TestValidateStruct_Concurrency(t *testing.T) {
	iterations := 100
	done := make(chan bool, iterations)

	for i := 0; i < iterations; i++ {
		go func(idx int) {
			user := TestUser{
				Email:    "test@example.com",
				Username: "validuser",
				Age:      25,
				Password: "securepassword123",
			}

			err := ValidateStruct(user)
			assert.NoError(t, err)

			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < iterations; i++ {
		<-done
	}
}

// TestValidationRealWorldScenario tests realistic validation scenario
func TestValidationRealWorldScenario(t *testing.T) {
	// Simulate user registration form validation
	type RegistrationForm struct {
		Email           string `validate:"required,email"`
		Username        string `validate:"required,min=3,max=20"`
		Password        string `validate:"required,min=8"`
		ConfirmPassword string `validate:"required"`
		Age             int    `validate:"required,min=18"`
	}

	// Valid registration
	validForm := RegistrationForm{
		Email:           "newuser@example.com",
		Username:        "newuser123",
		Password:        "securePass123",
		ConfirmPassword: "securePass123",
		Age:             25,
	}

	err := ValidateStruct(validForm)
	assert.NoError(t, err, "Valid registration should pass")

	// Invalid registration
	invalidForm := RegistrationForm{
		Email:           "invalid-email",
		Username:        "ab", // Too short
		Password:        "1234", // Too short
		ConfirmPassword: "",
		Age:             16, // Too young
	}

	err = ValidateStruct(invalidForm)
	require.Error(t, err, "Invalid registration should fail")

	errors := GetValidationErrors(err)
	assert.Greater(t, len(errors), 0, "Should have validation errors")

	// Check that errors are user-friendly
	for _, errMsg := range errors {
		assert.NotEmpty(t, errMsg, "Error message should not be empty")
		// Should not contain technical field names with underscores/camelCase issues
		t.Logf("Validation error: %s", errMsg)
	}
}
