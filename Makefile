
# Colors
GREEN=\033[0;32m
RESET=\033[0m

.PHONY: run_rabbitmq stop_rabbitmq run_timescaledb stop_timescaledb help

default: help

run_rabbitmq:
	@echo "${GREEN}Starting RabbitMQ...${RESET}"
	@docker run -d --rm --name rabbitmq -p 5672:5672 -p 15672:15672 rabbitmq:4-management


stop_rabbitmq:
	@echo "${GREEN}Stopping RabbitMQ...${RESET}"
	@docker stop rabbitmq


run_timescaledb:
	@echo "${GREEN}Starting TimescaleDB...${RESET}"
	@docker-compose up -d 

stop_timescaledb:
	@echo "${GREEN}Stopping TimescaleDB...${RESET}"
	@docker-compose down


help:
	@echo "${GREEN}Available commands:${RESET}"
	@echo "  make run_rabbitmq    		- Start RabbitMQ"
	@echo "  make stop_rabbitmq   		- Stop RabbitMQ"
	@echo "  make run_timescaledb 		- Start TimescaleDB"
	@echo "  make stop_timescaledb 	- Stop TimescaleDB"
	@echo "  make help   			- Show this help message"
