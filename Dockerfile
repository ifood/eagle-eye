FROM golang:1.20 AS builder

ENV PKG_CONFIG_PATH="/usr/local/lib/pkgconfig/"

WORKDIR /eagleeye
ADD . .
RUN make deps
RUN make build
RUN mkdir -p /app/deps && mkdir -p /app/app
RUN cp /eagleeye/eagleeye /app/app/eagleeye

FROM golang:1.20 AS runner

ENV PKG_CONFIG_PATH="/usr/local/lib/pkgconfig/"

COPY Makefile .
RUN make osdeps yara

COPY --from=builder /app /app
ADD ./config.yaml /app/data/config.yaml
COPY resources/rules/ransomware/ /app/data/rules/
ENV GOGC=25

EXPOSE 3000
ENTRYPOINT ["/app/app/eagleeye"]
