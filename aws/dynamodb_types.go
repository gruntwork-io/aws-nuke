package aws

import (
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/gruntwork-io/gruntwork-cli/errors"
)

type DynamoDB struct {
	DynamoTableNames []string
}

func (tables DynamoDB) ResourceName() string {
	return "dynamodb"
}

func (tables DynamoDB) ResourceIdentifiers() []string {
	return tables.DynamoTableNames
}

func (table DynamoDB) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle
	return 50
}

// Nuke - nuke all Dynamo DB Tables
func (tables DynamoDB) Nuke(awsSession *session.Session, identifiers []string) error {
	if err := nukeAllDynamoDBTables(awsSession, awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}
	return nil
}
