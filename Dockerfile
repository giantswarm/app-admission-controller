FROM gsoci.azurecr.io/giantswarm/alpine:3.20.1
WORKDIR /app
COPY app-admission-controller /app
CMD ["/app/app-admission-controller"]
