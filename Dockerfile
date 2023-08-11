FROM alpine:3.18.3
WORKDIR /app
COPY app-admission-controller /app
CMD ["/app/app-admission-controller"]
