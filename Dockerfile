FROM alpine:3.13.2
WORKDIR /app
COPY app-admission-controller /app
CMD ["/app/app-admission-controller"]
