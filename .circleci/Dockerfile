FROM circleci/golang:1.10

# A Dockerfile to use for building goiardi in circleci.

RUN sudo apt-get update && sudo apt-get install rpm python-sphinx ruby rubygems ruby-dev -y && \
	sudo gem install fpm && \
	sudo go get github.com/ctdk/gox
