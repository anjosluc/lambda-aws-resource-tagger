# Business Tagger Lambda

To build it for zip package:

```GOOS=linux go build -o main main.go```

For Docker use the Dockerfile for Lambda Image:

```docker build -t business-tagger:latest .```

It assumes that the originating event comes from SQS and it's treated one resource per invocation.

Take note about patterns that come from AWS Config if you're modifying JSON parsing on the function.