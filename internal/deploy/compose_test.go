package deploy

import (
	"context"
	"testing"

	composetypes "github.com/compose-spec/compose-go/v2/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseCompose_BasicServices(t *testing.T) {
	data := []byte(`
services:
  web:
    image: nginx:latest
    ports:
      - "8080:80"
    environment:
      - FOO=bar
      - BAZ=qux
    volumes:
      - ./html:/usr/share/nginx/html:ro
    depends_on:
      - db
    restart: unless-stopped
    labels:
      app: web
  db:
    image: postgres:16
    environment:
      POSTGRES_PASSWORD: secret
    volumes:
      - pgdata:/var/lib/postgresql/data
    restart: always

volumes:
  pgdata:

networks:
  default:
    driver: bridge
`)

	project, err := ParseCompose(context.Background(), data, "myapp")
	require.NoError(t, err)

	assert.Equal(t, "myapp", project.Name)
	assert.Len(t, project.Services, 2)

	web, err := project.GetService("web")
	require.NoError(t, err)
	assert.Equal(t, "nginx:latest", web.Image)
	assert.Len(t, web.Ports, 1)
	assert.Equal(t, uint32(80), web.Ports[0].Target)
	assert.Equal(t, "8080", web.Ports[0].Published)
	assert.Equal(t, "bar", *web.Environment["FOO"])
	assert.Equal(t, "qux", *web.Environment["BAZ"])
	assert.Contains(t, web.DependsOn, "db")
	assert.Equal(t, "unless-stopped", web.Restart)
	assert.Equal(t, "web", web.Labels["app"])

	db, err := project.GetService("db")
	require.NoError(t, err)
	assert.Equal(t, "postgres:16", db.Image)
	assert.Equal(t, "secret", *db.Environment["POSTGRES_PASSWORD"])
	assert.Equal(t, "always", db.Restart)

	assert.Contains(t, project.Volumes, "pgdata")
	assert.Equal(t, "bridge", project.Networks["default"].Driver)
}

func TestParseCompose_MapDependsOn(t *testing.T) {
	data := []byte(`
services:
  web:
    image: nginx
    depends_on:
      db:
        condition: service_healthy
      redis:
        condition: service_started
  db:
    image: postgres
  redis:
    image: redis
`)
	project, err := ParseCompose(context.Background(), data, "test")
	require.NoError(t, err)

	web, _ := project.GetService("web")
	assert.Contains(t, web.DependsOn, "db")
	assert.Contains(t, web.DependsOn, "redis")
}

func TestParseCompose_ServiceNetworksList(t *testing.T) {
	data := []byte(`
services:
  web:
    image: nginx
    networks:
      - frontend
      - backend
networks:
  frontend:
  backend:
`)
	project, err := ParseCompose(context.Background(), data, "test")
	require.NoError(t, err)

	web, _ := project.GetService("web")
	assert.Contains(t, web.Networks, "frontend")
	assert.Contains(t, web.Networks, "backend")
}

func TestParseCompose_ServiceNetworksMapWithAliases(t *testing.T) {
	data := []byte(`
services:
  web:
    image: nginx
    networks:
      frontend:
        aliases:
          - web-alias
      backend: {}
networks:
  frontend:
  backend:
`)
	project, err := ParseCompose(context.Background(), data, "test")
	require.NoError(t, err)

	web, _ := project.GetService("web")
	assert.Len(t, web.Networks, 2)
	assert.Equal(t, []string{"web-alias"}, web.Networks["frontend"].Aliases)
}

func TestParseCompose_CommandStringAndSlice(t *testing.T) {
	data := []byte(`
services:
  string_cmd:
    image: alpine
    command: echo hello world
  slice_cmd:
    image: alpine
    command:
      - echo
      - hello
      - world
`)
	project, err := ParseCompose(context.Background(), data, "test")
	require.NoError(t, err)

	strCmd, _ := project.GetService("string_cmd")
	assert.Equal(t, []string{"echo", "hello", "world"}, []string(strCmd.Command))

	sliceCmd, _ := project.GetService("slice_cmd")
	assert.Equal(t, []string{"echo", "hello", "world"}, []string(sliceCmd.Command))
}

func TestParseCompose_EnvironmentMap(t *testing.T) {
	data := []byte(`
services:
  web:
    image: nginx
    environment:
      FOO: bar
      BAZ: "123"
`)
	project, err := ParseCompose(context.Background(), data, "test")
	require.NoError(t, err)

	web, _ := project.GetService("web")
	assert.Equal(t, "bar", *web.Environment["FOO"])
	assert.Equal(t, "123", *web.Environment["BAZ"])
}

func TestParseCompose_LabelsAsList(t *testing.T) {
	data := []byte(`
services:
  web:
    image: nginx
    labels:
      - "com.example.foo=bar"
      - "com.example.baz=qux"
`)
	project, err := ParseCompose(context.Background(), data, "test")
	require.NoError(t, err)

	web, _ := project.GetService("web")
	assert.Equal(t, "bar", web.Labels["com.example.foo"])
	assert.Equal(t, "qux", web.Labels["com.example.baz"])
}

func TestParseCompose_ExternalNetwork(t *testing.T) {
	data := []byte(`
services:
  web:
    image: nginx
    networks:
      - mynet
networks:
  mynet:
    external: true
`)
	project, err := ParseCompose(context.Background(), data, "test")
	require.NoError(t, err)

	assert.True(t, bool(project.Networks["mynet"].External))
}

func TestParseCompose_ExternalVolume(t *testing.T) {
	data := []byte(`
services:
  web:
    image: nginx
    volumes:
      - shared:/data
volumes:
  shared:
    external: true
`)
	project, err := ParseCompose(context.Background(), data, "test")
	require.NoError(t, err)

	assert.True(t, bool(project.Volumes["shared"].External))
}

func TestParseCompose_PortFormats(t *testing.T) {
	data := []byte(`
services:
  web:
    image: nginx
    ports:
      - "80"
      - "8080:80"
      - "8080:80/udp"
      - "127.0.0.1:8080:80"
`)
	project, err := ParseCompose(context.Background(), data, "test")
	require.NoError(t, err)

	web, _ := project.GetService("web")
	assert.Len(t, web.Ports, 4)

	assert.Equal(t, uint32(80), web.Ports[0].Target)
	assert.Equal(t, uint32(80), web.Ports[1].Target)
	assert.Equal(t, "8080", web.Ports[1].Published)
	assert.Equal(t, "udp", web.Ports[2].Protocol)
	assert.Equal(t, "127.0.0.1", web.Ports[3].HostIP)
}

func TestParseCompose_DependencyOrder(t *testing.T) {
	data := []byte(`
services:
  web:
    image: nginx
    depends_on:
      - api
  api:
    image: myapi
    depends_on:
      - db
      - redis
  db:
    image: postgres
  redis:
    image: redis
`)
	project, err := ParseCompose(context.Background(), data, "test")
	require.NoError(t, err)

	var order []string
	err = project.ForEachService(nil, func(name string, svc *composetypes.ServiceConfig) error {
		order = append(order, name)
		return nil
	})
	require.NoError(t, err)

	indexOf := func(name string) int {
		for i, n := range order {
			if n == name {
				return i
			}
		}
		return -1
	}

	assert.Less(t, indexOf("db"), indexOf("api"))
	assert.Less(t, indexOf("redis"), indexOf("api"))
	assert.Less(t, indexOf("api"), indexOf("web"))
}
