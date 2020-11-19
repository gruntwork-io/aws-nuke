package aws

import (
	"fmt"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/gruntwork-io/gruntwork-cli/errors"
)

type DBSnapshots struct {
	SnapshotNames []string
}

// Name of the AWS resource
func (snapshot DBSnapshots) ResourceName() string {
	return "snapshots"
}

// Names of the RDS DB Snapshots
func (snapshot DBSnapshots) ResourceIdentifiers() []string {
	return snapshot.SnapshotNames
}

// MaxBatchSize decides how many snapshots to delete in one call.
func (snapshot DBSnapshots) MaxBatchSize() int {
	return 200
}

//Nuke/Delete all snapshots
func (snapshot DBSnapshots) Nuke(session *session.Session, identifiers []string) error {
	if err := nukeAllRdsSnapshots(session, awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

type RdsInstanceSnapshotAvailableError struct {
	instanceName string
	snapshotName string
}

func (e RdsInstanceSnapshotAvailableError) Error() string {
	return fmt.Sprintf("RDS DB Instance Snapshot %s not currently in available or failed state", e.snapshotName)
}

type RdsInstanceAvailableError struct {
	name string
}

func (e RdsInstanceAvailableError) Error() string {
	return fmt.Sprintf("RDS DB Instance %s not in available state", e.name)
}
