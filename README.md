# Knative Template Repository

## Introduction to Knative
Knative is a Kubernetes-based platform that simplifies the deployment of serverless workloads. It provides powerful building blocks such as event-driven computing and autoscaling capabilities, making it ideal for building scalable cloud-native applications.

Key features of Knative include:
- **Knative Serving**: Run serverless containers with advanced routing, autoscaling, and rollout capabilities.
- **Knative Eventing**: Build event-driven architectures using cloud events.

This repository provides a template to quickly bootstrap Knative-based services.

---

## Knative Installation Guide
### MacOS
1. Install the Knative client:
   ```bash
   brew install knative/client/kn
2. Install the Knative kn-plugins:
   ```bash
   brew tap knative-extensions/kn-plugins
3. Install the func:
   ```bash
   brew install func


### Ubuntu
1. Download the func binary for Ubuntu:
   ```bash
   wget https://github.com/knative/func/releases/download/knative-v1.16.1/func_linux_amd64
2. Make the binary executable:
   ```bash
   chmod +x func_linux_amd64
3. Move the binary to a directory in your PATH:
   ```bash
   sudo mv func_linux_amd64 /usr/local/bin/func
---

## Generate New Function
To create a new function, use the provided template:
```bash
    make gen-function
```

---

## Redis Integration
Redis can be used to manage state or as a message broker.
Redis client can be enabled with following environment variables.

```env
REDIS_HOST=''
REDIS_PORT=6379
REDIS_USER=redis_user # if exists
REDIS_PASS=redis_pass # if password is enabled
REDIS_ENABLED=true
```

---

## Logging
Logging is crucial for monitoring and debugging. To see the logs of your application go 
to the following [link](https://grafana.u-code.io/explore?schemaVersion=1&panes=%7B%222He%22:%7B%22datasource%22:%22loki%22,%22queries%22:%5B%7B%22refId%22:%22A%22,%22expr%22:%22%7Bnamespace%3D%5C%22knative-fn%5C%22%7D%20%7C%3D%20%60%60%22,%22queryType%22:%22range%22,%22datasource%22:%7B%22type%22:%22loki%22,%22uid%22:%22loki%22%7D,%22editorMode%22:%22builder%22%7D%5D,%22range%22:%7B%22from%22:%22now-1h%22,%22to%22:%22now%22%7D%7D%7D&orgId=1). And choose your function.


---

## Running on local
Make sure that `func` is installed on your laptop by running the command, `func version`.

Build image:
```bash
make build-function
```

Run container: You have to give external container port for PORT parametr
```bash
make run PORT=8888
```

Stop image: 
```bash
make stop
```

You can send request to the faas in local with the url: `http://localhost:PORT/`. 

Inside the ./faas/main.go you can build another paths too. But it is only recommended for specific cases.

---
For more details or support, feel free to raise issues in this repository.
