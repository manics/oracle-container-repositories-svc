module github.com/manics/binderhub-container-registry-helper

go 1.21

toolchain go1.21.6

require github.com/oracle/oci-go-sdk/v65 v65.67.1

require (
	github.com/aws/aws-sdk-go-v2 v1.28.0
	github.com/aws/aws-sdk-go-v2/config v1.27.19
	github.com/aws/aws-sdk-go-v2/service/ecr v1.28.6
	github.com/aws/aws-sdk-go-v2/service/sts v1.28.13
	github.com/prometheus/client_golang v1.19.1
	github.com/prometheus/common v0.54.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/aws/aws-sdk-go-v2/credentials v1.17.19 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.16.6 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.3.10 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.6.10 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.8.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.11.2 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.11.12 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.20.12 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.24.6 // indirect
	github.com/aws/smithy-go v1.20.2 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/gofrs/flock v0.8.1 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/prometheus/client_model v0.6.1 // indirect
	github.com/prometheus/procfs v0.13.0 // indirect
	github.com/sony/gobreaker v0.5.0 // indirect
	golang.org/x/sys v0.19.0 // indirect
	google.golang.org/protobuf v1.34.0 // indirect
)

replace github.com/manics/binderhub-container-registry-helper/oracle => ./oracle

replace github.com/manics/binderhub-container-registry-helper/amazon => ./amazon

replace github.com/manics/binderhub-container-registry-helper/common => ../common
