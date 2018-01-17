all:docker-build

docker-build:
	docker build -t tpr-webhook:live .

docker-run-hot:
	docker run -d -p 9000:9000 --name=tpr-webhook tpr-webhook:live -verbose -hooks=/etc/webhook/hooks.json -hotreload

docker-run-name:
	docker run -p 9000:9000 --name=tpr-webhook tpr-webhook:live

docker-run:
	docker run -p 9000:9000 tpr-webhook:live

now:
	now -f
	now alias