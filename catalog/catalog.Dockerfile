# FBC catalog image Dockerfile
# Built with: make catalog-build
# The catalog/ directory is the FBC root; opm serves it at /configs inside the image.
FROM quay.io/operator-framework/opm:latest
ENTRYPOINT ["/bin/opm"]
CMD ["serve", "/configs", "--cache-dir=/tmp/cache"]
ADD catalog /configs
RUN ["/bin/opm", "serve", "/configs", "--cache-dir=/tmp/cache", "--terminate-after-cache"]
LABEL operators.operatorframework.io.index.configs.v1=/configs
