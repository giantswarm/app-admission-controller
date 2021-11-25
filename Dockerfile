FROM alpine:3.15.0
WORKDIR /app
COPY app-admission-controller /app
CMD ["/app/app-admission-controller"]
