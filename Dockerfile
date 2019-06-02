FROM golang:1.12.5
WORKDIR /app
EXPOSE 8080
RUN go get github.com/lib/pq
RUN go get github.com/gorilla/mux
CMD ["go", "run", "server.go"]