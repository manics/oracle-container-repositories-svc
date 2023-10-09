module github.com/manics/binderhub-container-registry-helper

go 1.20

require github.com/oracle/oci-go-sdk/v65 v65.49.3

require (
	github.com/aws/aws-sdk-go-v2 v1.21.1
	github.com/aws/aws-sdk-go-v2/config v1.18.44
	github.com/aws/aws-sdk-go-v2/service/ecr v1.20.1
	github.com/aws/aws-sdk-go-v2/service/sts v1.23.1
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/aws/aws-sdk-go-v2/credentials v1.13.42 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.13.12 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.1.42 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.4.36 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.3.44 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.9.36 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.15.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.17.2 // indirect
	github.com/aws/smithy-go v1.15.0 // indirect
	github.com/gofrs/flock v0.8.1 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/sony/gobreaker v0.5.0 // indirect
	golang.org/x/sys v0.12.0 // indirect
)

replace github.com/manics/binderhub-container-registry-helper/oracle => ./oracle

replace github.com/manics/binderhub-container-registry-helper/amazon => ./amazon

replace github.com/manics/binderhub-container-registry-helper/common => ../common
