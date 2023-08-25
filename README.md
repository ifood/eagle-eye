![eagleeye](./logo.png)

# EagleEye Scanner

Scanner service checks files from cloud storage services like S3 for the presence of malware or encrypted artifacts.  
It uses Yara rules, VirusTotal integration, and file entropy calculation. Scan results can be submitted to Slack channels
and to mobile phone from the responsible team members.

# Getting started
Run the tool locally with all related infrastructure (redis, mocked s3 bucket, sns/sqs) using `make local`.

After the startup completes, a scan can be triggered by:
- Uploading a file to an integrated bucket. In the local environment, you can use the awscli:  
`aws --endpoint-url=http://127.0.0.1:4566 s3 cp <your_file> s3://samples-scanner-bucket`

- Submitting a file to the service endpoint:  
Locally, swagger is enabled by default and can be used to test the scanner. Available [here](http://localhost:3000/swagger/index.html).  
Be sure to add the default API key for testing `Bearer f00533f634f1047cfd1b1f31abf73b77934c6e8bf25640ecdcc8d1969cc4644b`.  
There are endpoints to submit local files and scanning files already in a bucket somewhere else.

- Check how to configure your own instance in AWS by looking at the [config.yaml](localstack/config.yaml) and 
[docker-compose](localstack/docker-compose.yml) files.

# Contributing  
**TODO** Is there an iFood template?

# Examples
## Approach 1 - Running inside a k8s cluster with KIAM
**Configure an appropriate service role** It must contain the following permissions. 
Be sure to be as restrictive as possible in the allowed resources the service can interact with.

```
# Read object backups
s3:GetObjectVersionTagging
s3:GetObjectTagging
s3:GetObject
s3:ListBucket

# Process messages
sqs:ReceiveMessage
sqs:DeleteMessage
sns:Publish
```

**Update the configuration file** 
Update the configuration file for your needs and make it available at the path /app/data/config.yaml. An example of configuration file can
be seen at localstack/config.yaml. Configuration file is used to set non secret values.

**Update the relevant environment variables**
Environment variables are used to pass secrets to the runtime, be sure to use hashicorp vault or another equivalent solution to make these
secrets available to the pods.
The relevant secrets can be consulted at the localstack/docker-compose.yaml. They start with `NOTIFICATION_`, `SCANNER_` and `REDIS_`.





