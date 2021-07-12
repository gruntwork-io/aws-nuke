package aws

import (
	"testing"
	"time"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/guardduty"
	"github.com/stretchr/testify/require"
)

// createTestGuardDuty creates a GuardDuty detector
func createTestGuardDuty(t *testing.T, session *session.Session) string {
	svc := guardduty.New(session)

	detector, err := svc.CreateDetector(&guardduty.CreateDetectorInput{
		Enable: awsgo.Bool(true),
	})
	require.NoErrorf(t, err, "Could not create test GuardDuty instance")

	return *detector.DetectorId
}

func TestNukeAllGuardDutyDetectors(t *testing.T) {
	t.Parallel()

	region, err := getRandomRegion()
	require.NoError(t, err)

	sess, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region),
	})
	require.NoError(t, err)

	detectorID := createTestGuardDuty(t, sess)
	defer func(session *session.Session, detectors []string) {
		err := nukeAllGuardDutyDetectors(session, detectors)
		require.NoError(t, err)

		// make sure all GuardDuty instances were actually deleted
		detectors, err = getAllGuardDutyDetectors(sess, time.Now().Add(1*time.Hour))
		require.NoErrorf(t, err, "Unable to list GuardDuty detector in region")

		require.Truef(t, len(detectors) == 0, "GuardDuty detectors still found after cleanup")
	}(sess, []string{detectorID})

	detectors, err := getAllGuardDutyDetectors(sess, time.Now().Add(1*time.Hour*-1))
	require.NoErrorf(t, err, "Unable to list GuardDuty detector in region")

	require.NotContains(t, detectors, detectorID)

	detectors, err = getAllGuardDutyDetectors(sess, time.Now().Add(1*time.Hour))
	require.NoErrorf(t, err, "Unable to list GuardDuty detector in region")

	require.Contains(t, detectors, detectorID)
}
