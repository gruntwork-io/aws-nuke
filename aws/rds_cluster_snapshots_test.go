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

// Custom waiter function waitUntilRdsClusterAvailable uses the Amazon RDS API operation DescribeDBClustersInput
// to wait for the cluster to be available before returning.
// If the condition is not met within the max attempt window, an error will be returned.
func waitUntilRdsClusterAvailable(svc *rds.RDS, name *string) error {
	input := &rds.DescribeDBClustersInput{
		DBClusterIdentifier: name,
	}

	for i := 0; i < 240; i++ {
		_, err := svc.DescribeDBClusters(input)
		if err != nil {
			return err
		}

		time.Sleep(1 * time.Second)
		logging.Logger.Debug("Waiting for RDS DB Cluster to be created")
	}

	return RdsClusterAvailableError{name: *name}
}

// Built-in waiter function WaitUntilDBClusterSnapshotAvailable not working as expected.
// Created a custom one
func waitUntilRdsClusterSnapshotAvailable(svc *rds.RDS, clusterName *string, snapshotName *string) error {
	input := &rds.DescribeDBClusterSnapshotsInput{
		DBClusterIdentifier:         clusterName,
		DBClusterSnapshotIdentifier: snapshotName,
	}
	for i := 0; i < 240; i++ {
		_, err := svc.DescribeDBClusterSnapshots(input)
		if err != nil {
			return err
		}

		time.Sleep(1 * time.Second)
		logging.Logger.Debug("Waiting for RDS DB Cluster Snapshot to be created")
	}

	return RdsClusterSnapshotAvailableError{clusterName: *clusterName, snapshotName: *snapshotName}
}

// createTestDBCluster generates a test DB Cluster since snapshots can only be created
// from an existing DB Cluster
func createTestDBCluster(t *testing.T, session *session.Session, name string) {
	svc := rds.New(session)
	input := &rds.CreateDBClusterInput{
		DBClusterIdentifier: awsgo.String(name),
		Engine:              awsgo.String("aurora-mysql"),
		MasterUsername:      awsgo.String("gruntwork"),
		MasterUserPassword:  awsgo.String("password"),
	}

	_, err := svc.CreateDBCluster(input)
	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}
	waitUntilRdsClusterAvailable(svc, &name)
}

// Tag a DB Cluster snapshot
func tagTestRDSClusterSnapshot(t *testing.T, session *session.Session, arn *string) {
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
		assert.Failf(t, "Unable to tag RDS DB Cluster Snapshot", errors.WithStackTrace(err).Error())
	}
}

// createTestRDSClusterSnapshot generates a test DB Snapshot
func createTestRDSClusterSnapshot(t *testing.T, session *session.Session, clusterName string, snapshotName string) {
	svc := rds.New(session)
	input := &rds.CreateDBClusterSnapshotInput{
		DBClusterIdentifier:         awsgo.String(clusterName),
		DBClusterSnapshotIdentifier: awsgo.String(snapshotName),
	}

	_, err := svc.CreateDBClusterSnapshot(input)
	require.NoError(t, errors.WithStackTrace(err))
	waitUntilRdsClusterSnapshotAvailable(svc, &clusterName, &snapshotName)

	result, err := svc.DescribeDBClusterSnapshots(&rds.DescribeDBClusterSnapshotsInput{
		DBClusterSnapshotIdentifier: &snapshotName,
	})

	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	for _, snapshot := range result.DBClusterSnapshots {
		arn := snapshot.DBClusterSnapshotArn
		tagTestRDSClusterSnapshot(t, session, arn)
	}

}

// TestFilterRDSClusterSnapshot tests for filtering cluster snapshots basing on matching include and exclude rules in the config file.
func TestFilterRDSClusterSnapshot(t *testing.T) {
	t.Parallel()

	region, err := getRandomRegion()

	require.NoError(t, errors.WithStackTrace(err))

	session, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region)},
	)

	// Create a test DB Cluster and use the dbClusterIdentifier to generate snapshots
	clusterName := "cloud-nuke-test-cluster" + util.UniqueID()
	createTestDBCluster(t, session, clusterName)
	excludeAfter := time.Now().Add(1 * time.Hour)
	clusterNames, err := getAllRdsClusters(session, excludeAfter)
	if err != nil {
		assert.Failf(t, "Unable to fetch list of RDS DB Clusters", errors.WithStackTrace(err).Error())
	}

	if len(clusterNames) > 0 {
		logging.Logger.Debug("List of RDS DB clusters is empty")
	}
	dbClusterIdentifier := awsgo.StringValueSlice(clusterNames)[0]

	// Clean up after this test
	defer nukeAllRdsClusters(session, []*string{&clusterName})

	// Create a test snapshot to nuke based on matching rules in the config file.
	snapshotName := "cloud-nuke-test-include-snapshot-" + util.UniqueID()
	createTestRDSClusterSnapshot(t, session, dbClusterIdentifier, snapshotName)

	// Create a test snapshot to exclude when nuking
	snapshotExcludedName := "cloud-nuke-test-exclude-snapshot-" + util.UniqueID()
	createTestRDSClusterSnapshot(t, session, dbClusterIdentifier, snapshotExcludedName)

	var configObj *config.Config
	configObj, err = config.GetConfig("../config/mocks/rdsSnapshots_filter_names.yaml")

	defer func() {
		snapShots, err := getAllRdsClusterSnapshots(session, excludeAfter, *configObj)

		if err != nil {
			assert.Failf(t, "Unable to fetch list of RDS DB Cluster snapshots", errors.WithStackTrace(err).Error())
		}
		nukeAllRdsClusterSnapshots(session, snapShots)

		snapshotNames, _ := getAllRdsClusterSnapshots(session, excludeAfter, *configObj)

		// Verify that the snapshot has been nuked
		assert.NotContains(t, awsgo.StringValueSlice(snapshotNames), strings.ToLower(snapshotName))

	}()

	snapShots, err := getAllRdsClusterSnapshots(session, excludeAfter, *configObj)

	if err != nil {
		assert.Failf(t, "Unable to fetch list of RDS DB Cluster snapshots", errors.WithStackTrace(err).Error())
	}

	assert.Contains(t, awsgo.StringValueSlice(snapShots), strings.ToLower(snapshotName))

	// Verify that the snapshot has been excluded
	assert.NotContains(t, awsgo.StringValueSlice(snapShots), strings.ToLower(snapshotExcludedName))

}
