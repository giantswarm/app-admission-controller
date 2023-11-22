FROM gsocr.azurecr.io/giantswarm/alpine:3.18.4
WORKDIR /app
COPY app-admission-controller /app
CMD ["/app/app-admission-controller"]
