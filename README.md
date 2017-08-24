## What is TGen?

TGen is a tiny utils written by Golang and suted for semi-automatic generating 
tests for Google gRPC protocol. It works with Protobuf 3 files and let you generate 
and execute tests for gRPC server methods which listed in .proto file.
you also can save test Go code and use / modify it.

### Flags

* --address  |    server address
* --file     |    name of proto file
* --method   |    method for test
* --save     |    optional flag if you want to save temp dir with Go test

Example:

```
./tgen --address 127.0.0.1:12345 --file server-grpc.proto --method DoSomeWork --save true
```

