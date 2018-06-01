docker:
	sudo docker build -t hub_backend .

run:
	@echo starting container on port 8000
	sudo docker run -p 8000:8000 hub_backend
