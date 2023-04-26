FROM public.ecr.aws/bitnami/golang:1.12
ADD go.mod go.sum ./
RUN go mod download
ADD . .
RUN GOOS=linux go build -o /main main.go
FROM public.ecr.aws/lambda/go:1
ARG FUNCTION_DIR="/var/task"
COPY --from=build /main ${FUNCTION_DIR}/main
CMD [ "main" ]