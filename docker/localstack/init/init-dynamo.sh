#!/bin/bash

awslocal dynamodb create-table \
    --table-name global01 \
    --key-schema AttributeName=id,KeyType=HASH \
    --attribute-definitions AttributeName=id,AttributeType=S \
    --billing-mode PAY_PER_REQUEST

awslocal dynamodb put-item \
    --table-name global01 \
    --item '{"id":{"S":"foo"}, "data": {"S":"data"}}'

awslocal dynamodb put-item \
    --table-name global01 \
    --item '{"id":{"S":"foo1"}, "data": {"S":"data1"}}'

awslocal dynamodb put-item \
    --table-name global01 \
    --item '{"id":{"S":"foo2"}, "data": {"S":"data2"}}'

awslocal dynamodb put-item \
    --table-name global01 \
    --item '{"id":{"S":"foo3"}, "data": {"S":"data3"}}'

awslocal dynamodb put-item \
    --table-name global01 \
    --item '{"id":{"S":"foo4"}, "data": {"S":"data4"}}'

awslocal dynamodb put-item \
    --table-name global01 \
    --item '{"id":{"S":"foo5"}, "data": {"S":"data5"}}'

awslocal dynamodb put-item \
    --table-name global01 \
    --item '{"id":{"S":"foo6"}, "data": {"S":"data6"}}'

awslocal dynamodb put-item \
    --table-name global01 \
    --item '{"id":{"S":"foo7"}, "data": {"S":"data7"}}'

awslocal dynamodb put-item \
    --table-name global01 \
    --item '{"id":{"S":"foo8"}, "data": {"S":"data8"}}'

awslocal dynamodb put-item \
    --table-name global01 \
    --item '{"id":{"S":"foo9"}, "data": {"S":"data9"}}'

awslocal dynamodb put-item \
    --table-name global01 \
    --item '{"id":{"S":"foo10"}, "data": {"S":"data10"}}'
