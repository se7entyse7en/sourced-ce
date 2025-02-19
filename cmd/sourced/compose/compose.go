package compose

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/src-d/sourced-ce/cmd/sourced/compose/workdir"
	"github.com/src-d/sourced-ce/cmd/sourced/dir"

	"github.com/pkg/errors"
)

// dockerComposeVersion is the version of docker-compose to download
// if docker-compose isn't already present in the system
const dockerComposeVersion = "1.24.0"

var composeContainerURL = fmt.Sprintf("https://github.com/docker/compose/releases/download/%s/run.sh", dockerComposeVersion)

type Compose struct {
	bin string
}

func (c *Compose) Run(ctx context.Context, arg ...string) error {
	return c.RunWithIO(ctx, os.Stdin, os.Stdout, os.Stderr, arg...)
}

func (c *Compose) RunWithIO(ctx context.Context, stdin io.Reader,
	stdout, stderr io.Writer, arg ...string) error {
	cmd := exec.CommandContext(ctx, c.bin, arg...)

	dir, err := workdir.ActivePath()
	if err != nil {
		return err
	}

	if err := workdir.ValidatePath(dir); err != nil {
		return err
	}

	cmd.Dir = dir
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	return cmd.Run()
}

func NewCompose() (*Compose, error) {
	bin, err := getOrInstallComposeBinary()
	if err != nil {
		return nil, err
	}

	return &Compose{bin: bin}, nil
}

func getOrInstallComposeBinary() (string, error) {
	path, err := exec.LookPath("docker-compose")
	if err == nil {
		bin := strings.TrimSpace(path)
		if bin != "" {
			return bin, nil
		}
	}

	path, err = getOrInstallComposeContainer()
	if err != nil {
		return "", errors.Wrapf(err, "error while getting docker-compose container")
	}

	return path, nil
}

func getOrInstallComposeContainer() (string, error) {
	datadir, err := dir.Path()
	if err != nil {
		return "", err
	}

	dirPath := filepath.Join(datadir, "bin")
	path := filepath.Join(dirPath, fmt.Sprintf("docker-compose-%s.sh", dockerComposeVersion))

	if _, err := os.Stat(path); err == nil {
		return path, nil
	} else if !os.IsNotExist(err) {
		return "", err
	}

	err = os.MkdirAll(dirPath, os.ModePerm)
	if err != nil {
		return "", errors.Wrapf(err, "error while creating directory %s", dirPath)
	}

	if err := downloadCompose(path); err != nil {
		return "", errors.Wrapf(err, "error downloading %s", composeContainerURL)
	}

	cmd := exec.CommandContext(context.Background(), "chmod", "+x", path)
	if err := cmd.Run(); err != nil {
		return "", errors.Wrapf(err, "cannot change permission to %s", path)
	}

	return path, nil
}

func downloadCompose(path string) error {
	if runtime.GOOS == "windows" {
		return fmt.Errorf("compose in container is not compatible with Windows")
	}

	return dir.DownloadURL(composeContainerURL, path)
}

func Run(ctx context.Context, arg ...string) error {
	comp, err := NewCompose()
	if err != nil {
		return err
	}

	return comp.Run(ctx, arg...)
}

func RunWithIO(ctx context.Context, stdin io.Reader, stdout, stderr io.Writer, arg ...string) error {
	comp, err := NewCompose()
	if err != nil {
		return err
	}

	return comp.RunWithIO(ctx, stdin, stdout, stderr, arg...)
}
