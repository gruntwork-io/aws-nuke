package aws

import (
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/gruntwork-io/go-commons/errors"
)

// GuardDutyInstances - represents all GuardDuty instances via detector IDs
type GuardDutyInstances struct {
	DetectorIds []string
}

// ResourceName - the simple name of the AWS resource
func (g GuardDutyInstances) ResourceName() string {
	return "guardduty"
}

// ResourceIdentifiers - the GuardDuty detector IDs
func (g GuardDutyInstances) ResourceIdentifiers() []string {
	return g.DetectorIds
}

// MaxBatchSize - decides how many GuardDuty instances to delete in one call
func (g GuardDutyInstances) MaxBatchSize() int {
	return 50
}

// Nuke - nuke 'em all, "it's the only way to be sure"
func (g GuardDutyInstances) Nuke(session *session.Session, identifiers []string) error {
	if err := nukeAllGuardDutyDetectors(session, identifiers); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
