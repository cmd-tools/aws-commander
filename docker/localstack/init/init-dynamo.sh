#!/bin/bash

awslocal dynamodb create-table \
    --table-name global01 \
    --key-schema AttributeName=id,KeyType=HASH \
    --attribute-definitions AttributeName=id,AttributeType=S \
    --billing-mode PAY_PER_REQUEST

for i in {1..200};
do
  data=$(echo '{"id":{"S":"foo__ID__"},"data":{"S":"data__ID__"}}' | sed "s/__ID__/$i/g")
  awslocal dynamodb put-item \
      --table-name table_for_pagination \
      --item ${data}
done
