# Quick start

Run the command to start the scanner service and its dependencies:
`make local`

Files can be uploaded to the fake bucket using:
`aws --endpoint-url=http://127.0.0.1:4566 s3 cp <your_file> s3://samples-scanner-bucket`

You must have awscli installed to be able to perform the following commands. Check
https://docs.aws.amazon.com/cli/latest/userguide/getting-started-install.html for more information.

# Advanced usage

If you wish to submit messages to slack during the test or send samples to VirusTotal, you must configure the
appropriate variables at localstack/docker-compose.yml, be sure to NEVER COMMIT secrets to the repo.