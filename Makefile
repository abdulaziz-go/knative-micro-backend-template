CURRENT_DIR=$(shell pwd)
FUNCTION_PATH=$(shell basename ${CURRENT_DIR})

gen-function:
	func create ${FUNCTION_PATH} -l go -t function --repository https://github.com/Ucode-io/knative-template