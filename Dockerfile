FROM gsoci.azurecr.io/giantswarm/alpine:3.21.0
WORKDIR /app
COPY app-admission-controller /app
CMD ["/app/app-admission-controller"]
