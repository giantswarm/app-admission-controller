FROM alpine:3.15.3
WORKDIR /app
COPY app-admission-controller /app
CMD ["/app/app-admission-controller"]
