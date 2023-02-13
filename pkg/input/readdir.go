package input

import (
	"encoding/json"
	"fmt"
	"github.com/klothoplatform/klotho/pkg/filter/predicate"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/klothoplatform/klotho/pkg/config"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/lang/csharp"
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
	foundProjectFile bool
	// projectFileOpener reads a package file (as opposed to a source file).
	// It is specialized to `core.File` so that the `languageFiles` struct doesn't need to be generic.
	// If you have a func that returns a more specific type, use Upcast to convert it to `fileOpener[core.File]`.
	projectFileOpener      fileOpener[core.File]
	projectFilePredicate   predicate.Predicate[string]
	projectFileDescription string
}

func (l languageFiles) isProjectFile(filepath string) bool {
	return l.projectFilePredicate != nil && l.projectFilePredicate(filepath)
}

func hasName(expected string) predicate.Predicate[string] {
	return func(s string) bool {
		return expected == s
	}
}

func hasSuffix(expected string) predicate.Predicate[string] {
	return func(name string) bool {
		return strings.HasSuffix(name, expected)
	}
}

type languageName string

const (
	JavaScript languageName = "JavaScript"
	Python     languageName = "Python"
	Go         languageName = "Go"
	CSharp     languageName = "C#"
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

	jsLang := &languageFiles{
		name:                   JavaScript,
		projectFilePredicate:   hasName("package.json"),
		projectFileDescription: "package.json",
		projectFileOpener:      Upcast(javascript.NewPackageFile)}
	pyLang := &languageFiles{
		name:                   Python,
		projectFilePredicate:   hasName("requirements.txt"),
		projectFileDescription: "requirements.txt",
		projectFileOpener:      Upcast(python.NewRequirementsTxt)}
	goLang := &languageFiles{
		name:                   Go,
		projectFilePredicate:   hasName("go.mod"),
		projectFileDescription: "go.mod",
		projectFileOpener:      Upcast(golang.NewGoMod)}
	csLang := &languageFiles{
		name:                   CSharp,
		projectFilePredicate:   hasSuffix(".csproj"),
		projectFileDescription: "MSBuild Project File (.csproj)",
		// TODO: project files in C# are currently unused, so no need to open & parse them.
		projectFileOpener: func(path string, content io.Reader) (f core.File, err error) { return &core.FileRef{FPath: path}, nil },
	}
	yamlLang := &languageFiles{name: Yaml}
	dockerfileLang := &languageFiles{name: DockerFile}
	allLangs := []*languageFiles{jsLang, pyLang, goLang, yamlLang, csLang}
	projectDirs := map[languageName][]string{}

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
		var f core.File = &core.FileRef{
			FPath:          relPath,
			RootConfigPath: cfg.Path,
		}

		if info.IsDir() {
			err = detectProjectsInDir(allLangs, fsys, path, projectDirs)
			if err != nil {
				return err
			}

			switch info.Name() {
			case "node_modules", "vendor":
				// Skip modules/vendor folder for performance.
				// If we ever support Klotho annotations from dependencies,
				// we'll need to remove this skip and check those.
				return fs.SkipDir
			case "bin", "obj":
				dir, _ := filepath.Split(info.Name())
				if dir == "" {
					dir = cfg.Path
				}
				for _, projectDir := range projectDirs[CSharp] {
					if dir == projectDir {
						zap.L().With(logging.FileField(f)).Debug("detected C# project output, skipping directory")
						return fs.SkipDir
					}
				}
			case ".idea", ".vscode":
				fallthrough
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
		isProjectFile := false
		for _, lang := range allLangs {
			if lang.isProjectFile(info.Name()) {
				f, err = addFile(fsys, path, relPath, lang.projectFileOpener)
				isProjectFile = true
				break
			}
		}
		if !isProjectFile {
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
				f, err = addFile(fsys, path, relPath, python.NewFile)
				pyLang.foundSources = true
			case ".go":
				f, err = addFile(fsys, path, relPath, golang.NewFile)
				goLang.foundSources = true
			case ".cs":
				f, err = addFile(fsys, path, relPath, csharp.NewFile)
				csLang.foundSources = true
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
		if lang.foundSources && !lang.foundProjectFile && lang.projectFilePredicate != nil {
			projectFile, err := openFindUpward(lang, cfg.Path, fsys)
			if err != nil {
				return nil, err
			}
			input.Add(projectFile)
			zap.L().With(logging.FileField(projectFile)).Sugar().Debugf("Read project file for %s", lang.name)
		}
	}

	return input, nil
}

func detectProjectsInDir(langs []*languageFiles, fsys fs.FS, dir string, projectDirs map[languageName][]string) error {
	entries, _ := fs.ReadDir(fsys, dir)
	for _, lang := range langs {
		for _, e := range entries {
			if lang.isProjectFile(e.Name()) {
				for _, projectDir := range projectDirs[lang.name] {
					if dir == projectDir {
						return fmt.Errorf("multiple '%s' files found in directory: %s", lang.projectFileDescription, dir)
					}
				}
				projectDirs[lang.name] = append(projectDirs[lang.name], dir)
				lang.foundProjectFile = true
			}
		}
	}
	return nil
}

// openFindUpward tries to open the `basename` file in `rootPath`, or any of its parent dirs up to `fsys`'s root.
func openFindUpward(lang *languageFiles, rootPath string, fsys fs.FS) (core.File, error) {
	for prjDir := rootPath; ; prjDir = filepath.Dir(prjDir) {
		entries, err := fs.ReadDir(fsys, prjDir)
		var projectFile core.File
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}

			if lang.isProjectFile(entry.Name()) {
				if projectFile != nil {
					return nil, fmt.Errorf("multiple '%s' files found in directory: %s", lang.projectFileDescription, prjDir)
				}

				prjFilePath := path.Join(prjDir, entry.Name())
				f, err := fsys.Open(prjFilePath)
				if err != nil {
					break
				}
				projectFile, err = func() (core.File, error) {
					defer f.Close()
					return lang.projectFileOpener(prjFilePath, f)
				}()
			}
		}
		if err != nil {
			return nil, errors.Wrapf(err, "error looking upward for project file")
		}
		if projectFile != nil {
			return projectFile, nil
		}
		if prjDir == "/" || prjDir == "." {
			break
		}
	}
	return nil, errors.Errorf("No %s file found", lang.projectFileDescription)
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
