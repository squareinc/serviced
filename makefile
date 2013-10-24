################################################################################
#
# Copyright (C) Zenoss, Inc. 2013, all rights reserved.
#
# This content is made available according to terms specified in
# License.zenoss under the directory where your Zenoss product is installed.
#
################################################################################

default: build

build: elastigo
	go get github.com/coopernurse/gorp
	go get github.com/ziutek/mymysql/godrv
	go get github.com/zenoss/glog
	go get github.com/samuel/go-zookeeper/zk
	go get github.com/araddon/gou
	go build
	cd client && make
	cd svc && make 
	cd agent && make
	cd web && make
	cd proxy && make
	cd dao && make
	cd serviced && make

dockerbuild:
	docker build -t zenoss/serviced-build .
	docker run -v `pwd`:/go/src/github.com/zenoss/serviced -t zenoss/serviced-build make

test: build docker_ok
	go test
	cd client && make test
	cd svc && make test
	cd agent && make test
	cd web && make test
	cd proxy && make test
	cd dao && make test
	cd serviced && make test

elastigo:../../mattbaird/elastigo

../../mattbaird/elastigo:
	mkdir ../../mattbaird -p && \
	cd ../../mattbaird && \
	git clone https://github.com/zenoss/elastigo.git 

docker_ok:
	if docker ps >/dev/null; then \
		echo "docker OK"; \
	else \
		echo "Check 'docker ps' command"; \
		exit 1;\
	fi

clean:
	go clean
	cd client && make clean
	cd serviced && make clean
	cd agent && make clean
	cd svc && make clean
	cd proxy && make clean
	cd dao && make clean
