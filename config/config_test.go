package config

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func emptyConfig() *Config {
	return &Config{
		ResourceType{FilterRule{}, FilterRule{}},
		ResourceType{FilterRule{}, FilterRule{}},
		ResourceType{FilterRule{}, FilterRule{}},
		ResourceType{FilterRule{}, FilterRule{}},
		ResourceType{FilterRule{}, FilterRule{}},
	}
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

func TestConfig_Malformed(t *testing.T) {
	configFilePath := "./mocks/malformed.yaml"
	_, err := GetConfig(configFilePath)

	// Expect malformed to throw a yaml TypeError
	require.Error(t, err, "Received expected error")
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

// ACMPCA Tests

func TestConfigACMPCA_Empty(t *testing.T) {
	configFilePath := "./mocks/acmpa_empty.yaml"
	configObj, err := GetConfig(configFilePath)

	require.NoError(t, err)

	if !reflect.DeepEqual(configObj, emptyConfig()) {
		assert.Fail(t, "Config should be empty, %+v\n", configObj.ACMPCA)
	}

	return
}

func TestConfigACMPCA_EmptyFilters(t *testing.T) {
	configFilePath := "./mocks/acmpca_empty_filters.yaml"
	configObj, err := GetConfig(configFilePath)

	require.NoError(t, err)

	if !reflect.DeepEqual(configObj, emptyConfig()) {
		assert.Fail(t, "Config should be empty, %+v\n", configObj)
	}

	return
}

func TestConfigACMPCA_EmptyRules(t *testing.T) {
	configFilePath := "./mocks/acmpca_empty_rules.yaml"
	configObj, err := GetConfig(configFilePath)

	require.NoError(t, err)

	if !reflect.DeepEqual(configObj, emptyConfig()) {
		assert.Fail(t, "Config should be empty, %+v\n", configObj)
	}

	return
}

func TestConfigACMPCA_IncludeNames(t *testing.T) {
	configFilePath := "./mocks/acmpca_include_names.yaml"
	configObj, err := GetConfig(configFilePath)

	require.NoError(t, err)

	if reflect.DeepEqual(configObj, emptyConfig()) {
		assert.Fail(t, "Config should not be empty, %+v\n", configObj)
	}

	if len(configObj.ACMPCA.IncludeRule.NamesRegExp) == 0 {
		assert.Fail(t, "ConfigObj should contain ACMPCA arn regexes, %+v\n", configObj)
	}

	return
}

func TestConfigACMPCA_ExcludeNames(t *testing.T) {
	configFilePath := "./mocks/acmpca_exclude_names.yaml"
	configObj, err := GetConfig(configFilePath)

	require.NoError(t, err)

	if reflect.DeepEqual(configObj, emptyConfig()) {
		assert.Fail(t, "Config should not be empty, %+v\n", configObj)
	}

	if len(configObj.ACMPCA.ExcludeRule.NamesRegExp) == 0 {
		assert.Fail(t, "ConfigObj should contain ACMPCA arn regexes, %+v\n", configObj)
	}

	return
}

func TestConfigACMPCA_FilterNames(t *testing.T) {
	configFilePath := "./mocks/acmpca_filter_names.yaml"
	configObj, err := GetConfig(configFilePath)

	require.NoError(t, err)

	if reflect.DeepEqual(configObj, emptyConfig()) {
		assert.Fail(t, "Config should not be empty, %+v\n", configObj)
	}

	if len(configObj.ACMPCA.IncludeRule.NamesRegExp) == 0 ||
		len(configObj.ACMPCA.ExcludeRule.NamesRegExp) == 0 {
		assert.Fail(t, "ConfigObj should contain ACMPCA arn regexes, %+v\n", configObj)
	}

	return
}

// S3 Tests

func TestConfigS3_Empty(t *testing.T) {
	configFilePath := "./mocks/s3_empty.yaml"
	configObj, err := GetConfig(configFilePath)

	require.NoError(t, err)

	if !reflect.DeepEqual(configObj, emptyConfig()) {
		assert.Fail(t, "Config should be empty, %+v\n", configObj.S3)
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

func TestConfigS3_EmptyRules(t *testing.T) {
	configFilePath := "./mocks/s3_empty_rules.yaml"
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

	if len(configObj.S3.IncludeRule.NamesRegExp) == 0 {
		assert.Fail(t, "ConfigObj should contain S3 names regexes, %+v\n", configObj)
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

	if len(configObj.S3.ExcludeRule.NamesRegExp) == 0 {
		assert.Fail(t, "ConfigObj should contain S3 names regexes, %+v\n", configObj)
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

	if len(configObj.S3.IncludeRule.NamesRegExp) == 0 ||
		len(configObj.S3.ExcludeRule.NamesRegExp) == 0 {
		assert.Fail(t, "ConfigObj should contain S3 names regexes, %+v\n", configObj)
	}

	return
}

// IAM Users Tests

func TestConfigIAM_Users_Empty(t *testing.T) {
	configFilePath := "./mocks/iam_users_empty.yaml"
	configObj, err := GetConfig(configFilePath)

	require.NoError(t, err)

	if !reflect.DeepEqual(configObj, emptyConfig()) {
		assert.Fail(t, "Config should be empty, %+v\n", configObj.IAMUsers)
	}

	return
}

func TestConfigIAM_Users_EmptyFilters(t *testing.T) {
	configFilePath := "./mocks/iam_users_empty_filters.yaml"
	configObj, err := GetConfig(configFilePath)

	require.NoError(t, err)

	if !reflect.DeepEqual(configObj, emptyConfig()) {
		assert.Fail(t, "Config should be empty, %+v\n", configObj)
	}

	return
}

func TestConfigIAM_Users_EmptyRules(t *testing.T) {
	configFilePath := "./mocks/iam_users_empty_rules.yaml"
	configObj, err := GetConfig(configFilePath)

	require.NoError(t, err)

	if !reflect.DeepEqual(configObj, emptyConfig()) {
		assert.Fail(t, "Config should be empty, %+v\n", configObj)
	}

	return
}

func TestConfigIAM_Users_IncludeNames(t *testing.T) {
	configFilePath := "./mocks/iam_users_include_names.yaml"
	configObj, err := GetConfig(configFilePath)

	require.NoError(t, err)

	if reflect.DeepEqual(configObj, emptyConfig()) {
		assert.Fail(t, "Config should not be empty, %+v\n", configObj)
	}

	if len(configObj.IAMUsers.IncludeRule.NamesRegExp) == 0 {
		assert.Fail(t, "ConfigObj should contain IAM names regexes, %+v\n", configObj)
	}

	return
}

func TestConfigIAM_Users_ExcludeNames(t *testing.T) {
	configFilePath := "./mocks/iam_users_exclude_names.yaml"
	configObj, err := GetConfig(configFilePath)

	require.NoError(t, err)

	if reflect.DeepEqual(configObj, emptyConfig()) {
		assert.Fail(t, "Config should not be empty, %+v\n", configObj)
	}

	if len(configObj.IAMUsers.ExcludeRule.NamesRegExp) == 0 {
		assert.Fail(t, "ConfigObj should contain IAM names regexes, %+v\n", configObj)
	}

	return
}

func TestConfigIAM_Users_FilterNames(t *testing.T) {
	configFilePath := "./mocks/iam_users_filter_names.yaml"
	configObj, err := GetConfig(configFilePath)

	require.NoError(t, err)

	if reflect.DeepEqual(configObj, emptyConfig()) {
		assert.Fail(t, "Config should not be empty, %+v\n", configObj)
	}

	if len(configObj.IAMUsers.IncludeRule.NamesRegExp) == 0 ||
		len(configObj.IAMUsers.ExcludeRule.NamesRegExp) == 0 {
		assert.Fail(t, "ConfigObj should contain IAM names regexes, %+v\n", configObj)
	}

	return
}

// Secrets Manager Tests

func TestConfigSecretsManager_Empty(t *testing.T) {
	configFilePath := "./mocks/secrets_manager_empty.yaml"
	configObj, err := GetConfig(configFilePath)

	require.NoError(t, err)

	if !reflect.DeepEqual(configObj, emptyConfig()) {
		assert.Fail(t, "Config should be empty, %+v\n", configObj.SecretsManagerSecrets)
	}

	return
}

func TestConfigSecretsManager_EmptyFilters(t *testing.T) {
	configFilePath := "./mocks/secrets_manager_empty_filters.yaml"
	configObj, err := GetConfig(configFilePath)

	require.NoError(t, err)

	if !reflect.DeepEqual(configObj, emptyConfig()) {
		assert.Fail(t, "Config should be empty, %+v\n", configObj)
	}

	return
}

func TestConfigSecretsManager_EmptyRules(t *testing.T) {
	configFilePath := "./mocks/secrets_manager_empty_rules.yaml"
	configObj, err := GetConfig(configFilePath)

	require.NoError(t, err)

	if !reflect.DeepEqual(configObj, emptyConfig()) {
		assert.Fail(t, "Config should be empty, %+v\n", configObj)
	}

	return
}

func TestConfigSecretsManager_IncludeNames(t *testing.T) {
	configFilePath := "./mocks/secrets_manager_include_names.yaml"
	configObj, err := GetConfig(configFilePath)

	require.NoError(t, err)

	if reflect.DeepEqual(configObj, emptyConfig()) {
		assert.Fail(t, "Config should not be empty, %+v\n", configObj)
	}

	if len(configObj.SecretsManagerSecrets.IncludeRule.NamesRegExp) == 0 {
		assert.Fail(t, "ConfigObj should contain secrets regexes, %+v\n", configObj)
	}

	return
}

func TestConfigSecretsManager_ExcludeNames(t *testing.T) {
	configFilePath := "./mocks/secrets_manager_exclude_names.yaml"
	configObj, err := GetConfig(configFilePath)

	require.NoError(t, err)

	if reflect.DeepEqual(configObj, emptyConfig()) {
		assert.Fail(t, "Config should not be empty, %+v\n", configObj)
	}

	if len(configObj.SecretsManagerSecrets.ExcludeRule.NamesRegExp) == 0 {
		assert.Fail(t, "ConfigObj should contain secrets regexes, %+v\n", configObj)
	}

	return
}

func TestConfigSecretsManager_FilterNames(t *testing.T) {
	configFilePath := "./mocks/secrets_manager_filter_names.yaml"
	configObj, err := GetConfig(configFilePath)

	require.NoError(t, err)

	if reflect.DeepEqual(configObj, emptyConfig()) {
		assert.Fail(t, "Config should not be empty, %+v\n", configObj)
	}

	if len(configObj.SecretsManagerSecrets.IncludeRule.NamesRegExp) == 0 ||
		len(configObj.SecretsManagerSecrets.ExcludeRule.NamesRegExp) == 0 {
		assert.Fail(t, "ConfigObj should contain secrets regexes, %+v\n", configObj)
	}

	return
}
