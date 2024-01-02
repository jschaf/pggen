// Package pgdocker creates one-off Postgres docker images to use so pggen can
// introspect the schema.
package pgdocker

import (
	"archive/tar"
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	dockerClient "github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/jackc/pgx/v4"
	"github.com/jschaf/pggen/internal/errs"
	"github.com/jschaf/pggen/internal/ports"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"text/template"
	"time"
)

// Client is a client to control the running Postgres Docker container.
type Client struct {
	docker      *dockerClient.Client
	containerID string // container ID if started, empty otherwise
	connString  string
}

// Start builds a Docker image and runs the image in a container.
func Start(ctx context.Context, initScripts []string) (client *Client, mErr error) {
	now := time.Now()
	dockerCl, err := dockerClient.NewClientWithOpts(dockerClient.FromEnv)
	if err != nil {
		return nil, fmt.Errorf("create client: %w", err)
	}
	c := &Client{docker: dockerCl}
	imageID, err := c.buildImage(ctx, initScripts)
	slog.DebugContext(ctx, "build image", slog.String("image_id", imageID))
	if err != nil {
		return nil, fmt.Errorf("build image: %w", err)
	}
	containerID, port, err := c.runContainer(ctx, imageID)
	if err != nil {
		return nil, fmt.Errorf("run container: %w", err)
	}
	// Enrich logs with Docker container logs.
	defer func() {
		if mErr != nil {
			logs, err := c.GetContainerLogs()
			if err != nil {
				mErr = errors.Join(mErr, err)
			} else {
				mErr = fmt.Errorf("%w\nContainer logs for container ID %s\n\n%s", mErr, containerID, logs)
			}
		}
	}()
	// Cleanup the container after we're done.
	defer func() {
		if client != nil && mErr != nil {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			if err := client.Stop(ctx); err != nil {
				slog.ErrorContext(ctx, "stop pgdocker client", slog.String("error", err.Error()))
			}
		}
	}()

	c.containerID = containerID
	c.connString = fmt.Sprintf("host=0.0.0.0 port=%d user=postgres", port)
	if err := c.waitIsReady(ctx); err != nil {
		return nil, fmt.Errorf("wait for postgres to be ready: %w", err)
	}
	slog.DebugContext(ctx, "started docker postgres", slog.Duration("start_duration", time.Since(now)))
	return c, nil
}

// GetContainerLogs returns a string of all stderr and stdout logs for a
// container. Useful to enrich output when pggen fails to start the Docker
// container.
func (c *Client) GetContainerLogs() (logs string, mErr error) {
	if c.containerID == "" {
		return "", nil
	}
	logsCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	logsR, err := c.docker.ContainerLogs(logsCtx, c.containerID, types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
	})
	defer errs.Capture(&mErr, logsR.Close, "close container logs")
	if err != nil {
		return "", fmt.Errorf("get container logs: %w", err)
	}
	bs, err := io.ReadAll(logsR)
	if err != nil {
		return "", fmt.Errorf("reall all container logs: %w", err)
	}
	return string(bs), nil
}

// buildImage creates a new Postgres Docker image with the given init scripts
// copied into the Postgres entry point.
func (c *Client) buildImage(ctx context.Context, initScripts []string) (id string, mErr error) {
	// Make each init script run in the order it was given using a numeric prefix.
	initTarNames := make([]string, len(initScripts))
	for i, script := range initScripts {
		initTarNames[i] = fmt.Sprintf("%03d_%s", i, filepath.Base(script))
	}

	// Create Dockerfile with template.
	dockerfileBuf := &bytes.Buffer{}
	tmpl, err := template.New("pgdocker").Parse(dockerfileTemplate)
	if err != nil {
		return "", fmt.Errorf("parse template: %w", err)
	}
	if err := tmpl.ExecuteTemplate(dockerfileBuf, "dockerfile", pgTemplate{
		InitScripts: initTarNames,
	}); err != nil {
		return "", fmt.Errorf("execute template: %w", err)
	}
	slog.DebugContext(ctx, "wrote template into buffer", slog.String("dockerfile", dockerfileBuf.String()))

	// Tar Dockerfile for build context.
	tarBuf := &bytes.Buffer{}
	tarW := tar.NewWriter(tarBuf)
	hdr := &tar.Header{Name: "Dockerfile", Size: int64(dockerfileBuf.Len())}
	if err := tarW.WriteHeader(hdr); err != nil {
		return "", fmt.Errorf("write dockerfile tar header: %w", err)
	}
	if _, err := tarW.Write(dockerfileBuf.Bytes()); err != nil {
		return "", fmt.Errorf("write dockerfile to tar: %w", err)
	}

	// Tar init scripts into build context.
	for i, script := range initScripts {
		tarName := initTarNames[i]
		if err := tarInitScript(tarW, script, tarName); err != nil {
			return "", fmt.Errorf("tar init file: %w", err)
		}
	}

	tarR := bytes.NewReader(tarBuf.Bytes())
	slog.DebugContext(ctx, "wrote tar dockerfile into buffer")

	// Send build request.
	opts := types.ImageBuildOptions{Dockerfile: "Dockerfile"}
	resp, err := c.docker.ImageBuild(ctx, tarR, opts)
	if err != nil {
		return "", fmt.Errorf("build postgres docker image: %w", err)
	}
	defer errs.Capture(&mErr, resp.Body.Close, "close image build response")
	response, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read image build response: %w", err)
	}

	// Match image ID.
	imageIDRegexp := regexp.MustCompile(`Successfully built ([a-z0-9]+)`)
	matches := imageIDRegexp.FindSubmatch(response)
	if len(matches) == 0 {
		return "", fmt.Errorf("unable find image ID in docker build output below:\n%s", string(response))
	}
	return string(matches[1]), nil
}

// tarInitScript writes the contents of an init script into the tar writer
// using tarName.
func tarInitScript(tarW *tar.Writer, script string, tarName string) (mErr error) {
	stat, err := os.Stat(script)
	if err != nil {
		return fmt.Errorf("stat docker postgres init script %s: %w", script, err)
	}
	hdr, err := tar.FileInfoHeader(stat, tarName)
	if err != nil {
		return fmt.Errorf("create tar file header: %w", err)
	}
	hdr.Name = tarName
	hdr.AccessTime = time.Time{}
	hdr.ChangeTime = time.Time{}
	hdr.ModTime = time.Time{}
	if err := tarW.WriteHeader(hdr); err != nil {
		return fmt.Errorf("write init script tar header: %w", err)
	}
	f, err := os.Open(script)
	if err != nil {
		return fmt.Errorf("read init script: %w", err)
	}
	defer errs.Capture(&mErr, f.Close, "close file to tar")
	if _, err := io.Copy(tarW, f); err != nil {
		return fmt.Errorf("copy init script to tar: %w", err)
	}
	return nil
}

// runContainer creates and starts a new Postgres container using imageID.
// The postgres port is mapped to an available port on the host system.
func (c *Client) runContainer(ctx context.Context, imageID string) (string, ports.Port, error) {
	port, err := ports.FindAvailable()
	if err != nil {
		return "", 0, fmt.Errorf("find available port: %w", err)
	}
	containerCfg := &container.Config{
		Image:        imageID,
		Env:          []string{"POSTGRES_HOST_AUTH_METHOD=trust"},
		ExposedPorts: nat.PortSet{"5432/tcp": struct{}{}},
		Cmd:          []string{"postgres", "-c", "fsync=off", "-c", "full_page_writes=off"},
	}
	hostCfg := &container.HostConfig{
		PortBindings: nat.PortMap{
			"5432/tcp": []nat.PortBinding{{HostIP: "0.0.0.0", HostPort: strconv.Itoa(port)}},
		},
		Tmpfs: map[string]string{"/var/lib/postgresql/data": ""},
	}
	resp, err := c.docker.ContainerCreate(ctx, containerCfg, hostCfg, nil, nil, "")
	if err != nil {
		return "", 0, fmt.Errorf("create container: %w", err)
	}
	containerID := resp.ID
	slog.DebugContext(ctx, "created postgres container", slog.String("container_id", containerID), slog.Int("port", port))
	err = c.docker.ContainerStart(ctx, containerID, types.ContainerStartOptions{})
	if err != nil {
		return "", 0, fmt.Errorf("start container: %w", err)
	}
	slog.DebugContext(ctx, "started container", slog.String("container_id", containerID))
	return containerID, port, nil
}

// waitIsReady waits until we can connect to the database.
func (c *Client) waitIsReady(ctx context.Context) error {
	connString, _ := c.ConnString()
	cfg, err := pgx.ParseConfig(connString + " connect_timeout=1")
	if err != nil {
		return fmt.Errorf("parse conn string: %w", err)
	}

	deadline := time.After(10 * time.Second)
	for {
		select {
		case <-deadline:
			return fmt.Errorf("postgres didn't start up with 10 seconds")
		case <-ctx.Done():
			return fmt.Errorf("postgres didn't start up before context expired")
		default:
			// continue
		}
		debounce := time.After(200 * time.Millisecond)
		conn, err := pgx.ConnectConfig(ctx, cfg)
		if err == nil {
			if err := conn.Close(ctx); err != nil {
				slog.DebugContext(ctx, "close postgres connection", slog.String("error", err.Error()))
			}
			return nil
		}
		slog.DebugContext(ctx, "attempted connection", slog.String("error", err.Error()))
		<-debounce
	}
}

// ConnString returns the connection string to connect to the started Postgres
// Docker container.
func (c *Client) ConnString() (string, error) {
	if c.connString == "" {
		return "", fmt.Errorf("conn string not set; did postgres start correctly")
	}
	return c.connString, nil
}

// Stop stops the running container, if any.
func (c *Client) Stop(ctx context.Context) error {
	if c.containerID == "" {
		return nil
	}
	if err := c.docker.ContainerStop(ctx, c.containerID, container.StopOptions{}); err != nil {
		return fmt.Errorf("stop container %s: %w", c.containerID, err)
	}
	err := c.docker.ContainerRemove(ctx, c.containerID, types.ContainerRemoveOptions{
		RemoveVolumes: true,
		RemoveLinks:   false,
		Force:         true,
	})
	if err != nil {
		return fmt.Errorf("remove container %s: %w", c.containerID, err)
	}
	return nil
}
