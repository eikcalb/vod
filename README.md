# VOD 
VOD is a multimedia processing service.

## How it works
Processing is abstracted to enable any arbitrary endpoint connect to the service.

## Scripts to setup application on AWS

To clear old zip, rebuild golang project and repackage the zip
```shell
rm function.zip && GOOS=linux go build main.go && zip function.zip config.json main
```

Create new function
```shell
aws lambda create-function --function-name vod-lambda-function \
--runtime go1.x --zip-file fileb://function.zip --handler main --role arn:aws:iam::468577352438:role/lambda-role \
--memory-size 1024 --timeout 840
```

Update lambda function
```shell
aws lambda update-function-code --function-name vod-lambda-function --zip-file fileb://function.zip 
```

```shell
aws lambda add-permission --function-name vod-lambda-function --principal s3.amazonaws.com \
--statement-id s3invoke --action "lambda:InvokeFunction" \
--source-arn arn:aws:s3:::vod-catalogue-storage \
--source-account 468577352438
```

```shell
aws lambda add-permission --function-name vod-lambda-function --principal s3.amazonaws.com \
--statement-id s3invoke2 --action "lambda:InvokeFunction" \
--source-arn arn:aws:s3:::vod-file-storage-test \
--source-account 468577352438
```