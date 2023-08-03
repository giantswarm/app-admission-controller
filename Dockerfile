FROM alpine:3.18.2
WORKDIR /app
COPY app-admission-controller /app
CMD ["/app/app-admission-controller"]
