package aws

import (
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/gruntwork-io/go-commons/errors"
)

// ACMPA - represents all ACMPA
type ACMPCA struct {
	ARNs []string
}

// ResourceName - the simple name of the aws resource
func (ca ACMPCA) ResourceName() string {
	return "acmpca"
}

// ResourceIdentifiers - The volume ids of the ebs volumes
func (ca ACMPCA) ResourceIdentifiers() []string {
	return ca.ARNs
}

func (ca ACMPCA) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle
	return 10
}

// Nuke - nuke 'em all!!!
func (ca ACMPCA) Nuke(session *session.Session, arns []string) error {
	if err := nukeAllACMPCA(session, awsgo.StringSlice(arns)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
