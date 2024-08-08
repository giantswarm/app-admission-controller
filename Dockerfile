FROM gsoci.azurecr.io/giantswarm/alpine:3.20.2
WORKDIR /app
COPY app-admission-controller /app
CMD ["/app/app-admission-controller"]
