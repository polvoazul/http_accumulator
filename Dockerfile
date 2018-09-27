FROM golang:alpine
COPY accumulator.go accumulator.go
RUN go build -o /accumulator accumulator.go

FROM alpine:latest
COPY --from=0 /accumulator .
CMD ["./accumulator"]
