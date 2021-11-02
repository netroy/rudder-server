#!/bin/sh
apt-get -y update
apt-get -y install build-essential unixodbc unixodbc-dev wget unzip libsasl2-modules-gssapi-mit netcat dpkg
wget https://databricks-bi-artifacts.s3.us-east-2.amazonaws.com/simbaspark-drivers/odbc/2.6.18/SimbaSparkODBC-2.6.18.1030-Debian-64bit.zip
unzip SimbaSparkODBC-2.6.18.1030-Debian-64bit.zip
dpkg -i simbaspark_2.6.18.1030-2_amd64.deb
rm -rf SimbaSparkODBC-2.6.18.1030-Debian-64bit.zip simbaspark_2.6.18.1030-2_amd64.deb docs/