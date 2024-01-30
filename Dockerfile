FROM golang:1.21.0 as builder


WORKDIR /app

COPY main.go .
COPY otel.go .
COPY rolldice.go .
COPY go.mod .
COPY go.sum .
RUN go mod tidy
RUN ls -l /app
RUN go build -o myapp .
# RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o myapp .
RUN ls -l /app

FROM alpine:3.14 as prod
# This line fixed alpine problem of finding folders 
RUN apk add libc6-compat
WORKDIR /app
COPY --from=builder /app/myapp .
RUN chmod +x myapp
RUN ls -l /app
EXPOSE 8080

# Command to run the executable
CMD ["./myapp"]
# RUN ./myapp%  