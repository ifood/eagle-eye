package awsutils

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sns"
)

func SendSMS(session *session.Session, config *aws.Config, phone, message string) error {
	svc := sns.New(session, config)

	params := sns.PublishInput{
		Message:     aws.String(message),
		PhoneNumber: aws.String(phone),
	}

	_, err := svc.Publish(&params)
	return err
}
