FROM golang:1.23.1
COPY . /root/
WORKDIR /root/cmd/
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v -o ReaperC2
ENV DEPLOY_ENV="ONPREM"
EXPOSE 8080
ENTRYPOINT [ "/root/cmd/ReaperC2" ]
