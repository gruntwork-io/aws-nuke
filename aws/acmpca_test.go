package aws

import (
	"testing"
	"time"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/acmpca"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/gruntwork-io/go-commons/errors"
	"github.com/stretchr/testify/assert"
)

// createTestACMPCA will create am ACMPCA and return its ARN.
func createTestACMPCA(t *testing.T, session *session.Session, name string) *string {
	svc := acmpca.New(session)
	ca, err := svc.CreateCertificateAuthority(&acmpca.CreateCertificateAuthorityInput{
		CertificateAuthorityConfiguration: &acmpca.CertificateAuthorityConfiguration{
			KeyAlgorithm:     awsgo.String(acmpca.KeyAlgorithmRsa2048),
			SigningAlgorithm: awsgo.String(acmpca.SigningAlgorithmSha256withrsa),
			Subject: &acmpca.ASN1Subject{
				CommonName: awsgo.String(name),
			},
		},
		CertificateAuthorityType: awsgo.String("ROOT"),
		Tags: []*acmpca.Tag{
			{
				Key:   awsgo.String("Name"),
				Value: awsgo.String(name),
			},
		},
	})
	if err != nil {
		assert.Failf(t, "Could not create ACMPCA", errors.WithStackTrace(err).Error())
	}
	return ca.CertificateAuthorityArn
}

func TestListACMPCA(t *testing.T) {
	t.Parallel()

	region, err := getRandomRegion()
	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}
	session, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region)},
	)

	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	uniqueTestID := "cloud-nuke-test-" + util.UniqueID()
	arn := createTestACMPCA(t, session, uniqueTestID)
	// clean up after this test
	defer nukeAllACMPCA(session, []*string{arn})

	newARNs, err := getAllACMPCA(session, region, time.Now().Add(1*time.Hour*-1))
	if err != nil {
		assert.Fail(t, "Unable to fetch list of ACMPCA arns")
	}
	assert.NotContains(t, awsgo.StringValueSlice(newARNs), awsgo.StringValue(arn))

	allARNs, err := getAllACMPCA(session, region, time.Now().Add(1*time.Hour))
	if err != nil {
		assert.Fail(t, "Unable to fetch list of ACMPCA arns")
	}

	assert.Contains(t, awsgo.StringValueSlice(allARNs), awsgo.StringValue(arn))
}

func TestNukeACMPCA(t *testing.T) {
	t.Parallel()

	region, err := getRandomRegion()
	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	session, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region)},
	)

	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	uniqueTestID := "cloud-nuke-test-" + util.UniqueID()
	arn := createTestACMPCA(t, session, uniqueTestID)

	if err := nukeAllACMPCA(session, []*string{arn}); err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	arns, err := getAllACMPCA(session, region, time.Now().Add(1*time.Hour))
	if err != nil {
		assert.Fail(t, "Unable to fetch list of ACMPCA arns")
	}

	assert.NotContains(t, awsgo.StringValueSlice(arns), awsgo.StringValue(arn))
}
