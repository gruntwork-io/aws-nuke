package aws

import (
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/gruntwork-io/gruntwork-cli/errors"
)

type DBInstances struct {
	InstanceNames []string
}

func (instance DBInstances) ResourceName() string {
	return "rds"
}

// ResourceIdentifiers - The instance names of the rds db instances
func (instance DBInstances) ResourceIdentifiers() []string {
	return instance.InstanceNames
}

func (instance DBInstances) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle
	return 200
}

// Nuke - nuke 'em all!!!
func (instance DBInstances) Nuke(session *session.Session, identifiers []string) error {
	if err := nukeAllRdsInstances(session, awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

type RdsDeleteError struct {
	name string
}

func (e RdsDeleteError) Error() string {
	return "RDS DB Instance:" + e.name + "was not deleted"
}
