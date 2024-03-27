#!/bin/bash

awslocal s3api create-bucket --bucket sample-bucket --create-bucket-configuration LocationConstraint=$AWS_DEFAULT_REGION
awslocal s3api create-bucket --bucket sample-bucket1 --create-bucket-configuration LocationConstraint=$AWS_DEFAULT_REGION
awslocal s3api create-bucket --bucket sample-bucket2 --create-bucket-configuration LocationConstraint=$AWS_DEFAULT_REGION
awslocal s3api create-bucket --bucket sample-bucket3 --create-bucket-configuration LocationConstraint=$AWS_DEFAULT_REGION

awslocal s3api put-object \
    --bucket sample-bucket \
    --key index.html \
    --body /etc/localstack/init/ready.d/data/s3/index.html