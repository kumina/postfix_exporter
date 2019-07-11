
build:
	docker build -t postfix_exporter .
	docker run --name postfix_exporter postfix_exporter /bin/true 2> /dev/null || echo "Its ok"
	docker cp postfix_exporter:/bin/postfix_exporter .
	docker rm postfix_exporter

clean:
	[ -f "postfix_exporter" ] && rm postfix_exporter

