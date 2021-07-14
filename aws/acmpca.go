package aws

import (
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/acmpca"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/go-commons/errors"
)

// getAllACMPCA returns a list of all arns of ACMPCA, which can be deleted.
func getAllACMPCA(session *session.Session, region string, excludeAfter time.Time) ([]*string, error) {
	svc := acmpca.New(session)

	result, err := svc.ListCertificateAuthorities(&acmpca.ListCertificateAuthoritiesInput{})
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	var arns []*string
	for _, ca := range result.CertificateAuthorities {
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
	var deletedARNs []*string

	for _, arn := range arns {
		logging.Logger.Infof("Setting status to 'DISABLED' for ACMPCA %s in region %s", *arn, *session.Config.Region)
		if _, updateStatusErr := svc.UpdateCertificateAuthority(&acmpca.UpdateCertificateAuthorityInput{
			CertificateAuthorityArn: arn,
			Status:                  aws.String(acmpca.CertificateAuthorityStatusDisabled),
		}); updateStatusErr != nil {
			logging.Logger.Errorf("[Failed] %s", updateStatusErr)
			continue
		}
		logging.Logger.Infof("Did set status to 'DISABLED' for ACMPCA: %s in region %s", *arn, *session.Config.Region)

		if _, deleteErr := svc.DeleteCertificateAuthority(&acmpca.DeleteCertificateAuthorityInput{
			CertificateAuthorityArn: arn,
			// the range is 7 to 30 days.
			// since cloud-nuke should not be used in production,
			// we assume that the minimum (7 days) is fine.
			PermanentDeletionTimeInDays: aws.Int64(7),
		}); deleteErr != nil {
			logging.Logger.Errorf("[Failed] %s", deleteErr)
			continue
		}
		deletedARNs = append(deletedARNs, arn)
		logging.Logger.Infof("Deleted ACMPCA: %s", *arn)
	}
	logging.Logger.Infof("[OK] %d ACMPCA(s) deleted in %s", len(arns), *session.Config.Region)
	return nil
}
