package linux

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/nix-community/go-nix/pkg/build"
	"github.com/nix-community/go-nix/pkg/derivation"
	"github.com/nix-community/go-nix/pkg/nixpath"
	oci "github.com/opencontainers/runtime-spec/specs-go"
)

const sandboxBuildDir = "/build"

const ociRuntime = "/nix/store/s6w2hxxiq4379dps0h2i5hcmzm2zxpf5-crun-1.4.5/bin/crun"

// const ociRuntime = "runc"

// nolint:gochecknoglobals
var envEscaper = strings.NewReplacer(
	"\\\\", "\\",
	"\\n", "\n",
	"\\r", "\r",
	"\\t", "\t",
	"\\\"", "\"",
)

// nolint:gochecknoglobals
var sandboxPaths = map[string]string{
	"/bin/sh": "/nix/store/kas8m76rr10h78hfl3yk66akdi08bkf9-busybox-static-x86_64-unknown-linux-musl-1.35.0/bin/busybox",
}

var _ build.Build = &OCIBuild{}

type OCIBuild struct {
	cmd     *exec.Cmd
	tmpDir  string // Path to mutable store that build outputs to
	started bool
}

func writeLines(writer io.StringWriter, lines ...string) error {
	for _, l := range lines {
		if _, err := writer.WriteString(l); err != nil {
			return err
		}

		if _, err := writer.WriteString("\n"); err != nil {
			return err
		}
	}

	return nil
}

func NewOCIBuild(ctx context.Context, drv *derivation.Derivation, buildInputs []string) (*OCIBuild, error) {
	// TODO: Call os.MkdirTemp
	tmpDir, err := filepath.Abs("./tmp")
	if err != nil {
		return nil, err
	}

	build := &OCIBuild{
		tmpDir: tmpDir,
	}

	// If this function returns an error, run the cleanup method on exit
	// doCleanup is set before successful return.
	doCleanup := true
	defer func() {
		if doCleanup {
			build.Close()
		}
	}()

	rootless := true //nolint:ifshort

	buildDir := filepath.Join(tmpDir, "builddir")
	rootFsDir := filepath.Join(tmpDir, "rootfs")

	// Create required file structure
	{
		err := os.Mkdir(tmpDir, 0o777)
		if err != nil {
			return nil, err
		}

		err = os.MkdirAll(filepath.Join(tmpDir, nixpath.StoreDir), 0o777)
		if err != nil {
			return nil, err
		}

		err = os.Mkdir(buildDir, 0o777)
		if err != nil {
			return nil, err
		}

		err = os.Mkdir(rootFsDir, 0o777)
		if err != nil {
			return nil, err
		}

		err = os.Mkdir(filepath.Join(rootFsDir, "etc"), 0o777)
		if err != nil {
			return nil, err
		}

		// /etc/passwd
		{
			f, err := os.Create(filepath.Join(rootFsDir, "etc", "passwd"))
			if err != nil {
				return nil, err
			}

			if err = writeLines(
				f,
				"root:x:0:0:Nix build user:0:/noshell",
				"nixbld:x:1000:100:Nix build user:/build:/noshell",
				"nobody:x:65534:65534:Nobody:/:/noshell",
			); err != nil {
				return nil, err
			}

			f.Close()
		}

		// /etc/group
		{
			f, err := os.Create(filepath.Join(rootFsDir, "etc", "group"))
			if err != nil {
				return nil, err
			}

			if err = writeLines(
				f,
				"root:x:0:",
				"nixbld:!:100:",
				"nogroup:x:65534:",
			); err != nil {
				return nil, err
			}

			if err = writeLines(
				f,
			); err != nil {
				return nil, err
			}

			f.Close()
		}
	}

	// Create OCI spec
	spec := &oci.Spec{
		Version: oci.Version,

		Process: &oci.Process{
			Terminal: false,
			User: oci.User{
				UID: 0,
				GID: 0,
			},
			Cwd: sandboxBuildDir,
			Capabilities: &oci.LinuxCapabilities{
				Bounding:    nil,
				Effective:   nil,
				Inheritable: nil,
				Permitted:   nil,
				Ambient:     nil,
			},
			Rlimits: []oci.POSIXRlimit{
				{
					Type: "RLIMIT_NOFILE",
					Hard: 1024,
					Soft: 1024,
				},
			},
			NoNewPrivileges: true,
		},

		Linux: &oci.Linux{
			Namespaces: []oci.LinuxNamespace{
				{
					Type: oci.PIDNamespace,
				},
				{
					Type: oci.IPCNamespace,
				},
				{
					Type: oci.UTSNamespace,
				},
				{
					Type: oci.MountNamespace,
				},
				{
					Type: oci.CgroupNamespace,
				},
			},
			MaskedPaths: []string{
				"/proc/kcore",
				"/proc/latency_stats",
				"/proc/timer_list",
				"/proc/timer_stats",
				"/proc/sched_debug",
				"/sys/firmware",
			},
			ReadonlyPaths: []string{
				"/proc/asound",
				"/proc/bus",
				"/proc/fs",
				"/proc/irq",
				"/proc/sys",
				"/proc/sysrq-trigger",
			},
		},

		Root: &oci.Root{
			Path:     rootFsDir,
			Readonly: true,
		},

		Hostname: "localhost",

		Mounts: []oci.Mount{
			{
				Destination: "/proc",
				Type:        "proc",
				Source:      "proc",
			},
			{
				Destination: "/dev",
				Type:        "tmpfs",
				Source:      "tmpfs",
				Options:     []string{"nosuid", "strictatime", "mode=755", "size=65536k"},
			},
			{
				Destination: "/dev/pts",
				Type:        "devpts",
				Source:      "devpts",
				Options: func() []string {
					options := []string{"nosuid", "noexec", "newinstance", "ptmxmode=0666", "mode=0620"}
					if rootless {
						return options
					}

					return append(options, "gid=5")
				}(),
			},
			{
				Destination: "/dev/shm",
				Type:        "tmpfs",
				Source:      "shm",
				Options:     []string{"nosuid", "noexec", "nodev", "mode=1777", "size=65536k"},
			},
			{
				Destination: "/dev/mqueue",
				Type:        "mqueue",
				Source:      "mqueue",
				Options:     []string{"nosuid", "noexec", "nodev"},
			},
			{
				Destination: "/sys",
				Type:        "none",
				Source:      "/sys",
				Options:     []string{"rbind", "nosuid", "noexec", "nodev", "ro"},
			},
			{
				Destination: "/sys/fs/cgroup",
				Type:        "cgroup",
				Source:      "cgroup",
				Options:     []string{"nosuid", "noexec", "nodev", "relatime", "ro"},
			},
			{
				Destination: "/tmp",
				Type:        "tmpfs",
				Source:      "tmpfs",
				Options:     []string{"nosuid", "noatime", "mode=700"},
			},

			// Mount /build, the scratch build directory
			{
				Destination: "/build",
				Type:        "none",
				Source:      buildDir,
				Options:     []string{"rbind", "rw"},
			},

			// # Mount /nix/store
			// It might seem counterintuitive that we mount the entire store
			// as writable, but it is what Nix has always done and scripts are expected to create
			// their outputs themselves.
			// If we created the output and bind mounted it there would be no way to detect if
			// a build fails to create one or more of it's outputs.
			{
				Destination: nixpath.StoreDir,
				Type:        "none",
				Source:      filepath.Join(tmpDir, nixpath.StoreDir),
				Options:     []string{"rbind", "rw"},
			},
		},
	}

	// Set build command
	spec.Process.Args = append(append(spec.Process.Args, drv.Builder), drv.Arguments...)

	// Populate env vars
	{
		spec.Process.Env = append(
			spec.Process.Env,
			"TMPDIR="+sandboxBuildDir,
			"TEMPDIR="+sandboxBuildDir,
			"TMP="+sandboxBuildDir,
			"TEMP="+sandboxBuildDir,
			"TERM=xterm-256color",
			"HOME=/homeless-shelter",
			"NIX_BUILD_TOP="+sandboxBuildDir,
			"NIX_BUILD_CORES=1",
			"NIX_LOG_FD=2",
			"NIX_STORE="+nixpath.StoreDir,
		)

		for key, value := range drv.Env {
			spec.Process.Env = append(spec.Process.Env, key+"="+envEscaper.Replace(value))
		}
	}

	// Allow user namespaces for rootless mode
	if rootless {
		spec.Linux.Namespaces = append(spec.Linux.Namespaces, oci.LinuxNamespace{
			Type: oci.UserNamespace,
		})
	}

	// Add mappings for rootless mode
	if rootless {
		spec.Linux.GIDMappings = []oci.LinuxIDMapping{
			{
				ContainerID: 0,
				HostID:      100,
				Size:        1,
			},
			{
				ContainerID: 1,
				HostID:      100000,
				Size:        65536,
			},
		}
		spec.Linux.UIDMappings = []oci.LinuxIDMapping{
			{
				ContainerID: 0,
				HostID:      1000,
				Size:        1,
			},
			{
				ContainerID: 1,
				HostID:      100000,
				Size:        65536,
			},
		}
	}

	// If fixed output allow networking
	if fixed := drv.GetFixedOutput(); fixed != nil {
		for _, file := range []string{"/etc/resolv.conf", "/etc/services", "/etc/hosts"} {
			if !pathExists(file) {
				continue
			}

			spec.Mounts = append(spec.Mounts, oci.Mount{
				Destination: file,
				Type:        "none",
				Source:      file,
				Options:     []string{"bind", "rprivate"},
			})
		}
	} else {
		spec.Linux.Namespaces = append(spec.Linux.Namespaces, oci.LinuxNamespace{
			Type: oci.NetworkNamespace,
		})
	}

	// Mount sandbox paths (such as /bin/sh)
	for destination, source := range sandboxPaths {
		spec.Mounts = append(spec.Mounts, oci.Mount{
			Destination: destination,
			Type:        "none",
			Source:      source,
			Options:     []string{"rbind", "ro"},
		})
	}

	// Mount input sources
	for _, inputSource := range drv.InputSources {
		spec.Mounts = append(spec.Mounts, oci.Mount{
			Destination: inputSource,
			Type:        "none",
			Source:      inputSource,
			Options:     []string{"rbind", "ro"},
		})
	}

	// Mount store paths of dependencies
	for _, buildInput := range buildInputs {
		spec.Mounts = append(spec.Mounts, oci.Mount{
			Destination: buildInput,
			Type:        "none",
			Source:      buildInput,
			Options:     []string{"rbind", "ro"},
		})
	}

	// # Platform unhandled (for now)
	// drv.pop("system")

	// Write out config.json
	{
		f, err := os.Create(filepath.Join(tmpDir, "config.json"))
		if err != nil {
			return nil, err
		}

		b, err := json.Marshal(spec)
		if err != nil {
			return nil, err
		}

		_, err = f.Write(b)
		if err != nil {
			return nil, err
		}
	}

	containerUUID, err := uuid.NewRandom()
	if err != nil {
		return nil, fmt.Errorf("error creating container uuid: %w", err)
	}

	build.cmd = exec.CommandContext(ctx, ociRuntime, "run", containerUUID.String()) // nolint:gosec
	{
		build.cmd.Dir = tmpDir
		build.cmd.Env = os.Environ() // TODO: Create environment from scratch
	}

	doCleanup = false

	return build, nil
}

func (o *OCIBuild) SetStderr(stderr io.Writer) error {
	if o.started {
		return fmt.Errorf("cannot set stderr: process already started")
	}

	o.cmd.Stderr = stderr

	return nil
}

func (o *OCIBuild) SetStdout(stdout io.Writer) error {
	if o.started {
		return fmt.Errorf("cannot set stdout: process already started")
	}

	o.cmd.Stdout = stdout

	return nil
}

func (o *OCIBuild) Start() error {
	o.started = true

	return o.cmd.Start()
}

func (o *OCIBuild) Wait() error {
	return o.cmd.Wait()
}

func (o *OCIBuild) Close() error {
	// TODO: Reinstate RemoveAll
	// return os.RemoveAll(o.tmpDir)
	return nil
}
