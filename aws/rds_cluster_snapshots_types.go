package aws

import (
	"fmt"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/gruntwork-io/gruntwork-cli/errors"
)

type DBClusterSnapshots struct {
	SnapshotNames []string
}

// Name of the AWS resource
func (snapshot DBClusterSnapshots) ResourceName() string {
	return "snapshots"
}

// Names of the RDS DB Cluster Snapshots
func (snapshot DBClusterSnapshots) ResourceIdentifiers() []string {
	return snapshot.SnapshotNames
}

// MaxBatchSize decides how many cluster snapshots to delete in one call.
func (snapshot DBClusterSnapshots) MaxBatchSize() int {
	return 200
}

// Nuke/Delete all RDS DB Cluster snapshots
func (snapshot DBClusterSnapshots) Nuke(session *session.Session, identifiers []string) error {
	if err := nukeAllRdsClusterSnapshots(session, awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

type RdsClusterSnapshotDeleteError struct {
	name string
}

func (e RdsClusterSnapshotDeleteError) Error() string {
	return fmt.Sprintf("RDS DB Cluster Snapshot %s was not deleted", e.name)
}

type RdsClusterSnapshotAvailableError struct {
	clusterName  string
	snapshotName string
}

func (e RdsClusterSnapshotAvailableError) Error() string {
	return fmt.Sprintf("RDS DB Cluster Snapshot %s not currently in available or failed state", e.snapshotName)
}

type RdsClusterAvailableError struct {
	name string
}

func (e RdsClusterAvailableError) Error() string {
	return fmt.Sprintf("RDS DB Cluster  %s not in available state", e.name)
}
