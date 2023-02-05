# BinderHub Container Registry Helper

[![Go](https://github.com/manics/binderhub-container-registry-helper/actions/workflows/build.yml/badge.svg)](https://github.com/manics/binderhub-container-registry-helper/actions/workflows/build.yml)

A microservice to help BinderHub work with Public cloud container registries.

Some cloud registries require a repository to be created before it can be used.
This micro-service provides a simple REST API to create repositories on demand, avoiding the need to integrate cloud provider libraries into BinderHub.

# Example

Build and run locally:

```
podman build -t oracle-container-repositories-svc .

podman run --rm -it \
  -eAUTH_TOKEN=secret-token \
  -eOCI_COMPARTMENT_ID=oci.compartment.id \
  -eOCI_GO_SDK_DEBUG=verbose \
  -v ./oci-config:/oci-config:ro,z \
  -v ./oci_api_key.pem:/oci_api_key.pem:ro,z \
  -p8080:8080 \
  oracle-container-repositories-svc /oci-config
```

- `AUTH_TOKEN`: Secret token used to authenticate callers
- `OCI_COMPARTMENT_ID`: OCI compartment or tenancy OCID
- `oci-config`: A file containing the [OCI configuration](https://docs.oracle.com/en-us/iaas/Content/API/Concepts/sdkconfig.htm).
- `oci_api_key.pem`: The private keyfile for the OCI user.

List repositories

```
curl -H'Authorization: Bearer secret-token' localhost:8080/repos
```

Create repository `test` (ignores repositories that already exist)

```
curl -XPOST -H'Authorization: Bearer secret-token' localhost:8080/repo/test
```

Get repository `test`

```
curl -H'Authorization: Bearer secret-token' localhost:8080/repo/test
```

Delete repository `test` (ignores repositories that don't exist)

```
curl -XDELETE -H'Authorization: Bearer secret-token' localhost:8080/repo/test
```

# BinderHub example

Deploy the Helm chart, see [`Values.yaml`](./helm-chart/values.yaml) for configuration options.

Append this example [BinderHub configuration](binderhub-example/binderhub_config.py) to your BinderHub `extraConfig` section.
For example:

```yaml
extraConfig:
  10-external-registry-helper: |
    <binderhub-example/binderhub_config.py>
```

# Development

Build and run

```
go run .
```

Add a new module

```
go mod tidy
```

### Debug logging

Set environment variable `OCI_GO_SDK_DEBUG={info,debug,verbose}`
