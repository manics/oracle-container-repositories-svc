module github.com/manics/binderhub-container-registry-helper

go 1.20

require github.com/oracle/oci-go-sdk/v65 v65.44.0

require (
	github.com/aws/aws-sdk-go-v2 v1.20.0
	github.com/aws/aws-sdk-go-v2/config v1.18.31
	github.com/aws/aws-sdk-go-v2/service/ecr v1.18.14
	github.com/aws/aws-sdk-go-v2/service/sts v1.21.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/aws/aws-sdk-go-v2/credentials v1.13.30 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.13.7 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.1.37 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.4.31 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.3.38 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.9.31 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.13.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.15.0 // indirect
	github.com/aws/smithy-go v1.14.0 // indirect
	github.com/gofrs/flock v0.8.1 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/sony/gobreaker v0.5.0 // indirect
	golang.org/x/sys v0.10.0 // indirect
)

replace github.com/manics/binderhub-container-registry-helper/oracle => ./oracle

replace github.com/manics/binderhub-container-registry-helper/amazon => ./amazon

replace github.com/manics/binderhub-container-registry-helper/common => ../common
