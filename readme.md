# Burgate

A distributed system that mimics a fast-food ordering, cooking and pick-up services. 

Use gRPC and RabbitMQ for inter-service communications. The project aims to be practical, but the main goal is for me to try and integrate different microservices.

#### Main Features:
1. Gateway functions as an entry point for customers, provides ordering api and menu altering api.
2. Scalable ordering microservice, use Nginx to load balance, similar to real-life ordering machines.
3. Payment system
4. Centralised Kitchen
5. Everything docker containerised
6. MongoDB
