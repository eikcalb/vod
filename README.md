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
aws lambda create-function --function-name vod-video-function \
--runtime go1.x --zip-file fileb://function.zip --handler main --role arn:aws:iam::406156568264:role/lambda-s3-role \
--memory-size 2048 --timeout 840
```

Update lambda function
```shell
aws lambda update-function-code --function-name vod-video-function --zip-file fileb://function.zip 
```

```shell
aws lambda add-permission --function-name vod-video-function --principal s3.amazonaws.com \
--statement-id s3invoke --action "lambda:InvokeFunction" \
--source-arn arn:aws:s3:::findapp-media-vod-input \
--source-account 406156568264
```



ffmpeg -i ~/downloads/test_video_upload.mp4 -movflags frag_keyframe+empty_moov -vf scale=1280:720:force_original_aspect_ratio=decrease,pad=1280:720:(ow-iw)/2:(oh-ih)/2 output.mp4