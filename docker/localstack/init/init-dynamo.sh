#!/bin/bash

awslocal dynamodb create-table \
    --table-name global01 \
    --key-schema AttributeName=id,KeyType=HASH \
    --attribute-definitions AttributeName=id,AttributeType=S \
    --billing-mode PAY_PER_REQUEST

awslocal dynamodb put-item \
    --table-name global01 \
    --item '{"id":{"S":"foo"}}'