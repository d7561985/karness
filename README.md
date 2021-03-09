# karness

k8s harness tests framework

Golang code-generators used to implement [Kubernetes-style API types](https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md).

# code generation

It makes use of the generators in [k8s.io/code-generator](https://github.com/kubernetes/code-generator)
to generate a typed client, informers, listers and deep-copy functions. You can
do this yourself using the `./hack/update-codegen.sh` script.

The `update-codegen` script will automatically generate the following files &
directories:

* `pkg/apis/samplecontroller/v1alpha1/zz_generated.deepcopy.go`
* `pkg/generated/`

Changes should not be made to these files manually, and when creating your own
controller based off of this implementation you should not copy these files and
instead run the `update-codegen` script to generate your own.