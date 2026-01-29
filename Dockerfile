FROM gsoci.azurecr.io/giantswarm/alpine:3.23.3
WORKDIR /app
COPY app-admission-controller /app
CMD ["/app/app-admission-controller"]
