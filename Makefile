build:
	@echo "Building for prod"
	docker build -t donnieashok/pickflick:prod .

up:
	@echo "Running for Prod"
	docker run -dit --rm -p 1339:8080 --name pickflick donnieashok/pickflick:prod

deploy: build
	docker push donnieashok/pickflick:prod
	@echo "Deployed!"

live:
	ssh root@vultr docker pull donnieashok/pickflick:prod
	- ssh root@vultr docker stop pickflick
	scp -r ./.env root@vultr:/root/
	ssh root@vultr docker run -d -v /home/pickflick/:/db/ --rm --env-file /root/.env -p 1339:8080 --name pickflick donnieashok/pickflick:prod
	ssh root@vultr rm /root/.env
	@echo "Is live"

publish: deploy live

clean:
	docker stop pickflick
	@echo "all clear"
