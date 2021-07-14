package config

import (
	"io/ioutil"
	"path/filepath"
	"regexp"

	"gopkg.in/yaml.v2"
)

// Config - the config object we pass around
type Config struct {
	S3                    ResourceType `yaml:"s3"`
	IAMUsers              ResourceType `yaml:"IAMUsers"`
	SecretsManagerSecrets ResourceType `yaml:"SecretsManager"`
	NatGateway            ResourceType `yaml:"NatGateway"`
	ACMPCA                ResourceType `yaml:"ACMPCA"`
}

type ResourceType struct {
	IncludeRule FilterRule `yaml:"include"`
	ExcludeRule FilterRule `yaml:"exclude"`
}

type FilterRule struct {
	NamesRegExp []Expression `yaml:"names_regex"`
}

type Expression struct {
	RE regexp.Regexp
}

// UnmarshalText - Internally used by yaml.Unmarshal to unmarshall an Expression field
func (expression *Expression) UnmarshalText(data []byte) error {
	var pattern string

	if err := yaml.Unmarshal(data, &pattern); err != nil {
		return err
	}

	re, err := regexp.Compile(pattern)
	if err != nil {
		return err
	}

	expression.RE = *re

	return nil
}

// GetConfig - Unmarshall the config file and parse it into a config object.
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

	err = yaml.Unmarshal(yamlFile, &configObj)
	if err != nil {
		return nil, err
	}

	return &configObj, nil
}

func matches(name string, regexps []Expression) bool {
	for _, re := range regexps {
		if re.RE.MatchString(name) {
			return true
		}
	}
	return false
}

// ShouldInclude - Checks if a name should be included according to the inclusion and exclusion rules
func ShouldInclude(name string, includeREs []Expression, excludeREs []Expression) bool {
	shouldInclude := false

	if len(includeREs) > 0 {
		// If any include rules are specified,
		// only check to see if an exclude rule matches when an include rule matches the user
		if matches(name, includeREs) {
			shouldInclude = true
			if matches(name, excludeREs) {
				shouldInclude = false
			}
		}
	} else if len(excludeREs) > 0 {
		// Only check to see if an exclude rule matches when there are no include rules defined
		if matches(name, excludeREs) {
			shouldInclude = false
		}
	} else {
		shouldInclude = true
	}

	return shouldInclude
}
