package config

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test the Empty Config case
func emptyConfig() *Config {
	return &Config{ResourceType{FilterRule{}, FilterRule{}}, ResourceType{FilterRule{}, FilterRule{}}}
}

// Test that garbage in the config file parses to an empty Config object.
func TestConfig_Garbage(t *testing.T) {
	configFilePath := "./mocks/garbage.yaml"
	configObj, err := GetConfig(configFilePath)

	require.NoError(t, err)

	if !reflect.DeepEqual(configObj, emptyConfig()) {
		assert.Fail(t, "Config should be empty, %+v\n", configObj)
	}

	return
}

// Test that a malformed config file does not get parsed.
func TestConfig_Malformed(t *testing.T) {
	configFilePath := "./mocks/malformed.yaml"
	_, err := GetConfig(configFilePath)

	// Expect malformed to throw a yaml TypeError
	require.Error(t, err, "Received expected error")
	return
}

// Test that an empty config file is valid but creates an empty Config object.
func TestConfig_Empty(t *testing.T) {
	configFilePath := "./mocks/empty.yaml"
	configObj, err := GetConfig(configFilePath)

	require.NoError(t, err)

	if !reflect.DeepEqual(configObj, emptyConfig()) {
		assert.Fail(t, "Config should be empty, %+v\n", configObj)
	}

	return
}

// Test that a resource config with no other sub-keys creates an empty Config object.
func TestConfigS3_Empty(t *testing.T) {
	configFilePath := "./mocks/s3_empty.yaml"
	configObj, err := GetConfig(configFilePath)

	require.NoError(t, err)

	if !reflect.DeepEqual(configObj, emptyConfig()) {
		assert.Fail(t, "Config should be empty, %+v\n", configObj.S3)
	}

	return
}

// Test that a resource config with filters keys, but no rules, creates an empty Config object.
func TestConfigS3_EmptyFilters(t *testing.T) {
	configFilePath := "./mocks/s3_empty_filters.yaml"
	configObj, err := GetConfig(configFilePath)

	require.NoError(t, err)

	if !reflect.DeepEqual(configObj, emptyConfig()) {
		assert.Fail(t, "Config should be empty, %+v\n", configObj)
	}

	return
}

// Test that a resource config with filters and rules keys, but no values in them, creates an empty Config object.
func TestConfigS3_EmptyRules(t *testing.T) {
	configFilePath := "./mocks/s3_empty_rules.yaml"
	configObj, err := GetConfig(configFilePath)

	require.NoError(t, err)

	if !reflect.DeepEqual(configObj, emptyConfig()) {
		assert.Fail(t, "Config should be empty, %+v\n", configObj)
	}

	return
}

// Test that rules get parsed for a resource config with include rules.
func TestConfigS3_IncludeNames(t *testing.T) {
	configFilePath := "./mocks/s3_include_names.yaml"
	configObj, err := GetConfig(configFilePath)

	require.NoError(t, err)

	if reflect.DeepEqual(configObj, emptyConfig()) {
		assert.Fail(t, "Config should not be empty, %+v\n", configObj)
	}

	if len(configObj.S3.IncludeRule.NamesRE) == 0 {
		assert.Fail(t, "ConfigObj should contain S3 names regexes, %+v\n", configObj)
	}

	return
}

// Test that rules get parsed for a resource config with exclude rules.
func TestConfigS3_ExcludeNames(t *testing.T) {
	configFilePath := "./mocks/s3_exclude_names.yaml"
	configObj, err := GetConfig(configFilePath)

	require.NoError(t, err)

	if reflect.DeepEqual(configObj, emptyConfig()) {
		assert.Fail(t, "Config should not be empty, %+v\n", configObj)
	}

	if len(configObj.S3.ExcludeRule.NamesRE) == 0 {
		assert.Fail(t, "ConfigObj should contain S3 names regexes, %+v\n", configObj)
	}

	return
}

// Test that rules get parsed for a resource config with both include and exclude rules.
func TestConfigS3_FilterNames(t *testing.T) {
	configFilePath := "./mocks/s3_filter_names.yaml"
	configObj, err := GetConfig(configFilePath)

	require.NoError(t, err)

	if reflect.DeepEqual(configObj, emptyConfig()) {
		assert.Fail(t, "Config should not be empty, %+v\n", configObj)
	}

	if len(configObj.S3.IncludeRule.NamesRE) == 0 ||
		len(configObj.S3.ExcludeRule.NamesRE) == 0 {
		assert.Fail(t, "ConfigObj should contain S3 names regexes, %+v\n", configObj)
	}

	return
}

// A single test for each of the other resources
func TestConfigIAMRole_FilterNames(t *testing.T) {
	configFilePath := "./mocks/iamrole_filter_names.yaml"
	configObj, err := GetConfig(configFilePath)

	require.NoError(t, err)

	if reflect.DeepEqual(configObj, emptyConfig()) {
		assert.Fail(t, "Config should not be empty, %+v\n", configObj)
	}

	if len(configObj.IAMRole.IncludeRule.NamesRE) == 0 ||
		len(configObj.IAMRole.ExcludeRule.NamesRE) == 0 {
		assert.Fail(t, "ConfigObj should contain IAMRole names regexes, %+v\n", configObj)
	}

	return
}

// Test that two or more resource configs together work
func TestConfigMultiple_FilterNames(t *testing.T) {
	configFilePath := "./mocks/multiple_resources_filter_names.yaml"
	configObj, err := GetConfig(configFilePath)

	require.NoError(t, err)

	if reflect.DeepEqual(configObj, emptyConfig()) {
		assert.Fail(t, "Config should not be empty, %+v\n", configObj)
	}

	if len(configObj.IAMRole.IncludeRule.NamesRE) == 0 ||
		len(configObj.IAMRole.ExcludeRule.NamesRE) == 0 {
		assert.Fail(t, "ConfigObj should contain IAMRole names regexes, %+v\n", configObj)
	}

	return
}

// TODO: Once there is support for the below:
// TODO: Test filtering another property.
// TODO: Test filtering both names and another property.
