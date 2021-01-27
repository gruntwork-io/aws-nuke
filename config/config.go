package config

import (
	"io/ioutil"
	"path/filepath"
	"regexp"

	"gopkg.in/yaml.v2"
)

// RawConfig - used to unmarshall the raw config file
type RawConfig struct {
	S3      rawResourceType `yaml:"s3"`
	IamRole rawResourceType `yaml:"iam-role"`
}

type rawResourceType struct {
	IncludeRule rawFilterRule `yaml:"include"`
	ExcludeRule rawFilterRule `yaml:"exclude"`
}

type rawFilterRule struct {
	NamesRE []string `yaml:"names_regex"`
}

// Config - the config object we pass around
// that is a parsed version of RawConfig
type Config struct {
	S3      ResourceType
	IamRole ResourceType
}

// ResourceType - the include and exclude
// rules for a resource type
type ResourceType struct {
	IncludeRule FilterRule
	ExcludeRule FilterRule
}

// FilterRule - contains regular expressions or plain text patterns
// used to match against a resource type's properties
type FilterRule struct {
	NamesRE []*regexp.Regexp
}

// GetConfig - unmarshall the raw config file
// and parse it into a config object.
func GetConfig(filePath string) (*Config, error) {
	var configObj Config

	absolutePath, err := filepath.Abs(filePath)
	if err != nil {
		return nil, err
	}

	yamlFile, err := ioutil.ReadFile(absolutePath)
	if err != nil {
		return nil, err
	}

	rawConfig := RawConfig{}

	err = yaml.Unmarshal(yamlFile, &rawConfig)
	if err != nil {
		return nil, err
	}

	for _, pattern := range rawConfig.S3.IncludeRule.NamesRE {
		re, err := regexp.Compile(pattern)
		if err != nil {
			return nil, err
		}

		configObj.S3.IncludeRule.NamesRE = append(configObj.S3.IncludeRule.NamesRE, re)
	}

	for _, pattern := range rawConfig.S3.ExcludeRule.NamesRE {
		re, err := regexp.Compile(pattern)
		if err != nil {
			return nil, err
		}

		configObj.S3.ExcludeRule.NamesRE = append(configObj.S3.ExcludeRule.NamesRE, re)
	}

	for _, pattern := range rawConfig.IamRole.IncludeRule.NamesRE {
		re, err := regexp.Compile(pattern)
		if err != nil {
			return nil, err
		}

		configObj.IamRole.IncludeRule.NamesRE = append(configObj.IamRole.IncludeRule.NamesRE, re)
	}

	for _, pattern := range rawConfig.IamRole.ExcludeRule.NamesRE {
		re, err := regexp.Compile(pattern)
		if err != nil {
			return nil, err
		}

		configObj.IamRole.ExcludeRule.NamesRE = append(configObj.IamRole.ExcludeRule.NamesRE, re)
	}

	return &configObj, nil
}
