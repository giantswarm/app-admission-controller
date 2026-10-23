FROM gsoci.azurecr.io/giantswarm/alpine:3.24.3
WORKDIR /app
ARG TARGETARCH
COPY app-admission-controller-linux-${TARGETARCH} /app/app-admission-controller
CMD ["/app/app-admission-controller"]
