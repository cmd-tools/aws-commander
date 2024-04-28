#!/bin/bash

function add_complex_items_in_table() {
  table_name="$1"
  maxItems="$2"
  for i in $(seq 1 $maxItems);
  do
    data=$(echo '{"id":{"S":"foo__ID__"},"data":{"S":"data__ID__"},"attribute1":{"S":"value__ID__"},"attribute2":{"N":"__ID__"},"large_attribute":{"S":"Lorem ipsum dolor sit amet, consectetur adipiscing elit. Integer nec odio. Praesent libero. Sed cursus ante dapibus diam. Sed nisi. Nulla quis sem at nibh elementum imperdiet. Duis sagittis ipsum. Praesent mauris. Fusce nec tellus sed augue semper porta. Mauris massa. Vestibulum lacinia arcu eget nulla. Class aptent taciti sociosqu ad litora torquent per conubia nostra, per inceptos himenaeos. Curabitur sodales ligula in libero. Sed dignissim lacinia nunc. Curabitur tortor. Pellentesque nibh. Aenean quam. In scelerisque sem at dolor. Maecenas mattis. Sed convallis tristique sem. Proin ut ligula vel nunc egestas porttitor. Morbi lectus risus, iaculis vel, suscipit quis, luctus non, massa. Fusce ac turpis quis ligula lacinia aliquet. Mauris ipsum. Nulla metus metus, ullamcorper vel, tincidunt sed, euismod in, nibh."}}' | sed "s/__ID__/$i/g")
    awslocal dynamodb put-item \
        --table-name "$table_name" \
        --item "${data}"
  done
}

function add_simple_items_in_table() {
    table_name="$1"
    maxItems="$2"
    for i in $(seq 1 $maxItems);
    do
      data=$(echo '{"id":{"S":"foo__ID__"},"data":{"S":"data__ID__"}}' | sed "s/__ID__/$i/g")
      awslocal dynamodb put-item \
          --table-name "$table_name" \
          --item ${data}
    done
}

awslocal dynamodb create-table \
    --table-name table_for_pagination \
    --key-schema AttributeName=id,KeyType=HASH \
    --attribute-definitions AttributeName=id,AttributeType=S \
    --billing-mode PAY_PER_REQUEST

add_simple_items_in_table table_for_pagination 98

awslocal dynamodb create-table \
    --table-name simple_table \
    --key-schema AttributeName=id,KeyType=HASH \
    --attribute-definitions AttributeName=id,AttributeType=S \
    --billing-mode PAY_PER_REQUEST

add_simple_items_in_table simple_table 7

awslocal dynamodb create-table \
    --table-name simple_table_with_complex_items \
    --key-schema AttributeName=id,KeyType=HASH \
    --attribute-definitions AttributeName=id,AttributeType=S \
    --billing-mode PAY_PER_REQUEST

add_complex_items_in_table simple_table_with_complex_items 5

awslocal dynamodb create-table \
    --table-name table_for_pagination_with_complex_items \
    --key-schema AttributeName=id,KeyType=HASH \
    --attribute-definitions AttributeName=id,AttributeType=S \
    --billing-mode PAY_PER_REQUEST

add_complex_items_in_table table_for_pagination_with_complex_items 150
