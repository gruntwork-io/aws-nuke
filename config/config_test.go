package config

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func emptyConfig() *Config {
	return &Config{ResourceType{FilterRule{}, FilterRule{}}, ResourceType{FilterRule{}, FilterRule{}}}
}

func TestConfig_Garbage(t *testing.T) {
	configFilePath := "./mocks/garbage.yaml"
	configObj, err := GetConfig(configFilePath)

	require.NoError(t, err)

	if !reflect.DeepEqual(configObj, emptyConfig()) {
		assert.Fail(t, "Config should be empty, %+v\n", configObj)
	}

	return
}

func TestConfig_Empty(t *testing.T) {
	configFilePath := "./mocks/empty.yaml"
	configObj, err := GetConfig(configFilePath)

	require.NoError(t, err)

	if !reflect.DeepEqual(configObj, emptyConfig()) {
		assert.Fail(t, "Config should be empty, %+v\n", configObj)
	}

	return
}

func TestConfigS3_Malformed(t *testing.T) {
	configFilePath := "./mocks/s3_malformed.yaml"
	_, err := GetConfig(configFilePath)

	// Expect malformed to throw a yaml TypeError
	require.Error(t, err, "Received expected error")
	return
}

func TestConfigIamRole_Malformed(t *testing.T) {
	configPath := "./mocks/iamRole_malformed.yaml"
	_, err := GetConfig(configPath)

	require.Error(t, err, "Received expected error")
	return
}

func TestConfigS3_Empty(t *testing.T) {
	configFilePath := "./mocks/s3_empty.yaml"
	configObj, err := GetConfig(configFilePath)

	require.NoError(t, err)

	if !reflect.DeepEqual(configObj, emptyConfig()) {
		assert.Fail(t, "Config should be empty, %+v\n", configObj.S3)
	}

	return
}

func TestConfigIamRole_Empty(t *testing.T) {
	configFilePath := "./mocks/iamRole_empty.yaml"
	configObj, err := GetConfig(configFilePath)

	require.NoError(t, err)

	if !reflect.DeepEqual(configObj, emptyConfig()) {
		assert.Fail(t, "Config should be empty, %+v\n", configObj.IamRole)
	}

	return
}

func TestConfigS3_EmptyFilters(t *testing.T) {
	configFilePath := "./mocks/s3_empty_filters.yaml"
	configObj, err := GetConfig(configFilePath)

	require.NoError(t, err)

	if !reflect.DeepEqual(configObj, emptyConfig()) {
		assert.Fail(t, "Config should be empty, %+v\n", configObj)
	}

	return
}

func TestConfigIamRole_EmptyFilters(t *testing.T) {
	configFilePath := "./mocks/iamRole_empty_filters.yaml"
	configObj, err := GetConfig(configFilePath)

	require.NoError(t, err)

	if !reflect.DeepEqual(configObj, emptyConfig()) {
		assert.Fail(t, "Config should be empty, %+v\n", configObj)
	}

	return
}

func TestConfigS3_EmptyRules(t *testing.T) {
	configFilePath := "./mocks/s3_empty_rules.yaml"
	configObj, err := GetConfig(configFilePath)

	require.NoError(t, err)

	if !reflect.DeepEqual(configObj, emptyConfig()) {
		assert.Fail(t, "Config should be empty, %+v\n", configObj)
	}

	return
}

func TestConfigIamRole_EmptyRules(t *testing.T) {
	configFilePath := "./mocks/iamRole_empty_rules.yaml"
	configObj, err := GetConfig(configFilePath)

	require.NoError(t, err)

	if !reflect.DeepEqual(configObj, emptyConfig()) {
		assert.Fail(t, "Config should be empty, %+v\n", configObj)
	}

	return
}

func TestConfigS3_IncludeNames(t *testing.T) {
	configFilePath := "./mocks/s3_include_names.yaml"
	configObj, err := GetConfig(configFilePath)

	require.NoError(t, err)

	if reflect.DeepEqual(configObj, emptyConfig()) {
		assert.Fail(t, "Config should not be empty, %+v\n", configObj)
	}

	//TODO: Use assert.Greater to handle this comparison instead
	if len(configObj.S3.IncludeRule.NamesRE) == 0 {
		assert.Fail(t, "ConfigObj should contain S3 names regexes, %+v\n", configObj)
	}

	return
}

func TestConfigIamRole_IncludeNames(t *testing.T) {
	configFilePath := "./mocks/iamRole_include_names.yaml"
	configObj, err := GetConfig(configFilePath)

	require.NoError(t, err)

	if reflect.DeepEqual(configObj, emptyConfig()) {
		assert.Fail(t, "Config should not be empty, %+v\n", configObj)
	}

	//TODO: Use assert.Greater to handle this comparison instead
	if len(configObj.IamRole.IncludeRule.NamesRE) == 0 {
		assert.Fail(t, "ConfigObj should contain IAM Role names regexes, %+v\n", configObj)
	}

	return
}

func TestConfigS3_ExcludeNames(t *testing.T) {
	configFilePath := "./mocks/s3_exclude_names.yaml"
	configObj, err := GetConfig(configFilePath)

	require.NoError(t, err)

	if reflect.DeepEqual(configObj, emptyConfig()) {
		assert.Fail(t, "Config should not be empty, %+v\n", configObj)
	}

	//TODO: Use assert.Greater to handle this comparison instead
	if len(configObj.S3.ExcludeRule.NamesRE) == 0 {
		assert.Fail(t, "ConfigObj should contain S3 names regexes, %+v\n", configObj)
	}

	return
}

func TestConfigIamRole_ExcludeNames(t *testing.T) {
	configFilePath := "./mocks/iamRole_exclude_names.yaml"
	configObj, err := GetConfig(configFilePath)

	require.NoError(t, err)

	if reflect.DeepEqual(configObj, emptyConfig()) {
		assert.Fail(t, "Config should not be empty, %+v\n", configObj)
	}

	//TODO: Use assert.Greater to handle this comparison instead
	if len(configObj.IamRole.ExcludeRule.NamesRE) == 0 {
		assert.Fail(t, "ConfigObj should contain IAM Role names regexes, %+v\n", configObj)
	}

	return
}

func TestConfigS3_FilterNames(t *testing.T) {
	configFilePath := "./mocks/s3_filter_names.yaml"
	configObj, err := GetConfig(configFilePath)

	require.NoError(t, err)

	if reflect.DeepEqual(configObj, emptyConfig()) {
		assert.Fail(t, "Config should not be empty, %+v\n", configObj)
	}

	//TODO: Use assert.Greater to handle this comparisons instead
	if len(configObj.S3.IncludeRule.NamesRE) == 0 ||
		len(configObj.S3.ExcludeRule.NamesRE) == 0 {
		assert.Fail(t, "ConfigObj should contain S3 names regexes, %+v\n", configObj)
	}

	return
}

func TestConfigIamRole_FilterNames(t *testing.T) {
	configFilePath := "./mocks/iamRole_filter_names.yaml"
	configObj, err := GetConfig(configFilePath)

	require.NoError(t, err)

	if reflect.DeepEqual(configObj, emptyConfig()) {
		assert.Fail(t, "Config should not be empty, %+v\n", configObj)
	}

	//TODO: Use assert.Greater to handle this comparisons instead
	if len(configObj.IamRole.IncludeRule.NamesRE) == 0 ||
		len(configObj.IamRole.ExcludeRule.NamesRE) == 0 {
		assert.Fail(t, "ConfigObj should contain IAM Role names regexes, %+v\n", configObj)
	}

	return
}

func TestConfig_MixedConfig(t *testing.T) {
	configFilePath := "./mocks/mixedConfig.yaml"
	configObj, err := GetConfig(configFilePath)

	require.NoError(t, err)

	if reflect.DeepEqual(configObj, emptyConfig()) {
		assert.Fail(t, "Config should not be empty, %+v\n", configObj)
	}

	//TODO: Use assert.Greater to handle this comparisons instead
	if len(configObj.S3.IncludeRule.NamesRE) == 0 ||
		len(configObj.S3.ExcludeRule.NamesRE) == 0 ||
		len(configObj.IamRole.IncludeRule.NamesRE) == 0 ||
		len(configObj.IamRole.ExcludeRule.NamesRE) == 0 {
		assert.Fail(t, "ConfigObj should contain both IAM Role and S3 names regexes, %+v\n", configObj)
	}

	return
}
