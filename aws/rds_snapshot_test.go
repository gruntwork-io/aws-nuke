package aws

import (
	"strings"
	"testing"
	"time"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/rds"

	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/gruntwork-io/gruntwork-cli/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createTestDBInstance generates a test DBInstance since snapshots can only be created
// from an existing DB Instance
func createTestDBInstance(t *testing.T, session *session.Session, name string) {
	svc := rds.New(session)
	input := &rds.CreateDBInstanceInput{
		AllocatedStorage:     awsgo.Int64(5),
		DBInstanceClass:      awsgo.String("db.t2.micro"),
		DBInstanceIdentifier: awsgo.String(name),
		Engine:               awsgo.String("postgres"),
		MasterUsername:       awsgo.String("gruntwork"),
		MasterUserPassword:   awsgo.String("password"),
	}

	_, err := svc.CreateDBInstance(input)
	require.NoError(t, err)

	svc.WaitUntilDBInstanceAvailable(&rds.DescribeDBInstancesInput{
		DBInstanceIdentifier: &name,
	})
}

// Tag a DB Instance snapshot
func tagTestRDSInstanceSnapshot(t *testing.T, session *session.Session, arn *string) {
	// Generate a tag
	svc := rds.New(session)
	var tags = []*rds.Tag{
		{
			Key:   awsgo.String("snapshot-tag1-key"),
			Value: awsgo.String("snapshot-tag1-value"),
		},
	}

	_, err := svc.AddTagsToResource(&rds.AddTagsToResourceInput{
		ResourceName: awsgo.String(*arn),
		Tags:         tags,
	})

	if err != nil {
		assert.Failf(t, "Unable to tag RDS DB Snapshot", errors.WithStackTrace(err).Error())
	}
}

//createTestRDSSnapshot generates a test DB Instance Snapshot
func createTestRDSSnapshot(t *testing.T, session *session.Session, instanceName string, snapshotName string) {
	svc := rds.New(session)
	input := &rds.CreateDBSnapshotInput{
		DBInstanceIdentifier: awsgo.String(instanceName),
		DBSnapshotIdentifier: awsgo.String(snapshotName),
	}

	_, err := svc.CreateDBSnapshot(input)
	require.NoError(t, errors.WithStackTrace(err))

	svc.WaitUntilDBSnapshotAvailable(&rds.DescribeDBSnapshotsInput{
		DBInstanceIdentifier: &instanceName,
		DBSnapshotIdentifier: &snapshotName,
	})

	result, err := svc.DescribeDBSnapshots(&rds.DescribeDBSnapshotsInput{
		DBSnapshotIdentifier: &snapshotName,
	})

	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	for _, snapshot := range result.DBSnapshots {
		arn := snapshot.DBSnapshotArn
		tagTestRDSInstanceSnapshot(t, session, arn)
	}
}

// TestFilterRDSSnapshot tests for filtering snapshots basing on matching include and exclude rules in the config file.
func TestFilterRDSSnapshot(t *testing.T) {
	t.Parallel()

	region, err := getRandomRegion()

	require.NoError(t, errors.WithStackTrace(err))

	session, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region)},
	)

	// Create a test DB Instance and use the dbInstanceIdentifier to generate snapshots
	instanceName := "cloud-nuke-test-instance-" + util.UniqueID()
	excludeAfter := time.Now().Add(1 * time.Hour)
	createTestDBInstance(t, session, instanceName)
	instanceNames, err := getAllRdsInstances(session, excludeAfter)

	if err != nil {
		assert.Failf(t, "Unable to fetch list of RDS DB Instances", errors.WithStackTrace(err).Error())
	}

	if len(instanceNames) > 0 {
		logging.Logger.Debug("List of RDS DB instances is empty")
	}
	dbInstanceIdentifier := awsgo.StringValueSlice(instanceNames)[0]

	// Clean up after this test
	defer nukeAllRdsInstances(session, []*string{&instanceName})

	// Create a test snapshot to nuke based on matching rules in the config file.
	snapshotName := "cloud-nuke-test-include-snapshot-" + util.UniqueID()
	createTestRDSSnapshot(t, session, dbInstanceIdentifier, snapshotName)

	// Create a test snapshot to exclude when nuking
	snapshotExcludedName := "cloud-nuke-test-exclude-snapshot-" + util.UniqueID()
	createTestRDSSnapshot(t, session, dbInstanceIdentifier, snapshotExcludedName)

	var configObj *config.Config
	configObj, err = config.GetConfig("../config/mocks/rdsSnapshots_filter_names.yaml")

	defer func() {
		snapShots, err := getAllRdsSnapshots(session, excludeAfter, *configObj)

		if err != nil {
			assert.Failf(t, "Unable to fetch list of RDS DB Instance snapshots", errors.WithStackTrace(err).Error())
		}
		nukeAllRdsSnapshots(session, snapShots)

		snapshotNames, _ := getAllRdsSnapshots(session, excludeAfter, *configObj)

		// Verify that the snapshot has been nuked
		assert.NotContains(t, awsgo.StringValueSlice(snapshotNames), strings.ToLower(snapshotName))

	}()

	snapShots, err := getAllRdsSnapshots(session, excludeAfter, *configObj)

	if err != nil {
		assert.Failf(t, "Unable to fetch list of RDS DB Instance snapshots", errors.WithStackTrace(err).Error())
	}

	assert.Contains(t, awsgo.StringValueSlice(snapShots), strings.ToLower(snapshotName))

	// Verfiy that the snapshot has been excluded
	assert.NotContains(t, awsgo.StringValueSlice(snapShots), strings.ToLower(snapshotExcludedName))

}
