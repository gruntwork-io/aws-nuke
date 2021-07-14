package aws

import (
	"os"
	"testing"
	"time"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/acmpca"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/gruntwork-io/go-commons/errors"
	"github.com/stretchr/testify/assert"
)

// enableACMPCAExpensiveEnv is used to control whether to run
// the following test or not. The idea is that the test are disabled
// per default and one has to opt-in to enable the test as creating
// and destroying a ACM PCA is expensive.
// Upper bound, worst case: $400 / month per single CA create/delete.
const enableACMPCAExpensiveEnv = "TEST_ACMPCA_EXPENSIVE_ENABLE"

// runOrSkip decides whether to run or skip the test depending
// whether the env-var `TEST_ACMPCA_EXPENSIVE_ENABLE` is set or not.
func runOrSkip(t *testing.T) {
	if _, isSet := os.LookupEnv(enableACMPCAExpensiveEnv); !isSet {
		t.Skipf("Skipping the integration test for acmpca. Set the env-var '%s' to enable this expensive test.", enableACMPCAExpensiveEnv)
	}
}

// createTestACMPCA will create am ACMPCA and return its ARN.
func createTestACMPCA(t *testing.T, session *session.Session, name string) *string {
	// As an additional safety guard, we are adding another check here
	// to decide whether to run the test or not.
	runOrSkip(t)

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
	runOrSkip(t)
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

	newARNs, err := getAllACMPCA(session, region, time.Now().Add(1*time.Hour*-1), config.Config{})
	if err != nil {
		assert.Fail(t, "Unable to fetch list of ACMPCA arns")
	}
	assert.NotContains(t, awsgo.StringValueSlice(newARNs), awsgo.StringValue(arn))

	allARNs, err := getAllACMPCA(session, region, time.Now().Add(1*time.Hour), config.Config{})
	if err != nil {
		assert.Fail(t, "Unable to fetch list of ACMPCA arns")
	}

	assert.Contains(t, awsgo.StringValueSlice(allARNs), awsgo.StringValue(arn))
}

func TestNukeACMPCA(t *testing.T) {
	runOrSkip(t)
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

	arns, err := getAllACMPCA(session, region, time.Now().Add(1*time.Hour), config.Config{})
	if err != nil {
		assert.Fail(t, "Unable to fetch list of ACMPCA arns")
	}

	assert.NotContains(t, awsgo.StringValueSlice(arns), awsgo.StringValue(arn))
}
