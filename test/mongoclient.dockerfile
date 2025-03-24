FROM ubuntu:jammy
RUN apt update -y
RUN apt install wget gnupg -y
RUN wget -qO- https://www.mongodb.org/static/pgp/server-8.0.asc | tee /etc/apt/trusted.gpg.d/server-8.0.asc
RUN echo "deb [ arch=amd64,arm64 ] https://repo.mongodb.org/apt/ubuntu jammy/mongodb-org/8.0 multiverse" | tee /etc/apt/sources.list.d/mongodb-org-8.0.list
RUN apt update -y
RUN apt-get install -y mongodb-mongosh mongodb-org-tools
RUN mongosh --version
COPY setup_mongo.sh /root/
COPY data.json /root/
WORKDIR "/root/"
CMD [ "./setup_mongo.sh" ]
