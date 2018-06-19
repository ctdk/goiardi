FROM circleci/golang:1.10

# A Dockerfile to use for building goiardi in circleci.

RUN sudo apt-get update && sudo apt-get install rpm python-sphinx ruby rubygems ruby-dev -y && \
	sudo apt-get clean -y && \
	sudo gem install fpm && \
	sudo gem install package_cloud -v "0.2.43" && \
	go get github.com/ctdk/gox
