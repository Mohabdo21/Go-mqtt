
# Colors
GREEN=\033[0;32m
RESET=\033[0m

.PHONY: run stop help

default: help

run:
	@echo "${GREEN}Starting RabbitMQ...${RESET}"
	@docker run -d --rm --name rabbitmq -p 5672:5672 -p 15672:15672 rabbitmq:4-management


stop:
	@echo "${GREEN}Stopping RabbitMQ...${RESET}"
	@docker stop rabbitmq


help:
	@echo "${GREEN}Available commands:${RESET}"
	@echo "  make run    - Start RabbitMQ"
	@echo "  make stop   - Stop RabbitMQ"
	@echo "  make help   - Show this help message"
