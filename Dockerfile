FROM alpine
RUN apk --update add ca-certificates && rm -rf /var/cache/apk/*
ADD ./build/lora-gateway-bridge /usr/local/bin/lora-gateway-bridge
RUN chmod 755 /usr/local/bin/lora-gateway-bridge
ENTRYPOINT ["/usr/local/bin/lora-gateway-bridge"]
