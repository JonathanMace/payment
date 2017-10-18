FROM golang:1.7

RUN mkdir /app
COPY . /go/src/github.com/JonathanMace/payment/

RUN go get github.com/JonathanMace/tracing-framework-go/cmd/modify-runtime
RUN go run /go/src/github.com/JonathanMace/tracing-framework-go/cmd/modify-runtime/modify.go
RUN go install -a std

RUN go get -u github.com/FiloSottile/gvt
RUN cd /go/src/github.com/JonathanMace/payment/ && gvt restore

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /app/main github.com/JonathanMace/payment/cmd/paymentsvc

CMD ["/app/main", "-port=80"]

#EXPOSE 80
