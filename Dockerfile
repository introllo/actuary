FROM golang:1.8.3

ENV VERSION 17.06.0
ENV GOBIN=$GOPATH/bin
ENV HTTP_PROXY  http://proxy-chain.intel.com:911
ENV HTTPS_PROXY http://proxy-chain.intel.com:912

# Configure APT
RUN touch /etc/apt/apt.conf
RUN echo 'Acquire::http::proxy "http://proxy-chain.intel.com:911/";' >> /etc/apt/apt.conf
RUN echo 'Acquire::https::proxy "http://proxy-chain.intel.com:912/";' >> /etc/apt/apt.conf


RUN apt-get  update && apt-get install -y git auditd

COPY . $GOPATH/src/github.com/introllo/actuary
WORKDIR $GOPATH/src/github.com/introllo/actuary
RUN go get github.com/introllo/actuary/actuary
RUN go get github.com/diogomonica/actuary/actuary
RUN go install github.com/introllo/actuary/cmd/actuary

ENTRYPOINT ["/go/bin/actuary"]
