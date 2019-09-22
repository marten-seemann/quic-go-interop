FROM martenseemann/quic-network-simulator-endpoint:latest

RUN apt-get update && apt-get install -y wget tar git vim python

RUN wget https://dl.google.com/go/go1.13.linux-amd64.tar.gz && \
  tar xfz go1.13.linux-amd64.tar.gz && \
  rm go1.13.linux-amd64.tar.gz

ENV PATH="/go/bin:${PATH}"

WORKDIR /quic-go-interop
ADD interop/go.* /quic-go-interop/
ADD interop/http09 /quic-go-interop/http09
ADD interop/client /quic-go-interop/client
ADD interop/server /quic-go-interop/server

# download dependencies
RUN cd client && go build && cd ../server && go build

COPY run_endpoint.sh .
RUN chmod +x run_endpoint.sh

ENTRYPOINT [ "./run_endpoint.sh" ]
