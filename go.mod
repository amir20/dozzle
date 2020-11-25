module github.com/amir20/dozzle

  replace github.com/docker/docker v0.0.0-20190827232753-32688a47f341 => github.com/docker/engine v0.0.0-20190827232753-32688a47f341

  // github.com/docker/engine v19.06.1-ce
  replace github.com/docker/docker => github.com/docker/engine v0.0.0-20190827232753-32688a47f341

  // github.com/docker/distribution master
  // a proper tagged release is expected in early fall(September 2018)
  // see; https://github.com/docker/distribution/issues/2693
  replace github.com/docker/distribution => github.com/docker/distribution v0.0.0-20190711223531-1fb7fffdb266

  require (
  github.com/Azure/go-ansiterm v0.0.0-20170929234023-d6e3b3328b78 // indirect
  github.com/Microsoft/go-winio v0.4.15 // indirect
  github.com/beme/abide v0.0.0-20190723115211-635a09831760
  github.com/containerd/containerd v1.4.1 // indirect
  github.com/docker/distribution v2.7.1+incompatible // indirect
  github.com/docker/docker v0.0.0-20190827232753-32688a47f341
  github.com/docker/go-connections v0.4.0 // indirect
  github.com/docker/go-units v0.4.0 // indirect
  github.com/fsnotify/fsnotify v1.4.9 // indirect
  github.com/gobuffalo/envy v1.9.0 // indirect
  github.com/gobuffalo/packd v1.0.0 // indirect
  github.com/gobuffalo/packr v1.30.1
  github.com/gogo/protobuf v1.3.1 // indirect
  github.com/golang/protobuf v1.4.3 // indirect
  github.com/gorilla/mux v1.8.0
  github.com/magiconair/properties v1.8.4
  github.com/mitchellh/mapstructure v1.3.3 // indirect
  github.com/morikuni/aec v0.0.0-20170113033406-39771216ff4c // indirect
  github.com/opencontainers/go-digest v1.0.0 // indirect
  github.com/opencontainers/image-spec v1.0.1 // indirect
  github.com/pelletier/go-toml v1.8.1 // indirect
  github.com/pkg/errors v0.9.1 // indirect
  github.com/rogpeppe/go-internal v1.6.2 // indirect
  github.com/sergi/go-diff v1.0.0 // indirect
  github.com/sirupsen/logrus v1.7.0
  github.com/spf13/afero v1.4.1 // indirect
  github.com/spf13/cast v1.3.1 // indirect
  github.com/spf13/jwalterweatherman v1.1.0 // indirect
  github.com/spf13/pflag v1.0.5
  github.com/spf13/viper v1.7.1
  github.com/stretchr/objx v0.2.0 // indirect
  github.com/stretchr/testify v1.6.1
  golang.org/x/net v0.0.0-20201110031124-69a78807bb2b // indirect
  golang.org/x/sys v0.0.0-20201119102817-f84b799fce68 // indirect
  golang.org/x/text v0.3.4 // indirect
  google.golang.org/genproto v0.0.0-20201119123407-9b1e624d6bc4 // indirect
  google.golang.org/grpc v1.33.2 // indirect
  google.golang.org/protobuf v1.25.0 // indirect
  gopkg.in/ini.v1 v1.62.0 // indirect
  gopkg.in/yaml.v2 v2.3.0 // indirect
  gotest.tools v2.2.0+incompatible // indirect
  )

  go 1.14
