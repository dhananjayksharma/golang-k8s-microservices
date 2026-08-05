#!/bin/bash

export MYSQL_DSN='root:root@tcp(localhost:3306)/appdb?parseTime=true'
export ORDER_PORT=8081
export PURCHASE_PORT=8085
export MESSAGE_PORT=8083
export QUEUE_PORT=8084
export INVENTORY_PORT=8914
export INVOICE_PORT=8086
export DATABASE_URL='postgres://order_user:order_password@localhost:5432/order_db?sslmode=disable'
export REDIS_ADDR='localhost:6379'
export RABBITMQ_URL='amqp://admin:admin@localhost:5672/'
 
echo "Environment variables exported:"
echo "MYSQL_DSN=$MYSQL_DSN"
echo "ORDER_PORT=$ORDER_PORT"
echo "PURCHASE_PORT=$PURCHASE_PORT"
echo "MESSAGE_PORT=$MESSAGE_PORT"
echo "QUEUE_PORT=$QUEUE_PORT"
echo "INVENTORY_PORT=$INVENTORY_PORT"
echo "INVOICE_PORT=$INVOICE_PORT"
echo "DATABASE_URL=$DATABASE_URL" 
echo "REDIS_ADDR=$REDIS_ADDR"
echo "RABBITMQ_URL=$RABBITMQ_URL"

echo "All environment variables set."
