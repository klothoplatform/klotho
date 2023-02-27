FROM golang:1.20 as builder

WORKDIR /usr/src/app
ENV GOOS=linux GOARCH=amd64 CGO_ENABLED=0
COPY go.mod ./
RUN go mod tidy && go mod download && go mod verify
COPY . .
RUN go build -o /usr/local/bin/app

FROM public.ecr.aws/lambda/provided:al2 
COPY --from=builder /usr/local/bin/app main
ENTRYPOINT ["./main"]
