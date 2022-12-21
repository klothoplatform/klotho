package input

import (
	"encoding/json"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/klothoplatform/klotho/pkg/config"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/lang/dockerfile"
	"github.com/klothoplatform/klotho/pkg/lang/golang"
	"github.com/klothoplatform/klotho/pkg/lang/javascript"
	"github.com/klothoplatform/klotho/pkg/lang/python"
	"github.com/klothoplatform/klotho/pkg/lang/yaml"
	"github.com/klothoplatform/klotho/pkg/logging"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

type fileOpener[F core.File] func(path string, content io.Reader) (f F, err error)

func Upcast[F core.File](o fileOpener[F]) fileOpener[core.File] {
	return func(path string, content io.Reader) (f core.File, err error) {
		return o(path, content)
	}
}

type languageFiles struct {
	name             languageName
	foundSources     bool
	foundPackageFile bool
	packageFileName  string
	// packageFileOpener reads a package file (as opposed to a source file).
	// It is specialized to `core.File` so that the `languageFiles` struct doesn't need to be generic.
	// If you have a func that returns a more specfic type, use Upcast to convert it to `fileOpener[core.File]`.
	packageFileOpener fileOpener[core.File]
}

type languageName string

const (
	JavaScript languageName = "JavaScript"
	Python     languageName = "Python"
	Go         languageName = "Go"
	Yaml       languageName = "Yaml"
	DockerFile languageName = "Dockerfile"
)

func ReadOSDir(cfg config.Application, cfgFilePath string) (*core.InputFiles, error) {
	var root string
	root, cfg.Path = splitPathRoot(cfg.Path)
	zap.S().Debugf("Resolved root='%s' and search-path='%s'", root, cfg.Path)
	return ReadDir(os.DirFS(root), cfg, cfgFilePath)
}

func splitPathRoot(cfgPath string) (root, path string) {
	// Start by cleaning the path to remove craziness like "././"
	// or redundant things like 'a/../b' (-> 'b') or './a' (-> 'a')
	cfgPath = filepath.Clean(cfgPath)

	if filepath.IsAbs(cfgPath) {
		if vol := filepath.VolumeName(cfgPath); vol != "" {
			// windows-y
			root = vol
		} else {
			// unix-y
			root = "/"
		}
		path = strings.TrimPrefix(cfgPath, root)
		return
	}

	// Relative paths that start with '../' or './' are invalid, so we must normalize those
	pathList := strings.Split(cfgPath, string(filepath.Separator))
	if pathList[0] == ".." {
		i := 0
		for ; i < len(pathList) && pathList[i] == ".."; i++ {
		}
		root = filepath.Join(pathList[:i]...)
		path = filepath.Join(pathList[i:]...)
	} else {
		root = "."
		path = cfgPath
	}
	return
}

func ReadDir(fsys fs.FS, cfg config.Application, cfgFilePath string) (*core.InputFiles, error) {
	input := new(core.InputFiles)

	// Need to check for tsconfig before WalkDir to make sure it's read first before any JS files.
	tsConfigPath := filepath.Join(cfg.Path, "tsconfig.json")
	var tsConfig struct {
		CompilerOptions struct {
			OutDir string `json:"outDir"`
		} `json:"compilerOptions"`
	}
	if f, err := fsys.Open(tsConfigPath); err == nil {
		err = json.NewDecoder(f).Decode(&tsConfig)
		if err != nil {
			return nil, errors.Wrapf(err, "could not decode TS config at '%s'", tsConfigPath)
		}
		tsConfig.CompilerOptions.OutDir = filepath.Join(cfg.Path, tsConfig.CompilerOptions.OutDir)
		zap.S().Debugf("Read TS config (%s): %+v", tsConfigPath, tsConfig)
	}

	jsLang := &languageFiles{name: JavaScript, packageFileName: "package.json", packageFileOpener: Upcast(javascript.NewPackageFile)}
	pyLang := &languageFiles{name: Python, packageFileName: "requirements.txt", packageFileOpener: Upcast(python.NewRequirementsTxt)}
	goLang := &languageFiles{name: Go, packageFileName: "go.mod", packageFileOpener: Upcast(golang.NewGoMod)}
	yamlLang := &languageFiles{name: Yaml}
	dockerfileLang := &languageFiles{name: DockerFile}
	allLangs := []*languageFiles{jsLang, pyLang, goLang, yamlLang}
	err := fs.WalkDir(fsys, cfg.Path, func(path string, info fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(cfg.Path, path)
		if err != nil {
			return err
		}
		if relPath == "." {
			// if the user passed in a single file, simply use the file's name
			// (so './dist/index.js' becomes 'index.js')
			relPath = info.Name()
		}
		var f core.File = &core.FileRef{FPath: relPath}

		if info.IsDir() {
			switch info.Name() {
			case "node_modules", "vendor":
				// Skip modules/vendor folder for performance.
				// If we ever support Klotho annotations from dependencies,
				// we'll need to remove this skip and check those.
				return fs.SkipDir

			case ".git", ".svn":
				return fs.SkipDir

			case cfg.OutDir:
				// Don't let previous compiled output as input
				return fs.SkipDir
			}
			if statFS, ok := fsys.(fs.StatFS); ok {
				checkPath := filepath.Join(path, "resources.json")
				if _, err = statFS.Stat(checkPath); err == nil {
					zap.L().With(logging.FileField(f)).Debug("detected klotho output directory, skipping")
					return fs.SkipDir
				}
			}
			return nil
		}
		isPackageFile := false
		for _, lang := range allLangs {
			if info.Name() == lang.packageFileName {
				f, err = addFile(fsys, path, relPath, lang.packageFileOpener)
				lang.foundPackageFile = true
				isPackageFile = true
				break
			}
		}
		if !isPackageFile {
			ext := filepath.Ext(info.Name())
			switch ext {
			case ".js":
				if tsConfig.CompilerOptions.OutDir != "" {
					tsPrefix := tsConfig.CompilerOptions.OutDir + string(os.PathSeparator)
					newPath := strings.TrimPrefix(path, tsPrefix) // tsPrefix is already joined to cfg.Path, so use `path` not `relPath`
					if relPath != newPath {
						zap.S().Debugf("Removing TS outdir from %s -> %s", relPath, newPath)
					}
					relPath = newPath
				}
				f, err = addFile(fsys, path, relPath, javascript.NewFile)
				jsLang.foundSources = true
			case ".py":
				// TODO we may need to do something similar to the js stuff above
				f, err = addFile(fsys, path, relPath, python.NewFile)
				pyLang.foundSources = true
			case ".go":
				// TODO we may need to do something similar to the js stuff above
				f, err = addFile(fsys, path, relPath, golang.NewFile)
				goLang.foundSources = true
			case ".yaml", ".yml":
				if path == cfgFilePath {
					return nil
				}
				f, err = addFile(fsys, path, relPath, yaml.NewFile)
				yamlLang.foundSources = true
			default:
				infoSections := strings.Split(info.Name(), ".")
				for _, sec := range infoSections {
					if strings.ToLower(sec) == "dockerfile" {
						f, err = addFile(fsys, path, relPath, dockerfile.NewFile)
						dockerfileLang.foundSources = true
						break
					}
				}

			}
		}
		if err != nil {
			return errors.Wrapf(err, "error reading '%s' (rel: '%s')", path, relPath)
		}
		zap.L().Debug("Read input file", logging.FileField(f))
		input.Add(f)
		return nil
	})

	if err != nil {
		return nil, err
	}
	for _, lang := range allLangs {
		if lang.foundSources && !lang.foundPackageFile && lang.packageFileName != "" {

			pkg, err := openFindUpward(lang.packageFileName, cfg.Path, fsys, lang.packageFileOpener)
			if err != nil {
				return nil, err
			}
			input.Add(pkg)
			zap.L().With(logging.FileField(pkg)).Sugar().Debugf("Read package file for %s", lang.name)
		}
	}

	return input, nil
}

// openFindUpward tries to open the `basename` file in `rootPath`, or any of its parent dirs up to `fsys`'s root.
func openFindUpward[F core.File](basename string, rootPath string, fsys fs.FS, opener fileOpener[F]) (core.File, error) {
	for pkgDir := rootPath; ; pkgDir = filepath.Dir(pkgDir) {
		pkgPath := filepath.Join(pkgDir, basename)
		f, err := fsys.Open(pkgPath)
		if errors.Is(err, fs.ErrNotExist) {
			if pkgDir == "/" || pkgDir == "." {
				break
			}
			continue
		}
		if err != nil {
			return nil, errors.Wrapf(err, "error looking upward for package file named '%s'", basename)
		}
		defer f.Close()
		return opener(basename, f)
	}
	return nil, errors.Errorf("No %s found", basename)
}

func addFile[F core.File](fsys fs.FS, path string, relPath string, opener fileOpener[F]) (core.File, error) {
	f, err := fsys.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	r, err := opener(relPath, f)
	if err != nil {
		return nil, err
	}
	return r, nil
}
