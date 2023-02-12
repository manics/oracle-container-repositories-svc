# BinderHub Container Registry Helper

[![Go](https://github.com/manics/binderhub-container-registry-helper/actions/workflows/build.yml/badge.svg)](https://github.com/manics/binderhub-container-registry-helper/actions/workflows/build.yml)

A microservice to help BinderHub work with Public cloud container registries.

Some cloud registries require a repository to be created before it can be used.
This micro-service provides a simple REST API to create repositories on demand, avoiding the need to integrate cloud provider libraries into BinderHub.

The following cloud provider registries are supported:

- [Oracle Cloud Infrastructure container registry](https://docs.oracle.com/en-us/iaas/Content/Registry/Concepts/registryoverview.htm)
- [Amazon Web Services Elastic Container Registry (Amazon ECR)](https://aws.amazon.com/ecr/)

## Build and run locally

You must install [Go 1.18](https://tip.golang.org/doc/go1.18).
If you are a Python developer using [Conda](https://docs.conda.io/en/latest/) or [Mamba](https://mamba.readthedocs.io/) and just want a quick way to install Go:

```
conda create -n go -c conda-forge go=1.18 go-cgo=1.18
conda activate go
```

```
make build
make test
```

Run with Oracle Cloud Infrastructure using a local [OCI configuration file `oci-config` and private key `oci_api_key.pem`](https://docs.oracle.com/en-us/iaas/Content/API/Concepts/sdkconfig.htm):

```
BINDERHUB_AUTH_TOKEN=secret-token ./binderhub-oracle oci-config
```

Run with Amazon Web Services using the local [AWS credentials](https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-files.html):

```
BINDERHUB_AUTH_TOKEN=secret-token ./binderhub-amazon
```

## API endpoints

List repositories

```
curl -H'Authorization: Bearer secret-token' localhost:8080/repos
```

Create repository `foo/test` (ignores repositories that already exist)

```
curl -XPOST -H'Authorization: Bearer secret-token' localhost:8080/repo/foo/test
```

Get repository `foo/test`

```
curl -H'Authorization: Bearer secret-token' localhost:8080/repo/foo/test
```

Delete repository `foo/test` (ignores repositories that don't exist)

```
curl -XDELETE -H'Authorization: Bearer secret-token' localhost:8080/repo/foo/test
```

## Build and run container

```
podman build -t binderhub-container-registry-helper .
```

```
podman run --rm -it \
  -eBINDERHUB_AUTH_TOKEN=secret-token \
  -eOCI_COMPARTMENT_ID=oci.compartment.id \
  -v ./oci-config:/oci-config:ro,z \
  -v ./oci_api_key.pem:/oci_api_key.pem:ro,z \
  -p8080:8080 \
  binderhub-container-registry-helper oracle /oci-config
```

## Running in the cloud

The recommended way to run this service is to use an IAM
[instance principal (Oracle Cloud)](https://blogs.oracle.com/developers/post/accessing-the-oracle-cloud-infrastructure-api-using-instance-principals)
or
[instance profile (AWS)](https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/iam-roles-for-amazon-ec2.html)
to authenticate with the cloud provider.

### Environment variables

The following environment variables are supported:

- `BINDERHUB_AUTH_TOKEN`: Secret token used to authenticate callers who should set the `Authorization: Bearer {BINDERHUB_AUTH_TOKEN}` header.
  Set `BINDERHUB_AUTH_TOKEN=""` to disable authentication.
- `RETURN_ERROR_DETAILS`: If set to `1` internal error details will be returned in the response body to clients. This may include internal configuration information, only enable this for internal use. Default `0`.

Amazon only:

- `AWS_REGISTRY_ID`: Registry ID to use for AWS ECR, only set this is you are not using the default registry for the AWS account.

Oracle cloud infrastructure only:

- `OCI_COMPARTMENT_ID`: OCI compartment or tenancy OCID if not the default.

## BinderHub example

Deploy the Helm chart, see [`Values.yaml`](./helm-chart/values.yaml) for configuration options.

Append this example [BinderHub configuration](binderhub-example/binderhub_config.py) to your BinderHub `extraConfig` section.
For example:

```yaml
extraConfig:
  10-external-registry-helper: |
    <binderhub-example/binderhub_config.py>
```

## Development

Build and run

```
make build
make test
```

For more detailed testing of a single module or test:

```
go test -v ./common/
go test -v ./common -run TestGetName
```

Add a new module

```
go mod tidy
```

### Debug logging

The Oracle Cloud SDK supports the environment variable `OCI_GO_SDK_DEBUG={info,debug,verbose}`.
Unfortunately the AWS SDK does not have an equivalent.
