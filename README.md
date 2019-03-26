# Infinimesh Kubernetes Operator

[![Docker Repository on Quay](https://quay.io/repository/infinimesh/operator/status "Docker Repository on Quay")](https://quay.io/repository/infinimesh/operator)

Repository for infinimesh's operator until kubebuilder supports go modules. Then this is going to be merged into the monorepo i/i.

## Installation
```
kubectl apply -f manifests
```

or
```
kubectl apply -f https://raw.githubusercontent.com/infinimesh/operator/master/manifests/crd.yaml
kubectl apply -f https://raw.githubusercontent.com/infinimesh/operator/master/manifests/operator.yaml
```
