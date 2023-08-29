YARA_VERSION := 4.2.2
LD_LIBRARY_PATH := "/usr/local/lib"
BINARY := eagleeye

osdeps:
	apt-get update -y && apt-get install -y libssl-dev file libjansson-dev bison python3 tini
	apt-get install -y python3-setuptools \
                                    python3-dev \
                                    libc-dev \
                                    libmagic-dev \
                                    automake \
                                    autoconf \
                                    libtool \
                                    flex \
                                    git \
                                    gcc

yara:
	set -x \
      && echo "Install Yara from source..." \
      && cd /tmp/ \
      && rm -rf /tmp/yara \
      && git clone --recursive --branch v${YARA_VERSION} https://github.com/VirusTotal/yara.git \
      && cd /tmp/yara \
      && ./bootstrap.sh \
      && sync \
      && ./configure --enable-cuckoo \
                     --enable-magic \
                     --with-crypto \
      && make \
      && make install \
      && ldconfig


godeps:
	go mod download

.PHONY: deps
deps: godeps osdeps yara

test:
	LD_LIBRARY_PATH=${LD_LIBRARY_PATH} go test -count=1 -v -failfast -cover ./... -coverprofile=coverage_unit.out

.PHONY: docs
docs:
	go install github.com/swaggo/swag/cmd/swag@latest
	swag init

.PHONY: gen
gen:
	go generate ./...

build: gen docs
	go build -o ${BINARY}

test/e2e:
	LD_LIBRARY_PATH=${LD_LIBRARY_PATH} go test --tags=e2e -count=1 -v -failfast -cover ./e2e/... -coverpkg ./... -coverprofile=coverage_e2e.out

.PHONY: lint
lint:
	golangci-lint run --out-format code-climate:gl-code-quality-report.json,colored-line-number,checkstyle:golangci-report.xml --sort-results;

local:
	docker build . -f Dockerfile -t eagleeye
	docker build . -f local.Dockerfile -t eagleeye-local
	cd localstack && docker-compose up

clean:
	docker rm $$(docker ps -a -q)
	docker image rm eagleeye-local
