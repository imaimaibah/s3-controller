package s3

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/aws/aws-sdk-go/service/sts"
	"io/ioutil"
	"log"
	"os"
)

type S3 struct {
	sess *session.Session
}

func (s *S3) CreateSession() error {
	// Session creation
	s.sess = session.Must(session.NewSession())

	// Create a new STS client to get temporary credentials
	initStsClient := sts.New(s.sess)

	// Getting the SA token
	awsWebIdentityTokenFile := os.Getenv("AWS_WEB_IDENTITY_TOKEN_FILE")

	if awsWebIdentityTokenFile != "" {
		log.Println("Using assumerole with web identity")
		awsRoleArn := os.Getenv("AWS_ROLE_ARN")
		awsWebIdentityToken, err := ioutil.ReadFile(awsWebIdentityTokenFile)
		if err != nil {
			return err
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
			return err
		}

		// Creating a new session with the temporary credentials
		s.sess = session.Must(session.NewSession(&aws.Config{
			Credentials: credentials.NewStaticCredentialsFromCreds(credentials.Value{
				AccessKeyID:     *identity.Credentials.AccessKeyId,
				SecretAccessKey: *identity.Credentials.SecretAccessKey,
				SessionToken:    *identity.Credentials.SessionToken,
				ProviderName:    "AssumeRoleWithWebIdentity",
			}),
		}))
	}

	return nil

}

func (s *S3) Create(name string) error {
	// Create a new S3 client
	s3Client := s3.New(s.sess)
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
			log.Printf("%v", err)
			return err
		}
		return nil
	}
	defer obj.Body.Close()

	return nil
}

func (s *S3) Update(name string, version bool, encryption bool) error {
	// Create a new S3 client
	s3Client := s3.New(s.sess)

	// Update Bucket
	// Versioning
	/*
		if version {
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
	*/

	// Encryption
	if encryption {
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

func (s *S3) Delete(name string) error {
	s3Client := s3.New(s.sess)

	if obj, err := s3Client.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(name),
		Key:    aws.String("//DO_NOT_DELETE"),
	}); err != nil && obj.Body == nil {
		// Delete everything in the bucket
		s.deleteItems(name)

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
	}

	return nil
}

func (s *S3) deleteItems(name string) error {
	s3Client := s3.New(s.sess)

	iter := s3manager.NewDeleteListIterator(s3Client, &s3.ListObjectsInput{
		Bucket: aws.String(name),
	})

	err := s3manager.NewBatchDeleteWithClient(s3Client).Delete(aws.BackgroundContext(), iter)
	if err != nil {
		return err
	}

	return nil
}
