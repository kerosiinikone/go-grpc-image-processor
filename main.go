package main

// Image Processing Service

// service.proto defines protobuf message formats for
// image transfering between client and the services

// The service itself houses 2 - 3 separate microservices
// that communicate using gRPC

// The image is sent from client to server (1) in chunks and
// subsequently streamed to the first processing service (2)
// and onwards

// Key Points: Service Chaining, gRPC, Microservices in a Nutshell

func main() {}