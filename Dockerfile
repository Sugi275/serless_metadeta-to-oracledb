FROM ubuntu:18.04 
MAINTAINER sugimount <https://twitter.com/sugimount>

# set working directory
WORKDIR /opt/oracle

# package install
RUN apt-get update && \
    apt-get install -y wget unzip libaio1 && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

# instant client
RUN wget https://download.oracle.com/otn_software/linux/instantclient/193000/instantclient-basic-linux.x64-19.3.0.0.0dbru.zip && \
    unzip instantclient-basic-linux.x64-19.3.0.0.0dbru.zip && \
    rm /opt/oracle/instantclient-basic-linux.x64-19.3.0.0.0dbru.zip

# runtime linkpath
COPY oracle-instantclient.conf /etc/ld.so.conf.d/oracle-instantclient.conf
RUN ldconfig

# user add
RUN adduser --uid 1000 go-oracledb && \
    chown go-oracledb: /opt/oracle/instantclient_19_3/network/admin

# go binary
COPY ./bin/serless_metadeta-to-oracledb /home/go-oracledb/serless_metadeta-to-oracledb

# default user
USER 1000:1000

ENTRYPOINT ["/home/go-oracledb/serless_metadeta-to-oracledb"]