package aws

import (
	"regexp"
	"time"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/gruntwork-cli/errors"
)

// Get all DB Instance snapshots
func getAllRdsSnapshots(session *session.Session, excludeAfter time.Time, configObj config.Config) ([]*string, error) {
	svc := rds.New(session)

	var results []*rds.DBSnapshot

	// Paginated API. Fetch all pages
	err := svc.DescribeDBSnapshotsPages(&rds.DescribeDBSnapshotsInput{},
		func(page *rds.DescribeDBSnapshotsOutput, lastPage bool) bool {
			results = append(results, page.DBSnapshots...)
			return !lastPage
		})
	if err != nil {
		return nil, err
	}

	var snapshots []*string

	for _, snapshot := range results {

		// List all DB Instance Snapshot tags
		tagsResult, err := svc.ListTagsForResource(&rds.ListTagsForResourceInput{
			ResourceName: snapshot.DBSnapshotArn,
		})

		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

		// Automated DB snapshots can only be deleted by deleting the DB instance or
		// changing the backup retention period for the DB instance to 0.
		// This edge case can't be handled since all DB instance related automated snapshots will be deleted.
		// Refer to https://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/USER_DeleteSnapshot.html
		if awsgo.StringValue(snapshot.SnapshotType) != "automated" {
			filterSnapshots(session, excludeAfter, configObj, snapshot, snapshots, tagsResult)
		}
	}

	return snapshots, nil
}

func filterSnapshots(session *session.Session, excludeAfter time.Time, configObj config.Config, snapshot *rds.DBSnapshot, snapshots []*string, tagsResult *rds.ListTagsForResourceOutput) []*string {
	var IncludeSnapshotByName bool
	IncludeSnapshotByName = shouldIncludeSnapshotByName(*snapshot.DBSnapshotIdentifier, configObj.RDSSnapshots.IncludeRule.NamesRE, configObj.RDSSnapshots.ExcludeRule.NamesRE)

	// Check the snapshot creation time
	if snapshot.SnapshotCreateTime == nil || !excludeAfter.After(awsgo.TimeValue(snapshot.SnapshotCreateTime)) {
		return nil
	}

	// Check snapshot name against config file rules
	if !IncludeSnapshotByName {
		return nil
	}

	// Check snapshot tags against config file rules
	if IncludeSnapshotByName && len(tagsResult.TagList) > 0 {
		for _, tag := range tagsResult.TagList {
			if shouldIncludeSnapshotByTag(*tag.Key, configObj.RDSSnapshots.IncludeRule.TagNamesRE, configObj.RDSSnapshots.ExcludeRule.TagNamesRE) {
				snapshots = append(snapshots, snapshot.DBSnapshotIdentifier)
				return snapshots
			}
			return nil
		}
	}
	snapshots = append(snapshots, snapshot.DBSnapshotIdentifier)
	return snapshots
}

// Match against any regex in config file
func matchesAnyRegex(name string, reList []*regexp.Regexp) bool {
	for _, re := range reList {
		if re.MatchString(name) {
			return true
		}
	}
	return false
}

// Filter DB Instance snapshot by names_regex in config file
func shouldIncludeSnapshotByName(snapshotName string, includeNamesREList []*regexp.Regexp, excludeNamesREList []*regexp.Regexp) bool {
	// If any include rules are defined
	// and the include rule matches the snapshot name, check to see if an exclude rule matches
	if len(includeNamesREList) > 0 {
		if matchesAnyRegex(snapshotName, includeNamesREList) {
			if !matchesAnyRegex(snapshotName, excludeNamesREList) {
				return true
			}
		}
		// If there are no include rules defined, check to see if an exclude rule matches
	} else if len(excludeNamesREList) > 0 && matchesAnyRegex(snapshotName, excludeNamesREList) {
		return false
	} else {
		// Otherwise
		return true
	}
	return false
}

// Filter DB Instance snapshot by tags_regex in config file
func shouldIncludeSnapshotByTag(tagName string, includeTagNamesREList []*regexp.Regexp, excludeTagNamesREList []*regexp.Regexp) bool {
	// If any include rules are defined
	// and the include rule matches the snapshot tag, check to see if an exclude rule matches
	if len(includeTagNamesREList) > 0 {
		if matchesAnyRegex(tagName, includeTagNamesREList) {
			if !matchesAnyRegex(tagName, excludeTagNamesREList) {
				return true
			}
		}
		// If there are no include rules defined, check to see if an exclude rule matches
	} else if len(excludeTagNamesREList) > 0 && matchesAnyRegex(tagName, excludeTagNamesREList) {
		return false
	} else {
		// Otherwise
		return true
	}
	return false
}

// Nuke-Delete all DB Instance snapshots
func nukeAllRdsSnapshots(session *session.Session, snapshots []*string) error {
	svc := rds.New(session)

	if len(snapshots) == 0 {
		logging.Logger.Infof("No RDS DB Snapshot to nuke in region %s", *session.Config.Region)
		return nil
	}

	logging.Logger.Infof("Deleting all RDS DB Snapshots in region %s", *session.Config.Region)
	deletedSnapshots := []*string{}

	for _, snapshot := range snapshots {
		input := &rds.DeleteDBSnapshotInput{
			DBSnapshotIdentifier: snapshot,
		}

		_, err := svc.DeleteDBSnapshot(input)

		if err != nil {
			logging.Logger.Errorf("[Failed] %s: %s", *snapshot, err)
		} else {
			deletedSnapshots = append(deletedSnapshots, snapshot)
			logging.Logger.Infof("Deleted RDS DB Snapshot: %s", awsgo.StringValue(snapshot))
		}
	}

	if len(deletedSnapshots) > 0 {
		for _, snapshot := range deletedSnapshots {

			err := svc.WaitUntilDBSnapshotDeleted(&rds.DescribeDBSnapshotsInput{
				DBSnapshotIdentifier: snapshot,
			})

			if err != nil {
				logging.Logger.Errorf("[Failed] %s", err)
				return errors.WithStackTrace(err)
			}
		}
	}

	if len(deletedSnapshots) != len(snapshots) {
		logging.Logger.Errorf("[Failed] - %d/%d - RDS DB Snapshot(s) failed deletion in %s", len(snapshots)-len(deletedSnapshots), len(snapshots), *session.Config.Region)
	}

	logging.Logger.Infof("[OK] %d RDS DB Snapshot(s) deleted in %s", len(deletedSnapshots), *session.Config.Region)
	return nil
}
