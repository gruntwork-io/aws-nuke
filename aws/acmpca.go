package aws

import (
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/acmpca"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/go-commons/errors"
	"github.com/hashicorp/go-multierror"
)

// getAllACMPCA returns a list of all arns of ACMPCA, which can be deleted.
func getAllACMPCA(session *session.Session, region string, excludeAfter time.Time) ([]*string, error) {
	svc := acmpca.New(session)
	var arns []*string
	if paginationErr := svc.ListCertificateAuthoritiesPages(&acmpca.ListCertificateAuthoritiesInput{}, func(p *acmpca.ListCertificateAuthoritiesOutput, lastPage bool) bool {
		for _, ca := range p.CertificateAuthorities {
			// one can only delete CAs if they are 'ACTIVE' or 'DISABLED'
			statusSafe := aws.StringValue(ca.Status)
			isCandidateForDeletion := statusSafe == acmpca.CertificateAuthorityStatusActive || statusSafe == acmpca.CertificateAuthorityStatusDisabled
			if !isCandidateForDeletion {
				continue
			}
			if excludeAfter.After(aws.TimeValue(ca.CreatedAt)) {
				arns = append(arns, ca.Arn)
			}
		}
		return true
	}); paginationErr != nil {
		return nil, errors.WithStackTrace(paginationErr)
	}
	return arns, nil
}

// nukeAllACMPCA will delete all ACMPCA, which are given by a list of arns.
func nukeAllACMPCA(session *session.Session, arns []*string) error {
	if len(arns) == 0 {
		logging.Logger.Infof("No ACMPCA to nuke in region %s", *session.Config.Region)
		return nil
	}
	svc := acmpca.New(session)

	logging.Logger.Infof("Deleting all ACMPCA in region %s", *session.Config.Region)
	// There is no bulk delete acmpca API, so we delete the batch of ARNs concurrently using go routines.
	wg := new(sync.WaitGroup)
	wg.Add(len(arns))
	errChans := make([]chan error, len(arns))
	for i, arn := range arns {
		errChans[i] = make(chan error, 1)
		go deleteACMPCAASync(wg, errChans[i], svc, arn, aws.StringValue(session.Config.Region))
	}
	wg.Wait()

	// Collect all the errors from the async delete calls into a single error struct.
	var allErrs *multierror.Error
	for _, errChan := range errChans {
		if err := <-errChan; err != nil {
			allErrs = multierror.Append(allErrs, err)
			logging.Logger.Errorf("[Failed] %s", err)
		}
	}
	return errors.WithStackTrace(allErrs.ErrorOrNil())
}

// deleteACMPCAASync deletes the provided ACMPCA arn. Intended to be run in a goroutine, using wait groups
// and a return channel for errors.
func deleteACMPCAASync(wg *sync.WaitGroup, errChan chan error, svc *acmpca.ACMPCA, arn *string, region string) {
	defer wg.Done()

	logging.Logger.Infof("Setting status to 'DISABLED' for ACMPCA %s in region %s", *arn, region)
	if _, updateStatusErr := svc.UpdateCertificateAuthority(&acmpca.UpdateCertificateAuthorityInput{
		CertificateAuthorityArn: arn,
		Status:                  aws.String(acmpca.CertificateAuthorityStatusDisabled),
	}); updateStatusErr != nil {
		errChan <- updateStatusErr
		return
	}
	logging.Logger.Infof("Did set status to 'DISABLED' for ACMPCA: %s in region %s", *arn, region)

	if _, deleteErr := svc.DeleteCertificateAuthority(&acmpca.DeleteCertificateAuthorityInput{
		CertificateAuthorityArn: arn,
		// the range is 7 to 30 days.
		// since cloud-nuke should not be used in production,
		// we assume that the minimum (7 days) is fine.
		PermanentDeletionTimeInDays: aws.Int64(7),
	}); deleteErr != nil {
		errChan <- deleteErr
		return
	}
	logging.Logger.Infof("Deleted ACMPCA: %s", *arn)
}
