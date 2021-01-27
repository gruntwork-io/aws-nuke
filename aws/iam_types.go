package aws

import (
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/gruntwork-io/gruntwork-cli/errors"
)

type IAMRoles struct {
	RoleNames []string
}

func (role IAMRoles) ResourceName() string {
	return "iamrole"
}

// ResourceIdentifiers - The IAM role name
func (role IAMRoles) ResourceIdentifiers() []string {
	return role.RoleNames
}

func (role IAMRoles) MaxBatchSize() int {
	return 100
}

// Nuke
func (role IAMRoles) Nuke(session *session.Session, identifiers []string) error {
	if err := nukeAllIAMRoles(session, awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

type IAMRoleDeleteError struct {
	name string
}

func (e IAMRoleDeleteError) Error() string {
	return "IAM Role: " + e.name + " was not deleted"
}
