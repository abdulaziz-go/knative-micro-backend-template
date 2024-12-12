CURRENT_DIR=$(shell pwd)
FUNCTION_PATH=$(shell basename ${CURRENT_DIR})

gen-function:
	func create ${FUNCTION_PATH} -l go -t function --repository https://github.com/Ucode-io/knative-template

build-function: 
	cd ${FUNCTION_PATH} && func build --registry ${FUNCTION_PATH} -v && cd ..

run:
	docker stop ${FUNCTION_PATH} || true && \
	docker rm ${FUNCTION_PATH} || true && \
	docker run -d --name ${FUNCTION_PATH} -p ${PORT}:8080 ${FUNCTION_PATH}/${FUNCTION_PATH}

stop:
	docker stop ${FUNCTION_PATH} || true && \
	docker rm ${FUNCTION_PATH} || true 