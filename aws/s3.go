package aws

import (
	"fmt"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/gruntwork-cli/errors"
)

// getS3BucketRegion returns S3 Bucket region.
func getS3BucketRegion(svc *s3.S3, bucketName string) (string, error) {
	input := &s3.GetBucketLocationInput{
		Bucket: aws.String(bucketName),
	}

	result, err := svc.GetBucketLocation(input)
	if err != nil {
		return "", err
	}

	if result.LocationConstraint == nil {
		// GetBucketLocation returns nil for us-east-1
		// https://github.com/aws/aws-sdk-go/issues/1687
		return "us-east-1", nil
	}
	return *result.LocationConstraint, nil
}

// getS3BucketTags returns S3 Bucket tags.
func getS3BucketTags(svc *s3.S3, bucketName string) ([]map[string]string, error) {
	input := &s3.GetBucketTaggingInput{
		Bucket: aws.String(bucketName),
	}

	tags := []map[string]string{}

	// Please note that svc argument should be created from a session object which is
	// in the same region as the bucket or GetBucketTagging will fail.
	result, err := svc.GetBucketTagging(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case "NoSuchTagSet":
				return tags, nil
			}
		}
		return tags, err
	}

	for _, tagSet := range result.TagSet {
		tags = append(tags, map[string]string{"Key": *tagSet.Key, "Value": *tagSet.Value})
	}

	return tags, nil
}

// S3Bucket - represents S3 bucket
type S3Bucket struct {
	Name          string
	CreationDate  time.Time
	Region        string
	Tags          []map[string]string
	Error         error
	IsValid       bool
	InvalidReason string
}

// getAllS3Buckets returns a map of per region AWS S3 buckets which were created before excludeAfter
func getAllS3Buckets(awsSession *session.Session, excludeAfter time.Time,
	targetRegions []string, bucketNameSubStr string, batchSize int,
	resourceNamePattern string, excludeResourceNamePattern string,
	requireResourceTag string, excludeResourceTag string) (map[string][]*string, error) {

	if batchSize <= 0 {
		return nil, fmt.Errorf("Invalid batchsize - %d - should be > 0", batchSize)
	}

	svc := s3.New(awsSession)
	input := &s3.ListBucketsInput{}
	output, err := svc.ListBuckets(input)
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	regionClients, err := getRegionClients(targetRegions)
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	var bucketNamesPerRegion = make(map[string][]*string)
	totalBuckets := len(output.Buckets)
	if totalBuckets == 0 {
		return bucketNamesPerRegion, nil
	}

	totalBatches := int(math.Ceil(float64(totalBuckets) / float64(batchSize)))
	batchCount := 1

	// Batch the get operation
	for batchStart := 0; batchStart < totalBuckets; batchStart += batchSize {
		batchEnd := int(math.Min(float64(batchStart)+float64(batchSize), float64(totalBuckets)))
		logging.Logger.Infof("Getting - %d-%d buckets of batch %d/%d", batchStart+1, batchEnd, batchCount, totalBatches)
		targetBuckets := output.Buckets[batchStart:batchEnd]
		currBucketNamesPerRegion, err := getBucketNamesPerRegion(svc, targetBuckets, excludeAfter, regionClients, bucketNameSubStr)

		if err != nil {
			return bucketNamesPerRegion, err
		}

		for region, buckets := range currBucketNamesPerRegion {
			if _, ok := bucketNamesPerRegion[region]; !ok {
				bucketNamesPerRegion[region] = []*string{}
			}
			for _, bucket := range buckets {
				bucketNamesPerRegion[region] = append(bucketNamesPerRegion[region], bucket)
			}
		}
		batchCount++
	}
	return bucketNamesPerRegion, nil
}

// getRegions creates s3 clients for target regions
func getRegionClients(regions []string) (map[string]*s3.S3, error) {
	var regionClients = make(map[string]*s3.S3)
	for _, region := range regions {
		logging.Logger.Debugf("S3 - creating session - region %s", region)
		awsSession, err := session.NewSession(&awsgo.Config{
			Region: awsgo.String(region)},
		)
		if err != nil {
			return regionClients, err
		}
		regionClients[region] = s3.New(awsSession)
	}
	return regionClients, nil
}

// getBucketNamesPerRegions gets valid bucket names concurrently from list of target buckets
func getBucketNamesPerRegion(svc *s3.S3, targetBuckets []*s3.Bucket, excludeAfter time.Time,
	regionClients map[string]*s3.S3, bucketNameSubStr string,
	resourceNamePattern string, excludeResourceNamePattern string,
	requireResourceTag string, excludeResourceTag string) (map[string][]*string, error) {

	var bucketNamesPerRegion = make(map[string][]*string)
	var bucketCh = make(chan *S3Bucket, len(targetBuckets))
	var wg sync.WaitGroup

	for _, bucket := range targetBuckets {
		if len(bucketNameSubStr) > 0 && !strings.Contains(*bucket.Name, bucketNameSubStr) {
			logging.Logger.Debugf("Skipping - Bucket %s - failed substring filter - %s", *bucket.Name, bucketNameSubStr)
			continue
		}

		wg.Add(1)
		go func(bucket *s3.Bucket) {
			defer wg.Done()
			getBucketInfo(
				svc, bucket, excludeAfter, regionClients, bucketCh,
				resourceNamePattern, excludeResourceNamePattern,
				requireResourceTag, excludeResourceTag
			)
		}(bucket)
	}

	go func() {
		wg.Wait()
		close(bucketCh)
	}()

	// Start reading from the channel as soon as the data comes in - so that skip
	// messages are shown to the user as soon as possible
	for bucketData := range bucketCh {
		if bucketData.Error != nil {
			logging.Logger.Warnf("Skipping - Bucket %s - region - %s - error: %s", bucketData.Name, bucketData.Region, bucketData.Error)
			continue
		}
		if !bucketData.IsValid {
			logging.Logger.Debugf("Skipping - Bucket %s - region - %s - %s", bucketData.Name, bucketData.Region, bucketData.InvalidReason)
			continue
		}
		if _, ok := bucketNamesPerRegion[bucketData.Region]; !ok {
			bucketNamesPerRegion[bucketData.Region] = []*string{}
		}
		bucketNamesPerRegion[bucketData.Region] = append(bucketNamesPerRegion[bucketData.Region], aws.String(bucketData.Name))
	}
	return bucketNamesPerRegion, nil
}

// getBucketInfo populates the local S3Bucket struct for the passed AWS bucket
func getBucketInfo(
	svc *s3.S3, bucket *s3.Bucket, excludeAfter time.Time,
	regionClients map[string]*s3.S3, bucketCh chan<- *S3Bucket,
	resourceNamePattern string, excludeResourceNamePattern string,
	requireResourceTag string, excludeResourceTag string
	) {
	var bucketData S3Bucket
	bucketData.Name = aws.StringValue(bucket.Name)
	bucketData.CreationDate = aws.TimeValue(bucket.CreationDate)

	bucketRegion, err := getS3BucketRegion(svc, bucketData.Name)
	if err != nil {
		bucketData.Error = err
		bucketCh <- &bucketData
		return
	}
	bucketData.Region = bucketRegion

	// Check if the bucket is in target region
	matchedRegion := false
	for region := range regionClients {
		if region == bucketData.Region {
			matchedRegion = true
			break
		}
	}
	if !matchedRegion {
		bucketData.InvalidReason = "Not in target region"
		bucketCh <- &bucketData
		return
	}

	// Check if the bucket has valid tags
	bucketTags, err := getS3BucketTags(regionClients[bucketData.Region], bucketData.Name)
	if err != nil {
		bucketData.Error = err
		bucketCh <- &bucketData
		return
	}
	bucketData.Tags = bucketTags
	if !HasValidTags(bucketData.Tags, requireResourceTag, excludeResourceTag) {
		bucketData.InvalidReason = "Matched tag filter"
		bucketCh <- &bucketData
		return
	}

	// Check if the bucket has valid name
	if !HasValidName(bucketData.Tags, resourceNamePattern, excludeResourceNamePattern) {
		bucketData.InvalidReason = "Matched name filter"
		bucketCh <- &bucketData
		return
	}

	// Check if the bucket is older than the required time
	if !excludeAfter.After(bucketData.CreationDate) {
		bucketData.InvalidReason = "Matched CreationDate filter"
		bucketCh <- &bucketData
		return
	}

	bucketData.IsValid = true
	bucketCh <- &bucketData
}

// getS3BucketObjects returns S3 bucket objects struct for a given bucket name
func getS3BucketObjects(svc *s3.S3, bucketName *string, isVersioned bool) ([]*s3.ObjectIdentifier, error) {
	identifiers := []*s3.ObjectIdentifier{}

	// Handle versioned buckets.
	if isVersioned {
		err := svc.ListObjectVersionsPages(
			&s3.ListObjectVersionsInput{
				Bucket: bucketName,
			},
			func(page *s3.ListObjectVersionsOutput, lastPage bool) (shouldContinue bool) {
				for _, obj := range page.Versions {
					logging.Logger.Debugf("Bucket %s object %s version %s", *bucketName, *obj.Key, *obj.VersionId)
					identifiers = append(identifiers, &s3.ObjectIdentifier{
						Key:       obj.Key,
						VersionId: obj.VersionId,
					})
				}
				for _, obj := range page.DeleteMarkers {
					logging.Logger.Debugf("Bucket %s object %s DeleteMarker %s", *bucketName, *obj.Key, *obj.VersionId)
					identifiers = append(identifiers, &s3.ObjectIdentifier{
						Key:       obj.Key,
						VersionId: obj.VersionId,
					})
				}
				return true
			},
		)
		return identifiers, err
	}

	// Handle non versioned buckets.
	err := svc.ListObjectsV2Pages(
		&s3.ListObjectsV2Input{
			Bucket: bucketName,
		},
		func(page *s3.ListObjectsV2Output, lastPage bool) (shouldContinue bool) {
			for _, obj := range page.Contents {
				logging.Logger.Debugf("Bucket %s object %s", *bucketName, *obj.Key)
				identifiers = append(identifiers, &s3.ObjectIdentifier{
					Key: obj.Key,
				})
			}
			return true
		},
	)

	return identifiers, err
}

// nukeAllS3BucketObjects batch deletes all objects in an S3 bucket
func nukeAllS3BucketObjects(svc *s3.S3, bucketName *string, batchSize int) error {
	versioningResult, err := svc.GetBucketVersioning(&s3.GetBucketVersioningInput{
		Bucket: bucketName,
	})
	if err != nil {
		return err
	}

	isVersioned := versioningResult.Status != nil && *versioningResult.Status == "Enabled"

	objects, err := getS3BucketObjects(svc, bucketName, isVersioned)
	if err != nil {
		return err
	}

	totalObjects := len(objects)

	if totalObjects == 0 {
		logging.Logger.Infof("Bucket: %s - empty - skipping object deletion", *bucketName)
		return nil
	}

	if batchSize < 1 || batchSize > 1000 {
		return fmt.Errorf("Invalid batchsize - %d - should be between %d and %d", batchSize, 1, 1000)
	}

	logging.Logger.Infof("Deleting - Bucket: %s - objects: %d", *bucketName, totalObjects)

	totalBatches := totalObjects / batchSize
	if totalObjects%batchSize != 0 {
		totalBatches = totalBatches + 1
	}
	batchCount := 1

	// Batch the delete operation
	for i := 0; i < len(objects); i += batchSize {
		j := i + batchSize
		if j > len(objects) {
			j = len(objects)
		}

		logging.Logger.Debugf("Deleting - %d-%d objects of batch %d/%d - Bucket: %s", i+1, j, batchCount, totalBatches, *bucketName)

		delObjects := objects[i:j]
		_, err = svc.DeleteObjects(
			&s3.DeleteObjectsInput{
				Bucket: bucketName,
				Delete: &s3.Delete{
					Objects: delObjects,
					Quiet:   aws.Bool(false),
				},
			},
		)
		if err != nil {
			return err
		}

		logging.Logger.Infof("[OK] - %d-%d objects of batch %d/%d - Bucket: %s - deleted", i+1, j, batchCount, totalBatches, *bucketName)

		batchCount++
	}

	return nil
}

// nukeEmptyS3Bucket deletes an empty S3 bucket
func nukeEmptyS3Bucket(svc *s3.S3, bucketName *string, verifyBucketDeletion bool) error {
	_, err := svc.DeleteBucket(&s3.DeleteBucketInput{
		Bucket: bucketName,
	})
	if err != nil {
		return err
	}

	if !verifyBucketDeletion {
		return err
	}

	err = svc.WaitUntilBucketNotExists(&s3.HeadBucketInput{
		Bucket: bucketName,
	})
	return err
}

// nukeAllS3Buckets deletes all S3 buckets passed as input
func nukeAllS3Buckets(awsSession *session.Session, bucketNames []*string, objectBatchSize int) (delCount int, err error) {
	svc := s3.New(awsSession)
	verifyBucketDeletion := true

	if len(bucketNames) == 0 {
		logging.Logger.Infof("No S3 Buckets to nuke in region %s", *awsSession.Config.Region)
		return 0, nil
	}

	totalCount := len(bucketNames)

	logging.Logger.Infof("Deleting - %d S3 Buckets in region %s", totalCount, *awsSession.Config.Region)

	for bucketIndex := 0; bucketIndex < totalCount; bucketIndex++ {

		bucketName := bucketNames[bucketIndex]
		logging.Logger.Debugf("Deleting - %d/%d - Bucket: %s", bucketIndex+1, totalCount, *bucketName)

		err = nukeAllS3BucketObjects(svc, bucketName, objectBatchSize)
		if err != nil {
			logging.Logger.Errorf("[Failed] - %d/%d - Bucket: %s - object deletion error - %s", bucketIndex+1, totalCount, *bucketName, err)
			continue
		}

		err = nukeEmptyS3Bucket(svc, bucketName, verifyBucketDeletion)
		if err != nil {
			logging.Logger.Errorf("[Failed] - %d/%d - Bucket: %s - bucket deletion error - %s", bucketIndex+1, totalCount, *bucketName, err)
			continue
		}

		logging.Logger.Infof("[OK] - %d/%d - Bucket: %s - deleted", bucketIndex+1, totalCount, *bucketName)
		delCount++
	}

	return delCount, nil
}
