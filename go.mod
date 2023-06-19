module github.com/manics/binderhub-container-registry-helper

go 1.18

require github.com/oracle/oci-go-sdk/v65 v65.41.0

require (
	github.com/aws/aws-sdk-go-v2 v1.17.4
	github.com/aws/aws-sdk-go-v2/config v1.18.13
	github.com/aws/aws-sdk-go-v2/service/ecr v1.18.3
	github.com/aws/aws-sdk-go-v2/service/sts v1.18.3
)

require (
	github.com/aws/aws-sdk-go-v2/credentials v1.13.13 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.12.22 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.1.28 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.4.22 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.3.29 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.9.22 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.12.2 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.14.2 // indirect
	github.com/aws/smithy-go v1.13.5 // indirect
	github.com/gofrs/flock v0.8.1 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/sony/gobreaker v0.5.0 // indirect
	golang.org/x/sys v0.6.0 // indirect
)

replace github.com/manics/binderhub-container-registry-helper/oracle => ./oracle

replace github.com/manics/binderhub-container-registry-helper/amazon => ./amazon

replace github.com/manics/binderhub-container-registry-helper/common => ../common
