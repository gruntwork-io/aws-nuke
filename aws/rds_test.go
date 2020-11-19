package aws

import (
	"strings"
	"testing"
	"time"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/rds"

	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/gruntwork-io/gruntwork-cli/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// There's a built-in function WaitUntilDBInstanceAvailable but
// the times that it was tested, it wasn't returning anything so we'll leave with the
// custom one.
func waitUntilRdsCreated(svc *rds.RDS, name *string) error {
	input := &rds.DescribeDBInstancesInput{
		DBInstanceIdentifier: name,
	}

	for i := 0; i < 240; i++ {
		instance, err := svc.DescribeDBInstances(input)
		status := instance.DBInstances[0].DBInstanceStatus

		// If SkipFinalSnapshot = false on delete, should also wait for "backing-up" also to finish
		if awsgo.StringValue(status) != "creating" {
			return nil
		}

		if err != nil {
			return err
		}

		time.Sleep(1 * time.Second)
		logging.Logger.Debug("Waiting for RDS DB Instance to be created")
	}

	return RdsDeleteError{name: *name}
}

func createTestRDSInstance(t *testing.T, session *session.Session, name string) {
	svc := rds.New(session)
	params := &rds.CreateDBInstanceInput{
		AllocatedStorage:     awsgo.Int64(5),
		DBInstanceClass:      awsgo.String("db.m5.large"),
		DBInstanceIdentifier: awsgo.String(name),
		Engine:               awsgo.String("postgres"),
		MasterUsername:       awsgo.String("gruntwork"),
		MasterUserPassword:   awsgo.String("password"),
	}

	_, err := svc.CreateDBInstance(params)
	require.NoError(t, err)

	waitUntilRdsCreated(svc, &name)
}

func TestNukeRDSInstance(t *testing.T) {
	t.Parallel()

	region, err := getRandomRegion()

	require.NoError(t, errors.WithStackTrace(err))

	session, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region)},
	)

	rdsName := "cloud-nuke-test-" + util.UniqueID()
	excludeAfter := time.Now().Add(1 * time.Hour)

	createTestRDSInstance(t, session, rdsName)

	defer func() {
		nukeAllRdsInstances(session, []*string{&rdsName})

		rdsNames, _ := getAllRdsInstances(session, excludeAfter)

		assert.NotContains(t, awsgo.StringValueSlice(rdsNames), strings.ToLower(rdsName))
	}()

	instances, err := getAllRdsInstances(session, excludeAfter)

	if err != nil {
		assert.Failf(t, "Unable to fetch list of RDS DB Instances", errors.WithStackTrace(err).Error())
	}

	assert.Contains(t, awsgo.StringValueSlice(instances), strings.ToLower(rdsName))

}
