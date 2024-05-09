# Go gRPC Image Processing Service Demo

I just wanted to play around and learn more about concurrent development in Go, as well as, gRPC in some ways. The following project is a simple service-chained "microservice" application that has two (as of now) worker nodes that run independently of each other, and a client node. The client is responsible for streaming an image file over gRPC to the first worker node. The worker has an utility function like resizing or color inverting. The processed image is streamed onwards while being concurrently processed with the help of Goroutines and Channels. The code is pretty rudimentary as I'm only just beginning to grasp the concept of concurrency in Golang. I thought it would be a fun side project.

![image](https://github.com/kerosiinikone/go-image-processor/assets/100020686/2e5d245e-0458-40be-9ac8-1387877c5781)

### How to run locally:
```
make run-worker-invert
make run-worker-resize
make run-client
```

Keep the order of commands as the connections between nodes are dependent on the gRPC server coming alive first and they are established immediately after execution.

## TODOs

- Tests
- Common function signatures and refactoring with interfaces
- Dynamic resizing
- Refactoring logic
