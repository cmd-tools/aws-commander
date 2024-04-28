up: init docker-compose-up
down: docker-compose-down

init:
	@aws configure set aws_access_key_id test --profile localstack && aws configure set aws_secret_access_key test --profile localstack && aws configure set region eu-west-1 --profile localstack && aws configure set output json --profile localstack && aws configure set endpoint_url http://localhost:4566 --profile localstack

docker-compose-up:
	@docker-compose up -d
	@echo "#########################################################################"
	@echo "Wait couple of seconds to let "init-*.sh" files to be done with the seed."
	@echo "#########################################################################"

docker-compose-down:
	@docker-compose down