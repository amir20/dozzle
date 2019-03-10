module github.com/amir20/dozzle

replace github.com/docker/docker v0.0.0-20170601211448-f5ec1e2936dc => github.com/docker/engine v0.0.0-20180718150940-a3ef7e9a9bda

require (
	github.com/Azure/go-ansiterm v0.0.0-20170929234023-d6e3b3328b78 // indirect
	github.com/Microsoft/go-winio v0.4.12 // indirect
	github.com/Nvveen/Gotty v0.0.0-20120604004816-cd527374f1e5 // indirect
	github.com/beme/abide v0.0.0-20181227202223-4c487ef9d895
	github.com/cespare/reflex v0.2.0 // indirect
	github.com/docker/distribution v2.7.1+incompatible // indirect
	github.com/docker/docker v1.13.1
	github.com/docker/go-connections v0.4.0 // indirect
	github.com/docker/go-units v0.3.3 // indirect
	github.com/gobuffalo/envy v1.6.15 // indirect
	github.com/gobuffalo/packd v0.0.0-20190224160250-d04dd98aca5b // indirect
	github.com/gobuffalo/packr v1.22.0
	github.com/gobuffalo/syncx v0.0.0-20190224160051-33c29581e754 // indirect
	github.com/gogo/protobuf v1.2.1 // indirect
	github.com/golang/protobuf v1.3.0 // indirect
	github.com/google/go-cmp v0.2.0 // indirect
	github.com/gorilla/mux v1.7.0
	github.com/kballard/go-shellquote v0.0.0-20180428030007-95032a82bc51 // indirect
	github.com/konsorten/go-windows-terminal-sequences v1.0.2 // indirect
	github.com/magiconair/properties v1.8.0
	github.com/ogier/pflag v0.0.1 // indirect
	github.com/opencontainers/go-digest v1.0.0-rc1 // indirect
	github.com/opencontainers/image-spec v1.0.1 // indirect
	github.com/rogpeppe/go-internal v1.2.2 // indirect
	github.com/sirupsen/logrus v1.3.0
	github.com/spf13/pflag v1.0.3
	github.com/stretchr/testify v1.3.0
	golang.org/x/crypto v0.0.0-20190308221718-c2843e01d9a2 // indirect
	golang.org/x/net v0.0.0-20190301231341-16b79f2e4e95 // indirect
	golang.org/x/sys v0.0.0-20190309122539-980fc434d28e // indirect
	golang.org/x/text v0.3.1-0.20180807135948-17ff2d5776d2 // indirect
	golang.org/x/time v0.0.0-20190308202827-9d24e82272b4 // indirect
	google.golang.org/genproto v0.0.0-20190307195333-5fe7a883aa19 // indirect
	gotest.tools v2.2.0+incompatible // indirect
)

// github.com/docker/engine v18.06.1-ce
replace github.com/docker/docker => github.com/docker/engine v0.0.0-20180816081446-320063a2ad06

// github.com/docker/distribution master
// a proper tagged release is expected in early fall(September 2018)
// see; https://github.com/docker/distribution/issues/2693
replace github.com/docker/distribution => github.com/docker/distribution v2.6.0-rc.1.0.20180820212402-02bf4a2887a4+incompatible
