FROM alpine:3.15
ENV VERSION=1.17.1
LABEL multi.lang="go" multi.type="webserver" other="homework"
ADD bin/amd64/httpserver /httpserver
EXPOSE 8080
#ENTRYPOINT /httpserver
CMD ["/httpserver", "-v=4","-logtostderr"]