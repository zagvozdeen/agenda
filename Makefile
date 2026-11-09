build:
	CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o bin/agenda ./cmd/main.go

deploy:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags='-s -w' -o bin/agenda ./cmd/main.go
	scp bin/agenda config.production.yaml root@185.221.214.4:/tmp/
	ssh root@185.221.214.4 'install -g www-data -m 750 /tmp/agenda /var/www/agenda.trixy.space/agenda && install -g www-data -m 640 /tmp/config.production.yaml /var/www/agenda.trixy.space/config.yaml && rm /tmp/agenda /tmp/config.production.yaml && systemctl restart agenda'
