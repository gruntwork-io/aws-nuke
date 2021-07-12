package aws

import (
	"fmt"
	"math/rand"
	"sort"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/go-commons/collections"
	"github.com/gruntwork-io/go-commons/errors"
)

// OptInNotRequiredRegions contains all regions that are enabled by default on new AWS accounts
// Beginning in Spring 2019, AWS requires new regions to be explicitly enabled
// See https://aws.amazon.com/blogs/security/setting-permissions-to-enable-accounts-for-upcoming-aws-regions/
var OptInNotRequiredRegions = []string{
	"eu-north-1",
	"ap-south-1",
	"eu-west-3",
	"eu-west-2",
	"eu-west-1",
	"ap-northeast-2",
	"ap-northeast-1",
	"sa-east-1",
	"ca-central-1",
	"ap-southeast-1",
	"ap-southeast-2",
	"eu-central-1",
	"us-east-1",
	"us-east-2",
	"us-west-1",
	"us-west-2",
}

// GovCloudRegions contains all of the U.S. GovCloud regions. In accounts with GovCloud enabled, these are the
// only available regions.
var GovCloudRegions = []string{
	"us-gov-east-1",
	"us-gov-west-1",
}

const (
	GlobalRegion string = "global"
	// us-east-1 is the region that is available in every account
	defaultRegion string = "us-east-1"
)

func newSession(region string) *session.Session {
	return session.Must(
		session.NewSessionWithOptions(
			session.Options{
				SharedConfigState: session.SharedConfigEnable,
				Config: awsgo.Config{
					Region: awsgo.String(region),
				},
			},
		),
	)
}

// Try a describe regions command with the most likely enabled regions
func retryDescribeRegions() (*ec2.DescribeRegionsOutput, error) {
	regionsToTry := append(OptInNotRequiredRegions, GovCloudRegions...)
	for _, region := range regionsToTry {
		svc := ec2.New(newSession(region))
		regions, err := svc.DescribeRegions(&ec2.DescribeRegionsInput{})
		if err != nil {
			continue
		}
		return regions, nil
	}
	return nil, errors.WithStackTrace(fmt.Errorf("could not find any enabled regions"))
}

// GetEnabledRegions - Get all regions that are enabled (DescribeRegions excludes those not enabled by default)
func GetEnabledRegions() ([]string, error) {
	var regionNames []string

	// We don't want to depend on a default region being set, so instead we
	// will choose a region from the list of regions that are enabled by default
	// and use that to enumerate all enabled regions.
	// Corner case: user has intentionally disabled one or more regions that are
	// enabled by default. If that region is chosen, API calls will fail.
	// Therefore we retry until one of the regions works.
	regions, err := retryDescribeRegions()
	if err != nil {
		return nil, err
	}

	for _, region := range regions.Regions {
		regionNames = append(regionNames, awsgo.StringValue(region.RegionName))
	}

	return regionNames, nil
}

func getRandomRegion() (string, error) {
	allRegions, err := GetEnabledRegions()
	if err != nil {
		return "", errors.WithStackTrace(err)
	}
	rand.Seed(time.Now().UnixNano())
	randIndex := rand.Intn(len(allRegions))
	logging.Logger.Infof("Random region chosen: %s", allRegions[randIndex])
	return allRegions[randIndex], nil
}

func split(identifiers []string, limit int) [][]string {
	if limit < 0 {
		limit = -1 * limit
	} else if limit == 0 {
		return [][]string{identifiers}
	}

	var chunk []string
	chunks := make([][]string, 0, len(identifiers)/limit+1)
	for len(identifiers) >= limit {
		chunk, identifiers = identifiers[:limit], identifiers[limit:]
		chunks = append(chunks, chunk)
	}
	if len(identifiers) > 0 {
		chunks = append(chunks, identifiers[:len(identifiers)])
	}

	return chunks
}

// GetTargetRegions - Used enabled, selected and excluded regions to create a
// final list of valid regions
func GetTargetRegions(enabledRegions []string, selectedRegions []string, excludedRegions []string) ([]string, error) {
	if len(enabledRegions) == 0 {
		return nil, fmt.Errorf("Cannot have empty enabled regions")
	}

	// neither selectedRegions nor excludedRegions => select enabledRegions
	if len(selectedRegions) == 0 && len(excludedRegions) == 0 {
		return enabledRegions, nil
	}

	if len(selectedRegions) > 0 && len(excludedRegions) > 0 {
		return nil, fmt.Errorf("Cannot specify both selected and excluded regions")
	}

	var invalidRegions []string

	// Validate selectedRegions
	for _, selectedRegion := range selectedRegions {
		if !collections.ListContainsElement(enabledRegions, selectedRegion) {
			invalidRegions = append(invalidRegions, selectedRegion)
		}
	}
	if len(invalidRegions) > 0 {
		return nil, fmt.Errorf("Invalid values for region: [%s]", invalidRegions)
	}

	if len(selectedRegions) > 0 {
		return selectedRegions, nil
	}

	// Validate excludedRegions
	for _, excludedRegion := range excludedRegions {
		if !collections.ListContainsElement(enabledRegions, excludedRegion) {
			invalidRegions = append(invalidRegions, excludedRegion)
		}
	}
	if len(invalidRegions) > 0 {
		return nil, fmt.Errorf("Invalid values for exclude-region: [%s]", invalidRegions)
	}

	// Filter out excludedRegions from enabledRegions
	var targetRegions []string
	if len(excludedRegions) > 0 {
		for _, region := range enabledRegions {
			if !collections.ListContainsElement(excludedRegions, region) {
				targetRegions = append(targetRegions, region)
			}
		}
	}
	if len(targetRegions) == 0 {
		return nil, fmt.Errorf("Cannot exclude all regions: %s", excludedRegions)
	}
	return targetRegions, nil
}

// GetAllResources - Lists all aws resources
func GetAllResources(targetRegions []string, excludeAfter time.Time, resourceTypes []string, configObj config.Config) (*AwsAccountResources, error) {
	account := AwsAccountResources{
		Resources: make(map[string]AwsRegionResource),
	}

	count := 1
	totalRegions := len(targetRegions)
	var resourcesCache = map[string]map[string][]*string{}

	for _, region := range targetRegions {
		// The "global" region case is handled outside this loop
		if region == GlobalRegion {
			continue
		}

		logging.Logger.Infof("Checking region [%d/%d]: %s", count, totalRegions, region)

		session, err := session.NewSession(&awsgo.Config{
			Region: awsgo.String(region)},
		)

		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

		resourcesInRegion := AwsRegionResource{}

		// The order in which resources are nuked is important
		// because of dependencies between resources

		// ASG Names
		asGroups := ASGroups{}
		if IsNukeable(asGroups.ResourceName(), resourceTypes) {
			groupNames, err := getAllAutoScalingGroups(session, region, excludeAfter)
			if err != nil {
				return nil, errors.WithStackTrace(err)
			}
			if len(groupNames) > 0 {
				asGroups.GroupNames = awsgo.StringValueSlice(groupNames)
				resourcesInRegion.Resources = append(resourcesInRegion.Resources, asGroups)
			}
		}
		// End ASG Names

		// Launch Configuration Names
		configs := LaunchConfigs{}
		if IsNukeable(configs.ResourceName(), resourceTypes) {
			configNames, err := getAllLaunchConfigurations(session, region, excludeAfter)
			if err != nil {
				return nil, errors.WithStackTrace(err)
			}
			if len(configNames) > 0 {
				configs.LaunchConfigurationNames = awsgo.StringValueSlice(configNames)
				resourcesInRegion.Resources = append(resourcesInRegion.Resources, configs)
			}
		}
		// End Launch Configuration Names

		// LoadBalancer Names
		loadBalancers := LoadBalancers{}
		if IsNukeable(loadBalancers.ResourceName(), resourceTypes) {
			elbNames, err := getAllElbInstances(session, region, excludeAfter)
			if err != nil {
				return nil, errors.WithStackTrace(err)
			}
			if len(elbNames) > 0 {
				loadBalancers.Names = awsgo.StringValueSlice(elbNames)
				resourcesInRegion.Resources = append(resourcesInRegion.Resources, loadBalancers)
			}
		}
		// End LoadBalancer Names

		// LoadBalancerV2 Arns
		loadBalancersV2 := LoadBalancersV2{}
		if IsNukeable(loadBalancersV2.ResourceName(), resourceTypes) {
			elbv2Arns, err := getAllElbv2Instances(session, region, excludeAfter)
			if err != nil {
				return nil, errors.WithStackTrace(err)
			}
			if len(elbv2Arns) > 0 {
				loadBalancersV2.Arns = awsgo.StringValueSlice(elbv2Arns)
				resourcesInRegion.Resources = append(resourcesInRegion.Resources, loadBalancersV2)
			}
		}
		// End LoadBalancerV2 Arns

		// SQS Queues
		sqsQueue := SqsQueue{}
		if IsNukeable(sqsQueue.ResourceName(), resourceTypes) {
			queueUrls, err := getAllSqsQueue(session, region, excludeAfter)
			if err != nil {
				return nil, errors.WithStackTrace(err)
			}
			if len(queueUrls) > 0 {
				sqsQueue.QueueUrls = awsgo.StringValueSlice(queueUrls)
				resourcesInRegion.Resources = append(resourcesInRegion.Resources, sqsQueue)
			}
		}
		// End SQS Queue

		// TransitGatewayVpcAttachment
		transitGatewayVpcAttachments := TransitGatewaysVpcAttachment{}
		transitGatewayIsAvailable, err := tgIsAvailableInRegion(session, region)
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}
		if IsNukeable(transitGatewayVpcAttachments.ResourceName(), resourceTypes) && transitGatewayIsAvailable {
			transitGatewayVpcAttachmentIds, err := getAllTransitGatewayVpcAttachments(session, region, excludeAfter)
			if err != nil {
				return nil, errors.WithStackTrace(err)
			}
			if len(transitGatewayVpcAttachmentIds) > 0 {
				transitGatewayVpcAttachments.Ids = awsgo.StringValueSlice(transitGatewayVpcAttachmentIds)
				resourcesInRegion.Resources = append(resourcesInRegion.Resources, transitGatewayVpcAttachments)
			}
		}
		// End TransitGatewayVpcAttachment

		// TransitGatewayRouteTable
		transitGatewayRouteTables := TransitGatewaysRouteTables{}
		if IsNukeable(transitGatewayRouteTables.ResourceName(), resourceTypes) && transitGatewayIsAvailable {
			transitGatewayRouteTableIds, err := getAllTransitGatewayRouteTables(session, region, excludeAfter)
			if err != nil {
				return nil, errors.WithStackTrace(err)
			}
			if len(transitGatewayRouteTableIds) > 0 {
				transitGatewayRouteTables.Ids = awsgo.StringValueSlice(transitGatewayRouteTableIds)
				resourcesInRegion.Resources = append(resourcesInRegion.Resources, transitGatewayRouteTables)
			}
		}
		// End TransitGatewayRouteTable

		// TransitGateway
		transitGateways := TransitGateways{}
		if IsNukeable(transitGateways.ResourceName(), resourceTypes) && transitGatewayIsAvailable {
			transitGatewayIds, err := getAllTransitGatewayInstances(session, region, excludeAfter)
			if err != nil {
				return nil, errors.WithStackTrace(err)
			}
			if len(transitGatewayIds) > 0 {
				transitGateways.Ids = awsgo.StringValueSlice(transitGatewayIds)
				resourcesInRegion.Resources = append(resourcesInRegion.Resources, transitGateways)
			}
		}
		// End TransitGateway

		// NATGateway
		natGateways := NatGateways{}
		if IsNukeable(natGateways.ResourceName(), resourceTypes) {
			ngwIDs, err := getAllNatGateways(session, excludeAfter, configObj)
			if err != nil {
				return nil, errors.WithStackTrace(err)
			}
			if len(ngwIDs) > 0 {
				natGateways.NatGatewayIDs = awsgo.StringValueSlice(ngwIDs)
				resourcesInRegion.Resources = append(resourcesInRegion.Resources, natGateways)
			}
		}
		// End NATGateway

		// EC2 Instances
		ec2Instances := EC2Instances{}
		if IsNukeable(ec2Instances.ResourceName(), resourceTypes) {
			instanceIds, err := getAllEc2Instances(session, region, excludeAfter)
			if err != nil {
				return nil, errors.WithStackTrace(err)
			}
			if len(instanceIds) > 0 {
				ec2Instances.InstanceIds = awsgo.StringValueSlice(instanceIds)
				resourcesInRegion.Resources = append(resourcesInRegion.Resources, ec2Instances)
			}
		}
		// End EC2 Instances

		// EBS Volumes
		ebsVolumes := EBSVolumes{}
		if IsNukeable(ebsVolumes.ResourceName(), resourceTypes) {
			volumeIds, err := getAllEbsVolumes(session, region, excludeAfter)
			if err != nil {
				return nil, errors.WithStackTrace(err)
			}
			if len(volumeIds) > 0 {
				ebsVolumes.VolumeIds = awsgo.StringValueSlice(volumeIds)
				resourcesInRegion.Resources = append(resourcesInRegion.Resources, ebsVolumes)
			}
		}
		// End EBS Volumes

		// EIP Addresses
		eipAddresses := EIPAddresses{}
		if IsNukeable(eipAddresses.ResourceName(), resourceTypes) {
			allocationIds, err := getAllEIPAddresses(session, region, excludeAfter)
			if err != nil {
				return nil, errors.WithStackTrace(err)
			}
			if len(allocationIds) > 0 {
				eipAddresses.AllocationIds = awsgo.StringValueSlice(allocationIds)
				resourcesInRegion.Resources = append(resourcesInRegion.Resources, eipAddresses)
			}
		}
		// End EIP Addresses

		// AMIs
		amis := AMIs{}
		if IsNukeable(amis.ResourceName(), resourceTypes) {
			imageIds, err := getAllAMIs(session, region, excludeAfter)
			if err != nil {
				return nil, errors.WithStackTrace(err)
			}
			if len(imageIds) > 0 {
				amis.ImageIds = awsgo.StringValueSlice(imageIds)
				resourcesInRegion.Resources = append(resourcesInRegion.Resources, amis)
			}
		}
		// End AMIs

		// GuardDuty
		gdInstances := GuardDutyInstances{}
		if IsNukeable(gdInstances.ResourceName(), resourceTypes) {
			detectors, err := getAllGuardDutyDetectors(session, excludeAfter)
			if err != nil {
				return nil, errors.WithStackTrace(err)
			}
			if len(detectors) > 0 {
				gdInstances.DetectorIds = detectors
				resourcesInRegion.Resources = append(resourcesInRegion.Resources, gdInstances)
			}
		}
		// End GuardDuty

		// Snapshots
		snapshots := Snapshots{}
		if IsNukeable(snapshots.ResourceName(), resourceTypes) {
			snapshotIds, err := getAllSnapshots(session, region, excludeAfter)
			if err != nil {
				return nil, errors.WithStackTrace(err)
			}
			if len(snapshotIds) > 0 {
				snapshots.SnapshotIds = awsgo.StringValueSlice(snapshotIds)
				resourcesInRegion.Resources = append(resourcesInRegion.Resources, snapshots)
			}
		}
		// End Snapshots

		// ECS resources
		ecsServices := ECSServices{}
		if IsNukeable(ecsServices.ResourceName(), resourceTypes) {
			clusterArns, err := getAllEcsClusters(session)
			if err != nil {
				return nil, errors.WithStackTrace(err)
			}
			if len(clusterArns) > 0 {
				serviceArns, serviceClusterMap, err := getAllEcsServices(session, clusterArns, excludeAfter)
				if err != nil {
					return nil, errors.WithStackTrace(err)
				}
				ecsServices.Services = awsgo.StringValueSlice(serviceArns)
				ecsServices.ServiceClusterMap = serviceClusterMap
				resourcesInRegion.Resources = append(resourcesInRegion.Resources, ecsServices)
			}
		}

		ecsClusters := ECSClusters{}
		if IsNukeable(ecsClusters.ResourceName(), resourceTypes) {
			ecsClusterArns, err := getAllEcsClustersOlderThan(session, region, excludeAfter)
			if err != nil {
				return nil, errors.WithStackTrace(err)
			}
			if len(ecsClusterArns) > 0 {
				ecsClusters.ClusterArns = awsgo.StringValueSlice(ecsClusterArns)
				resourcesInRegion.Resources = append(resourcesInRegion.Resources, ecsClusters)
			}
		}
		// End ECS resources

		// EKS resources
		eksClusters := EKSClusters{}
		if IsNukeable(eksClusters.ResourceName(), resourceTypes) {
			if eksSupportedRegion(region) {
				eksClusterNames, err := getAllEksClusters(session, excludeAfter)
				if err != nil {
					return nil, errors.WithStackTrace(err)
				}
				if len(eksClusterNames) > 0 {
					eksClusters.Clusters = awsgo.StringValueSlice(eksClusterNames)
					resourcesInRegion.Resources = append(resourcesInRegion.Resources, eksClusters)
				}
			}
		}
		// End EKS resources

		// RDS DB Instances
		dbInstances := DBInstances{}
		if IsNukeable(dbInstances.ResourceName(), resourceTypes) {
			instanceNames, err := getAllRdsInstances(session, excludeAfter)

			if err != nil {
				return nil, errors.WithStackTrace(err)
			}

			if len(instanceNames) > 0 {
				dbInstances.InstanceNames = awsgo.StringValueSlice(instanceNames)
				resourcesInRegion.Resources = append(resourcesInRegion.Resources, dbInstances)
			}
		}
		// End RDS DB Instances

		// RDS DB Clusters
		// These reference the Aurora Clusters, for the use it's the same resource (rds), but AWS
		// has different abstractions for each.
		dbClusters := DBClusters{}
		if IsNukeable(dbClusters.ResourceName(), resourceTypes) {
			clustersNames, err := getAllRdsClusters(session, excludeAfter)

			if err != nil {
				return nil, errors.WithStackTrace(err)
			}

			if len(clustersNames) > 0 {
				dbClusters.InstanceNames = awsgo.StringValueSlice(clustersNames)
				resourcesInRegion.Resources = append(resourcesInRegion.Resources, dbClusters)
			}
		}
		// End RDS DB Clusters

		// Lambda Functions
		lambdaFunctions := LambdaFunctions{}
		if IsNukeable(lambdaFunctions.ResourceName(), resourceTypes) {
			lambdaFunctionNames, err := getAllLambdaFunctions(session, excludeAfter)

			if err != nil {
				return nil, errors.WithStackTrace(err)
			}

			if len(lambdaFunctionNames) > 0 {
				lambdaFunctions.LambdaFunctionNames = awsgo.StringValueSlice(lambdaFunctionNames)
				resourcesInRegion.Resources = append(resourcesInRegion.Resources, lambdaFunctions)
			}
		}
		// End Lambda Functions

		// Secrets Manager Secrets
		secretsManagerSecrets := SecretsManagerSecrets{}
		if IsNukeable(secretsManagerSecrets.ResourceName(), resourceTypes) {
			secrets, err := getAllSecretsManagerSecrets(session, excludeAfter, configObj)
			if err != nil {
				return nil, errors.WithStackTrace(err)
			}

			if len(secrets) > 0 {
				secretsManagerSecrets.SecretIDs = awsgo.StringValueSlice(secrets)
				resourcesInRegion.Resources = append(resourcesInRegion.Resources, secretsManagerSecrets)
			}
		}
		// End Secrets Manager Secrets

		// S3 Buckets
		s3Buckets := S3Buckets{}
		if IsNukeable(s3Buckets.ResourceName(), resourceTypes) {
			var bucketNamesPerRegion map[string][]*string

			// AWS S3 buckets list operation lists all buckets irrespective of regions.
			// For each bucket we have to make a separate call to find the bucket region.
			// Hence for x buckets and a total of y target regions - we need to make:
			// (x + 1) * y calls i.e. 1 call to list all x buckets, x calls to find out
			// each bucket's region and repeat the process for each of the y regions.

			// getAllS3Buckets returns a map of regions to buckets and we call it only once -
			// thereby reducing total calls from (x + 1) * y to only (x + 1) for the first region -
			// followed by a cache lookup for rest of the regions.

			// Cache lookup to check if we already obtained bucket names per region
			bucketNamesPerRegion, ok := resourcesCache["S3"]

			if !ok {
				bucketNamesPerRegion, err = getAllS3Buckets(session, excludeAfter, targetRegions, "", s3Buckets.MaxConcurrentGetSize(), configObj)
				if err != nil {
					return nil, errors.WithStackTrace(err)
				}

				resourcesCache["S3"] = make(map[string][]*string)

				for bucketRegion, _ := range bucketNamesPerRegion {
					resourcesCache["S3"][bucketRegion] = bucketNamesPerRegion[bucketRegion]
				}
			}

			bucketNames, ok := resourcesCache["S3"][region]

			if ok && len(bucketNames) > 0 {
				s3Buckets.Names = aws.StringValueSlice(bucketNames)
				resourcesInRegion.Resources = append(resourcesInRegion.Resources, s3Buckets)
			}
		}
		// End S3 Buckets

		if len(resourcesInRegion.Resources) > 0 {
			account.Resources[region] = resourcesInRegion
		}
		count++
	}

	// Global Resources - These resources are global and do not belong to a specific region
	// Only process them if the global region was not explicitly excluded
	if collections.ListContainsElement(targetRegions, GlobalRegion) {
		logging.Logger.Infof("Checking region [%d/%d]: %s", count, totalRegions, GlobalRegion)

		// As there is no actual region named global we have to pick a valid one just to create the session
		sessionRegion := defaultRegion
		session, err := session.NewSession(&awsgo.Config{
			Region: awsgo.String(sessionRegion)},
		)
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

		globalResources := AwsRegionResource{}

		// IAM Users
		iamUsers := IAMUsers{}
		if IsNukeable(iamUsers.ResourceName(), resourceTypes) {
			userNames, err := getAllIamUsers(session, excludeAfter, configObj)

			if err != nil {
				return nil, errors.WithStackTrace(err)
			}
			if len(userNames) > 0 {
				iamUsers.UserNames = awsgo.StringValueSlice(userNames)
				globalResources.Resources = append(globalResources.Resources, iamUsers)
			}
		}
		// End IAM Users

		if len(globalResources.Resources) > 0 {
			account.Resources[GlobalRegion] = globalResources
		}
	}

	return &account, nil
}

// ListResourceTypes - Returns list of resources which can be passed to --resource-type
func ListResourceTypes() []string {
	resourceTypes := []string{
		ASGroups{}.ResourceName(),
		LaunchConfigs{}.ResourceName(),
		LoadBalancers{}.ResourceName(),
		LoadBalancersV2{}.ResourceName(),
		SqsQueue{}.ResourceName(),
		TransitGatewaysVpcAttachment{}.ResourceName(),
		TransitGatewaysRouteTables{}.ResourceName(),
		TransitGateways{}.ResourceName(),
		EC2Instances{}.ResourceName(),
		EBSVolumes{}.ResourceName(),
		EIPAddresses{}.ResourceName(),
		AMIs{}.ResourceName(),
		Snapshots{}.ResourceName(),
		ECSClusters{}.ResourceName(),
		ECSServices{}.ResourceName(),
		EKSClusters{}.ResourceName(),
		DBInstances{}.ResourceName(),
		LambdaFunctions{}.ResourceName(),
		S3Buckets{}.ResourceName(),
		IAMUsers{}.ResourceName(),
		SecretsManagerSecrets{}.ResourceName(),
		NatGateways{}.ResourceName(),
		GuardDutyInstances{}.ResourceName(),
	}
	sort.Strings(resourceTypes)
	return resourceTypes
}

// IsValidResourceType - Checks if a resourceType is valid or not
func IsValidResourceType(resourceType string, allResourceTypes []string) bool {
	return collections.ListContainsElement(allResourceTypes, resourceType)
}

// IsNukeable - Checks if we should nuke a resource or not
func IsNukeable(resourceType string, resourceTypes []string) bool {
	if len(resourceTypes) == 0 ||
		collections.ListContainsElement(resourceTypes, "all") ||
		collections.ListContainsElement(resourceTypes, resourceType) {
		return true
	}
	return false
}

func nukeAllResourcesInRegion(account *AwsAccountResources, region string, session *session.Session) error {
	resourcesInRegion := account.Resources[region]

	for _, resources := range resourcesInRegion.Resources {
		length := len(resources.ResourceIdentifiers())

		// Split api calls into batches
		logging.Logger.Infof("Terminating %d resources in batches", length)
		batches := split(resources.ResourceIdentifiers(), resources.MaxBatchSize())

		for i := 0; i < len(batches); i++ {
			batch := batches[i]
			if err := resources.Nuke(session, batch); err != nil {
				// TODO: Figure out actual error type
				if strings.Contains(err.Error(), "RequestLimitExceeded") {
					logging.Logger.Info("Request limit reached. Waiting 1 minute before making new requests")
					time.Sleep(1 * time.Minute)
					continue
				}

				return errors.WithStackTrace(err)
			}

			if i != len(batches)-1 {
				logging.Logger.Info("Sleeping for 10 seconds before processing next batch...")
				time.Sleep(10 * time.Second)
			}
		}
	}

	return nil
}

// NukeAllResources - Nukes all aws resources
func NukeAllResources(account *AwsAccountResources, regions []string) error {
	for _, region := range regions {
		// region that will be used to create a session
		sessionRegion := region

		// As there is no actual region named global we have to pick a valid one just to create the session
		if region == GlobalRegion {
			sessionRegion = defaultRegion
		}

		session, err := session.NewSession(&awsgo.Config{
			Region: awsgo.String(sessionRegion)},
		)

		if err != nil {
			return errors.WithStackTrace(err)
		}

		err = nukeAllResourcesInRegion(account, region, session)

		if err != nil {
			return errors.WithStackTrace(err)
		}
	}

	return nil
}
