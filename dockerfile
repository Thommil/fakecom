FROM golang:1.16.0-alpine3.13

RUN mkdir /app

WORKDIR /app

ADD . .
RUN go build -o /app/serve ./server/main.go

ENV QUEUE_THRESHOLD=1000
ENV QUEUE_SIZE=1000

EXPOSE 9080
#docker run -p 80:9080 -e QUEUE_THRESHOLD=100 -e QUEUE_SIZE=100  fakecom
CMD ["/app/serve"]