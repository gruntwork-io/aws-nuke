package aws

import (
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"

	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/gruntwork-cli/errors"
	"github.com/stretchr/testify/assert"
)

//TODO: This test should check that IAM Roles with path /aws-service-role/
// are not part of the nukable roles, and fail if so.
func TestListIamRoles(t *testing.T) {
	sess, err := session.NewSession(&aws.Config{})

	input := &iam.ListRolesInput{}

	_, err = getAllIamRoles(sess, input)
	if err != nil {
		assert.Fail(t, "Failed to list IAM Roles", errors.WithStackTrace(err))
	}
}

func TestNukeIamRoles(t *testing.T) {
	sess, err := session.NewSession(&aws.Config{})

	input := &iam.ListRolesInput{}

	output, err := getAllIamRoles(sess, input)
	if err != nil {
		assert.Fail(t, "Failed to list IAM Roles", errors.WithStackTrace(err))
	}

	var roles []string
	confObj := config.Config{}

	for _, role := range output.Roles {
		roles = append(roles, role)
	}

	filteredRoles := excludeServiceIamRoles(roles)

	err = nukeAllIamRoles(sess, filteredRoles, configObj)
	if err != nil {
		assert.Fail(t, "Failed to nuke IAM Roles", errors.WithStackTrace(err))
	}
}
