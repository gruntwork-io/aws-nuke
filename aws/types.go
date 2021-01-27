package aws

import (
	"github.com/aws/aws-sdk-go/aws/session"
)

const AwsResourceExclusionTagKey = "cloud-nuke-excluded"

type AwsAccountResources struct {
	RegionResources    map[string]AwsRegionResource
	NonRegionResources []AwsResources
}

type AwsResources interface {
	ResourceName() string
	ResourceIdentifiers() []string
	MaxBatchSize() int
	Nuke(session *session.Session, identifiers []string) error
}

type AwsRegionResource struct {
	Resources []AwsResources
}
