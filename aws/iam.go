package aws

import (
	"regexp"
	"time"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/gruntwork-cli/errors"
)

func getAllIAMRoles(
	session *session.Session,
	excludeAfter time.Time,
	configObj config.Config,
) ([]*string, error) {
	svc := iam.New(session)

	result, err := svc.ListRoles(&iam.ListRolesInput{})

	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	var names []*string

	for _, role := range result.Roles {
		if role.CreateDate != nil && excludeAfter.After(awsgo.TimeValue(role.CreateDate)) {

			// Check if the role name matches config file rules
			if shouldIncludeRole(*role.RoleName, configObj.IAMRole.IncludeRule.NamesRE, configObj.IAMRole.ExcludeRule.NamesRE) {
				names = append(names, role.RoleName)
			}
		}
	}

	return names, nil
}

func shouldIncludeRole(roleName string, includeNamesREList []*regexp.Regexp, excludeNamesREList []*regexp.Regexp) bool {
	shouldInclude := false

	if len(includeNamesREList) > 0 {
		// If any include rules are specified,
		// only check to see if an exclude rule matches when an include rule matches the bucket
		if includeRoleByREList(roleName, includeNamesREList) {
			shouldInclude = excludeBucketByREList(roleName, excludeNamesREList)
		}
	} else if len(excludeNamesREList) > 0 {
		// Only check to see if an exclude rule matches when there are no include rules defined
		shouldInclude = excludeRoleByREList(roleName, excludeNamesREList)
	} else {
		shouldInclude = true
	}

	return shouldInclude
}

func includeRoleByREList(roleName string, reList []*regexp.Regexp) bool {
	for _, re := range reList {
		if re.MatchString(roleName) {
			return true
		}
	}
	return false
}

func excludeRoleByREList(roleName string, reList []*regexp.Regexp) bool {
	for _, re := range reList {
		if re.MatchString(roleName) {
			return false
		}
	}

	return true
}

func nukeAllIAMRoles(session *session.Session, names []*string) error {
	svc := iam.New(session)

	if len(names) == 0 {
		logging.Logger.Info("No IAM Roles to nuke.")
		return nil
	}

	logging.Logger.Info("Deleting all IAM Roles.")
	deletedNames := []*string{}

	for _, name := range names {
		// Nuke Policies first
		err := nukeRolePolicies(session, name)

		if err != nil {
			logging.Logger.Errorf("[Nuking role policies failed] %s: %s", awsgo.StringValue(name), err)
			return errors.WithStackTrace(err)
		}

		// Nuke Role
		params := &iam.DeleteRoleInput{
			RoleName: name,
		}

		_, err = svc.DeleteRole(params)

		if err != nil {
			logging.Logger.Errorf("[Nuking role failed] %s: %s", awsgo.StringValue(name), err)
			return errors.WithStackTrace(err)

		} else {
			deletedNames = append(deletedNames, name)
			logging.Logger.Infof("Deleted IAM Role: %s", awsgo.StringValue(name))
		}
	}

	logging.Logger.Infof("[OK] %d IAM Role(s) deleted.", len(deletedNames))
	return nil
}

func nukeRolePolicies(session *session.Session, name *string) error {
	svc := iam.New(session)

	// ListAttachedRolePolicies lists all managed policies attached to the role.
	// If you find managed policies, only detach them, don't delete them.
	paramsForManagedPolicies := &iam.ListAttachedRolePoliciesInput{
		RoleName: name,
	}

	resultManaged, err := svc.ListAttachedRolePolicies(paramsForManagedPolicies)

	for _, policy := range resultManaged.AttachedPolicies {

		paramsForDetachRolePolicies := &iam.DetachRolePolicyInput{
			PolicyArn: policy.PolicyArn,
			RoleName:  name,
		}

		_, err = svc.DetachRolePolicy(paramsForDetachRolePolicies)

		if err != nil {
			logging.Logger.Errorf("[Detaching role policy failed] %s: %s", awsgo.StringValue(policy.PolicyName), err)
			return errors.WithStackTrace(err)
		}
	}

	// ListRolePolicies lists all inline policies attached to the role.
	// If you find inline policies, detach and delete them.
	paramsForInlinePolicies := &iam.ListRolePoliciesInput{
		RoleName: name,
	}

	resultInline, err := svc.ListRolePolicies(paramsForInlinePolicies)

	for _, policy := range resultInline.PolicyNames {
		params := &iam.DeleteRolePolicyInput{
			PolicyName: policy,
			RoleName:   name,
		}

		_, err = svc.DeleteRolePolicy(params)

		if err != nil {
			logging.Logger.Errorf("[Deleting inline policy failed] %s: %s", awsgo.StringValue(policy), err)
			return errors.WithStackTrace(err)
		}
	}

	if err != nil {
		logging.Logger.Errorf("[Failed] %s: %s", awsgo.StringValue(name), err)
		// TODO: keep handling errors...
	}

	return nil
}
