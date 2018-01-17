GO ?= CGO_ENABLED=0 GOOS=linux go
GOPATH := $(CURDIR)/_vendor:$(GOPATH)

all:linux settings

linux:
	CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o docker/build/linux/tpr-journalist .
	CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o docker/build/linux/tpr-postman .

mac:
	go build -o docker/build/mac/journalist .

settings:
	cp .env docker/build/linux/.env
	cp docker/run-journalist.sh docker/build/linux/run-journalist.sh
	cp docker/run-postman.sh docker/build/linux/run-postman.sh

settings-mac:
	cp .env docker/build/mac/.env
	cp -R ./tpr_email_templates ./docker/build/linux

basic:
	go run main.go collect --all --log --save