FROM alpine:3.13.1
WORKDIR /app
COPY app-admission-controller /app
CMD ["/app/app-admission-controller"]
