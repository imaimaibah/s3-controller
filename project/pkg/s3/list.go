package s3

import (
	"encoding/json"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/sts"
	"io/ioutil"
	"log"
	"os"
)

func createSession() (*session.Session, error) {
	// Session creation
	sess := session.Must(session.NewSession())

	// Create a new STS client to get temporary credentials
	initStsClient := sts.New(sess)

	// Getting the SA token
	awsWebIdentityTokenFile := os.Getenv("AWS_WEB_IDENTITY_TOKEN_FILE")

	if awsWebIdentityTokenFile != "" {
		log.Println("Using assumerole with web identity")
		awsRoleArn := os.Getenv("AWS_ROLE_ARN")
		awsWebIdentityToken, err := ioutil.ReadFile(awsWebIdentityTokenFile)
		if err != nil {
			return nil, err
		}

		// Requesting temporary credentials
		identity, err := initStsClient.AssumeRoleWithWebIdentity(
			&sts.AssumeRoleWithWebIdentityInput{
				RoleArn:          aws.String(awsRoleArn),
				RoleSessionName:  aws.String("sedex-s3"),
				WebIdentityToken: aws.String(string(awsWebIdentityToken)),
				DurationSeconds:  aws.Int64(3600),
			})
		if err != nil {
			return nil, err
		}

		// Creating a new session with the temporary credentials
		sess = session.Must(session.NewSession(&aws.Config{
			Credentials: credentials.NewStaticCredentialsFromCreds(credentials.Value{
				AccessKeyID:     *identity.Credentials.AccessKeyId,
				SecretAccessKey: *identity.Credentials.SecretAccessKey,
				SessionToken:    *identity.Credentials.SessionToken,
				ProviderName:    "AssumeRoleWithWebIdentity",
			}),
		}))
	}

	// Create a new sts client from IAM role's credentials and print the current identity
	stsClient := sts.New(sess)
	identity, err := stsClient.GetCallerIdentity(&sts.GetCallerIdentityInput{})
	if err != nil {
		return nil, err
	}
	jsonIdentity, err := json.MarshalIndent(*identity, "", "  ")
	log.Printf("%s", string(jsonIdentity))

	return sess, nil

}

func Create(name string) error {
	// Create session
	sess, err := createSession()
	if err != nil {
		log.Printf("Couldn't create a session")
		return err
	}

	// Create a new S3 client and print all buckets
	s3Client := s3.New(sess)
	obj, err := s3Client.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(name),
		Key:    aws.String("//"),
	})
	if err != nil {
		_, err = s3Client.CreateBucket(&s3.CreateBucketInput{
			Bucket: aws.String(name),
			CreateBucketConfiguration: &s3.CreateBucketConfiguration{
				LocationConstraint: aws.String("eu-west-2"),
			},
		})
		if err != nil {
			log.Printf("Unable to create bucket %q", name)
			return err
		}
	}

	//DELETEBELOW
	jsonBuckets, err := json.MarshalIndent(*obj, "", "  ")
	log.Printf("%+v", string(jsonBuckets))
	return nil
}

func Update(name string, version string, encryption string) error {
	// Create session
	sess, err := createSession()
	if err != nil {
		log.Printf("Couldn't create a session")
		return err
	}
	s3Client := s3.New(sess)

	// Update Bucket
	// Versioning
	if version == "true" {
		verInput := &s3.PutBucketVersioningInput{
			Bucket: aws.String(name),
			VersioningConfiguration: &s3.VersioningConfiguration{
				MFADelete: aws.String("Disabled"),
				Status:    aws.String("Enabled"),
			},
		}
		_, err := s3Client.PutBucketVersioning(verInput)
		if err != nil {
			return err
		}
	}

	// Encryption
	if encryption == "true" {
		rule := &s3.ServerSideEncryptionRule{
			ApplyServerSideEncryptionByDefault: &s3.ServerSideEncryptionByDefault{
				SSEAlgorithm: aws.String("AES256"),
			},
		}
		rules := []*s3.ServerSideEncryptionRule{rule}

		serverConfig := &s3.ServerSideEncryptionConfiguration{
			Rules: rules,
		}

		input := &s3.PutBucketEncryptionInput{
			Bucket:                            aws.String(name),
			ServerSideEncryptionConfiguration: serverConfig,
		}
		_, err := s3Client.PutBucketEncryption(input)
		if err != nil {
			return err
		}
	}
	return nil
}

func Delete(name string) error {
	// Create session
	sess, err := createSession()
	if err != nil {
		log.Printf("Couldn't create a session")
		return err
	}
	s3Client := s3.New(sess)

	// Delete bucket
	if _, err = s3Client.DeleteBucket(&s3.DeleteBucketInput{
		Bucket: aws.String(name),
	}); err != nil {
		return err
	}

	// Wait until bucket is deleted before finishing
	if err = s3Client.WaitUntilBucketNotExists(&s3.HeadBucketInput{
		Bucket: aws.String(name),
	}); err != nil {
		return err
	}

	return nil
}
