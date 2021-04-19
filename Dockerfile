FROM alpine:3.13.5
WORKDIR /app
COPY app-admission-controller /app
CMD ["/app/app-admission-controller"]
