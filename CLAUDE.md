# gRPC Examples

This project is intended to exercise the capabilities of gRPC and Golang.
It is concerned with comprehensive coverage of the tools and demostration of ability to work with them.

As gRPC uses Protobufs by default, we want to have a collection of data structures that exercise all the various ways that data can be modeled into protobufs.
And as many services are oriented towards REST-like API services, this project leverages this project for core tooling: https://github.com/grpc-ecosystem/grpc-gateway

We want to create an example system that has a service that hosts the various interface gRPC patterns (e.g., Unary, Server Streaming, Client Streaming, and Bidirectional Streaming, as well as the usage of Intercepters to modify the behavior of those patterns.

Another goal is to use as much "modern" tooling as possible, such as Justfiles rather than Makefiles, Buf (https://github.com/bufbuild/buf) for generating bindings from proto files.

The desired outcome is a server that demonstrates comprehensive gRPC utility with a server and a client that interacts with the server. While a real server would likely have some datastore for data persistence, for the first pass an in-memory k/v store will do; that should be abstracted so that a proper database backend can easily be added in.

Note that this project was started some time ago and needs to be updated, there is no need to worry about migrating anything as it's not otherwise in use. If you see code or data patterns that don't seem correct please point them out and offer suggested improvements.
