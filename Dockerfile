FROM alpine:3.14.3
WORKDIR /app
COPY app-admission-controller /app
CMD ["/app/app-admission-controller"]
