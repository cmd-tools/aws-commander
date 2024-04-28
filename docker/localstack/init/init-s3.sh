#!/bin/bash

awslocal s3api create-bucket --bucket sample-bucket --create-bucket-configuration LocationConstraint=$AWS_DEFAULT_REGION
awslocal s3api create-bucket --bucket empty-bucket --create-bucket-configuration LocationConstraint=$AWS_DEFAULT_REGION
awslocal s3api create-bucket --bucket bucket-with-subfolders --create-bucket-configuration LocationConstraint=$AWS_DEFAULT_REGION

awslocal s3api put-object \
    --bucket sample-bucket \
    --key index.html \
    --body /etc/localstack/init/ready.d/data/s3/index.html

awslocal s3 cp /etc/localstack/init/ready.d/data/s3/testFolder1 s3://bucket-with-subfolders/testFolder1/ --recursive
