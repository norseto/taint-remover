# Taint Remover
Remove Spot Instance taints set by the cloud provider.

# How to deploy to a cluster
## Deploy CRD, RBAC, Controller
```
kubectl apply -k github.com/norseto/taint-remover/config/default?ref=release-0.4
```
## Deploy CR for OCI(Oracle Cloud)
A sample is made for OCI.  
```
kubectl apply -k github.com/norseto/taint-remover/config/samples?ref=release-0.4
```

You can examine the sample by `kubectl kustomize`
```YAML
apiVersion: nodes.peppy-ratio.dev/v1alpha1
kind: TaintRemover
metadata:
  labels:
    app.kubernetes.io/created-by: taint-remover
    app.kubernetes.io/instance: taintremover-sample
    app.kubernetes.io/managed-by: kustomize
    app.kubernetes.io/name: taintremover
    app.kubernetes.io/part-of: taint-remover
  name: taintremover-sample
  namespace: taint-remover-system
spec:
  taints:
  - effect: NoSchedule
    key: oci.oraclecloud.com/oke-is-preemptible

## License

This project is primarily licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

Additionally, certain components within the `internal/taints` directory are derived from Kubernetes (`k8s.io/kubernetes/pkg/util/taints`) and are licensed under the Apache License 2.0. The original copyright notices are preserved within the respective source files. A copy of the Apache License 2.0 can be found in the [LICENSES/Apache-2.0.txt](LICENSES/Apache-2.0.txt) file.
