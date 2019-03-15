module github.com/amir20/dozzle

replace github.com/docker/docker v0.0.0-20170601211448-f5ec1e2936dc => github.com/docker/engine v0.0.0-20180718150940-a3ef7e9a9bda

require (
	github.com/beme/abide v0.0.0-20181227202223-4c487ef9d895
	github.com/docker/distribution v0.0.0-00010101000000-000000000000 // indirect
	github.com/docker/docker v1.13.1
	github.com/docker/go-connections v0.4.0 // indirect
	github.com/docker/go-units v0.3.3 // indirect
	github.com/gobuffalo/packr v1.22.0
	github.com/gogo/protobuf v1.2.1 // indirect
	github.com/gorilla/mux v1.7.0
	github.com/opencontainers/go-digest v1.0.0-rc1 // indirect
	github.com/opencontainers/image-spec v1.0.1 // indirect
	github.com/sirupsen/logrus v1.3.0
	github.com/spf13/pflag v1.0.3
	github.com/stretchr/testify v1.3.0
)

// github.com/docker/engine v18.06.1-ce
replace github.com/docker/docker => github.com/docker/engine v0.0.0-20180816081446-320063a2ad06

// github.com/docker/distribution master
// a proper tagged release is expected in early fall(September 2018)
// see; https://github.com/docker/distribution/issues/2693
replace github.com/docker/distribution => github.com/docker/distribution v2.6.0-rc.1.0.20180820212402-02bf4a2887a4+incompatible
