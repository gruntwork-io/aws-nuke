package aws

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/guardduty"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/go-commons/errors"
)

// getAllGuardDutyDetectors returns active GuardDuty detectors in a given region.
func getAllGuardDutyDetectors(session *session.Session, excludeAfter time.Time) ([]string, error) {
	svc := guardduty.New(session)

	ctx := context.Background()
	var detectorIds []string

	err := svc.ListDetectorsPagesWithContext(ctx, &guardduty.ListDetectorsInput{},
		func(d *guardduty.ListDetectorsOutput, lastPage bool) bool {
			for _, detectorId := range d.DetectorIds {
				detectorIds = append(detectorIds, aws.StringValue(detectorId))
			}
			return true
		})
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	var results []string

	for _, detectorId := range detectorIds {
		layout := "2006-01-02T15:04:05.000Z"
		output, err := svc.GetDetector(&guardduty.GetDetectorInput{DetectorId: aws.String(detectorId)})
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

		createdTime, err := time.Parse(layout, *output.CreatedAt)
		if excludeAfter.After(createdTime) {
			results = append(results, detectorId)
		}

	}

	return results, nil
}

// nukeAllGuardDutyDetectors disables GuardDuty for an AWS region. This will also remove all associated findings.
func nukeAllGuardDutyDetectors(session *session.Session, detectorIds []string) error {
	svc := guardduty.New(session)

	if len(detectorIds) == 0 {
		logging.Logger.Infof("GuardDuty was not found to be enabled in an region: %s", *session.Config.Region)
		return nil
	}

	logging.Logger.Infof("Deleting all GuardDuty instances in region: %s", *session.Config.Region)

	ctx := context.Background()
	for _, detectorId := range detectorIds {
		_, err := svc.DeleteDetectorWithContext(ctx, &guardduty.DeleteDetectorInput{
			DetectorId: aws.String(detectorId),
		})
		if err != nil {
			logging.Logger.Errorf("[Failed] %s", err)
		} else {
			logging.Logger.Infof("Deleted GuardDuty instance: %s", detectorId)
		}
	}

	return nil
}
