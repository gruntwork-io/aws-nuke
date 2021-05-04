package aws

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/gruntwork-io/gruntwork-cli/errors"
	"github.com/gruntwork-io/terratest/modules/random"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)


func createTestDynamoTables(t *testing.T)   {
	awsSession := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))
	tables := 0
	var tablesToCreate []string
	// Creates 20 tables to simulate a smaller environment. We can make this dynamic if needed.
	for tables <= 20 {

		tablesToCreate = append(tablesToCreate, "dynamo-nuke-test-" + random.UniqueId() )
		tables +=1

	}


	svc := dynamodb.New(awsSession)

	for num, table := range tablesToCreate {
		input := &dynamodb.CreateTableInput{
			TableName: &table,
			AttributeDefinitions: []*dynamodb.AttributeDefinition{
				{
					AttributeName: aws.String("Nuke" + string(rune(num))),
					AttributeType: aws.String("S"),
				},
				{
					AttributeName: aws.String("TypeofNuke" + string(rune(num))),
					AttributeType: aws.String("S"),
				},
			},
			KeySchema: []*dynamodb.KeySchemaElement{
				{
					AttributeName: aws.String("Nuke" + string(rune(num)) ),
					KeyType:       aws.String("HASH"),
				},
				{
					AttributeName: aws.String("TypeofNuke" + string(rune(num))),
					KeyType:       aws.String("RANGE"),
				},
			},
			ProvisionedThroughput: &dynamodb.ProvisionedThroughput{
				ReadCapacityUnits:  aws.Int64(1),
				WriteCapacityUnits: aws.Int64(1),
			},

		}
		_, err := svc.CreateTable(input)
		time.Sleep(2 * time.Second)
		require.NoError(t, err)
	}

}

func TestGetTables(t *testing.T) {
	t.Parallel()
	region, err := getRandomRegion()
	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}
	
	awsSession, err := session.NewSession(&aws.Config{
		Region: &region,
	})
	if err != nil {
		require.Error(t, err)
	}
	getAllDynamoTables(awsSession, time.Now().Add(1*time.Hour*-1))
}

func TestNukeAllDynamoDBTables(t *testing.T) {
	t.Parallel()
	awsSession := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	createTestDynamoTables(t)

	tables, err := getAllDynamoTables(awsSession, time.Now().Add(1*time.Hour*-1))
	if err != nil {
		require.Error(t, err)
		}
		// Dynamo needs a wait of about 2 seconds to create the tables before delete
		time.Sleep(2 * time.Second)
	nukeErr := nukeAllDynamoDBTables(awsSession, tables)
	require.NoError(t, nukeErr)
}
