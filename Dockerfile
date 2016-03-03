FROM phusion/baseimage:0.9.18

# Use baseimage-docker's init system.
CMD ["/sbin/my_init"]

ENV DEBIAN_FRONTEND=noninteractive
RUN apt-get update

# Update Ubuntu itself.
RUN apt-get upgrade -y -o Dpkg::Options::="--force-confold"

# Install Git.
RUN apt-get install -y git wget unzip

# Apt cleanup.
RUN apt-get clean && rm -rf /var/lib/apt/lists/*

# Create dedicated resizer user.
RUN adduser --system --no-create-home --shell /bin/false resizer

# Install Golang.
WORKDIR /usr/local
RUN curl -O https://storage.googleapis.com/golang/go1.5.3.linux-amd64.tar.gz && \
    tar -xvf go1.5.3.linux-amd64.tar.gz

# Add Golang bin folder to path.
ENV PATH=$PATH:/usr/local/go/bin

# Set $GOPATH
RUN mkdir /var/gopath
ENV GOPATH=/var/gopath

# Install godep for Go dependency management.
RUN go get github.com/tools/godep

# Get the resizer git repo.
RUN mkdir -p /var/gopath/src/github.com/hellofresh/resizer

# Define DST_DIR var for further actions.
ENV DST_DIR=/var/gopath/src/github.com/hellofresh/resizer

# Copy supervise run scripts into place.
COPY docker-resources/resizer_service.sh /etc/service/resizer/run

# See config.json for port definition.
EXPOSE 8080

# Define directory where the cache is located.
RUN mkdir -p /var/resizer_cache/storage && \
    chown -R resizer:nogroup /var/resizer_cache

# Copy the relevant source files plus configuration into place.
COPY config.json ${DST_DIR}/
COPY Godeps/ ${DST_DIR}/Godeps/
COPY cache/ ${DST_DIR}/cache/
COPY *.go ${DST_DIR}/

# Here the work gets done.
WORKDIR /var/gopath/src/github.com/hellofresh/resizer

# Restore Go build dependencies and build resizer binary.
RUN  /var/gopath/bin/godep restore && \
     go build

