FROM golang:1.18.1
RUN curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl" && \
    chmod +x kubectl && mv kubectl /usr/local/bin/
RUN curl -L -o kubebuilder https://github.com/kubernetes-sigs/kubebuilder/releases/download/v3.4.0/kubebuilder_linux_amd64 && \
    chmod +x kubebuilder && mv kubebuilder /usr/local/bin/
ENV GO111MODULE=on
WORKDIR /go/src/github.com/linuxshokunin/s3-controller
