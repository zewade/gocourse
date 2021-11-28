#FROM golang:1.17-alpine AS build
#LABEL author="zewade" course="cloudnative"
#COPY main.go go.mod /go/src/httpserver/
#WORKDIR /go/src/httpserver/
#RUN go build -o /bin/httpserver
#FROM alpine
#COPY --from=build /bin/httpserver /bin/httpserver
#EXPOSE 8080
#ENTRYPOINT ["/bin/httpserver"]
FROM alpine
ENV VERSION=1.17.1
LABEL multi.lang="go" multi.type="webserver" other="homework"
ADD bin/amd64/httpserver /httpserver
EXPOSE 8080
ENTRYPOINT /httpserver