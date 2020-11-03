FROM alpine:3.12
WORKDIR /app
COPY app-admission-controller /app
CMD ["/app/app-admission-controller"]
