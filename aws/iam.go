package aws

import (
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/gruntwork-io/cloud-nuke/config"
)

func getAllIamRoles(session *session.Session, input *iam.ListRolesInput) ([]*string, error) {}

func excludeServiceIamRoles([]string) []string {}

func nukeAllIamRoles(session *session.Session, list []string, configObj config.Config) error {}
