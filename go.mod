module github.com/moralpriest/tela-gateway

go 1.24

require (
	github.com/aws/aws-lambda-go v1.54.0
	github.com/awslabs/aws-lambda-go-api-proxy v0.16.2
	github.com/civilware/Gnomon v0.0.0-20240403103529-8b2fdb2b3106
	github.com/civilware/tela v0.0.0-20250806221602-aa892d2ff8d4
	github.com/deroproject/derohe v0.0.0-20240405032004-bd300c0e086e
)

require (
	github.com/aws/aws-sdk-go-v2 v1.42.1 // indirect
	github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream v1.7.14 // indirect
	github.com/aws/aws-sdk-go-v2/config v1.32.30 // indirect
	github.com/aws/aws-sdk-go-v2/credentials v1.19.29 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.18.30 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.4.30 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.7.30 // indirect
	github.com/aws/aws-sdk-go-v2/internal/v4a v1.4.31 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.13.13 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/checksum v1.9.23 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.13.30 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/s3shared v1.19.31 // indirect
	github.com/aws/aws-sdk-go-v2/service/s3 v1.105.2 // indirect
	github.com/aws/aws-sdk-go-v2/service/signin v1.4.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.32.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.37.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.44.1 // indirect
	github.com/aws/smithy-go v1.27.3 // indirect
	github.com/blang/semver/v4 v4.0.0 // indirect
	github.com/caarlos0/env/v6 v6.10.1 // indirect
	github.com/cespare/xxhash v1.1.0 // indirect
	github.com/coder/websocket v1.8.12 // indirect
	github.com/creachadair/jrpc2 v0.35.4 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dchest/siphash v1.2.3 // indirect
	github.com/deroproject/graviton v0.0.0-20220130070622-2c248a53b2e1 // indirect
	github.com/fxamacker/cbor/v2 v2.4.0 // indirect
	github.com/go-logr/logr v1.2.3 // indirect
	github.com/go-logr/zapr v1.2.3 // indirect
	github.com/gorilla/websocket v1.5.0 // indirect
	github.com/klauspost/cpuid/v2 v2.2.6 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mgutz/ansi v0.0.0-20200706080929-d51e80ef957d // indirect
	github.com/minio/sha256-simd v1.0.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/robfig/cron/v3 v3.0.1 // indirect
	github.com/satori/go.uuid v1.2.0 // indirect
	github.com/segmentio/fasthash v1.0.3 // indirect
	github.com/sirupsen/logrus v1.9.3 // indirect
	github.com/stretchr/testify v1.8.4 // indirect
	github.com/templexxx/xorsimd v0.4.4 // indirect
	github.com/tjfoc/gmsm v1.4.1 // indirect
	github.com/valyala/histogram v1.2.0 // indirect
	github.com/x-cray/logrus-prefixed-formatter v0.5.2 // indirect
	github.com/x448/float16 v0.8.4 // indirect
	go.etcd.io/bbolt v1.3.7 // indirect
	go.uber.org/atomic v1.10.0 // indirect
	go.uber.org/multierr v1.9.0 // indirect
	go.uber.org/zap v1.24.0 // indirect
	golang.org/x/crypto v0.18.0 // indirect
	golang.org/x/net v0.20.0 // indirect
	golang.org/x/sync v0.5.0 // indirect
	golang.org/x/sys v0.16.0 // indirect
	golang.org/x/term v0.16.0 // indirect
	golang.org/x/xerrors v0.0.0-20231012003039-104605ab7028 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

// Official DERO Foundation tree (module path remains github.com/deroproject/derohe)
replace github.com/deroproject/derohe => github.com/DEROFDN/derohe v0.0.0-20260527071132-5cd042e8f541
