package javascript

import (
	"bytes"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/klothoplatform/klotho/pkg/filter"

	"github.com/klothoplatform/klotho/pkg/annotation"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/logging"
	"github.com/klothoplatform/klotho/pkg/multierr"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

type (
	Pubsub struct {
		runtime Runtime

		result *core.CompilationResult
		deps   *core.Dependencies

		emitters             map[VarSpec]*emitterValue
		proxyGenerationCalls []proxyGenerationCall
	}

	emitterValue struct {
		Publishers  []emitterUsage
		Subscribers []emitterUsage
		Resource    *core.PubSub
	}

	emitterUsage struct {
		filePath string
		event    string
		unitId   string
	}

	proxyGenerationCall struct {
		filepath string
		emitters []EmitterSubscriberProxyEntry
	}
)

const pubsubVarType = "EventEmitter"
const pubsubVarTypeModule = "events"

func (p Pubsub) Name() string { return "Pubsub" }

func (p Pubsub) Transform(result *core.CompilationResult, deps *core.Dependencies) error {
	p.result = result

	p.deps = deps
	p.emitters = make(map[VarSpec]*emitterValue)

	var errs multierr.Error
	for _, unit := range core.GetResourcesOfType[*core.ExecutionUnit](result) {
		vars := DiscoverDeclarations(unit.Files(), pubsubVarType, pubsubVarTypeModule, true, FilterByCapability(annotation.PubSubCapability))
		for spec, value := range vars {
			if value.Annotation.Capability.ID == "" {
				errs.Append(core.NewCompilerError(value.File, value.Annotation, errors.New("'id' is required")))
			}
			if _, ok := p.emitters[spec]; !ok {
				resource := core.PubSub{
					Path: spec.DefinedIn,
					Name: value.Annotation.Capability.ID,
				}
				p.emitters[spec] = &emitterValue{
					Resource: &resource,
				}
			}
		}

		err := p.findProxiesNeeded(unit)
		if err != nil {
			errs.Append(err)
			continue
		}

		varsByFile := vars.SplitByFile()
		if len(vars) == 0 {
			continue
		}

		err = p.rewriteEmitters(unit, varsByFile)
		if err != nil {
			errs.Append(err)
		}

	}

	for _, call := range p.proxyGenerationCalls {
		err := p.generateProxies(call.filepath, call.emitters)
		if err != nil {
			errs.Append(err)
		}
	}

	for _, v := range p.emitters {
		p.result.Add(v.Resource)
	}

	err := p.generateEmitterDefinitions()
	errs.Append(err)

	return errs.ErrOrNil()
}

func (p *Pubsub) rewriteEmitters(unit *core.ExecutionUnit, fileVars map[string]VarDeclarations) error {
	var errs multierr.Error

	for _, f := range unit.Files() {
		js, ok := Language.ID.CastFile(f)
		if !ok {
			continue
		}
		err := p.rewriteFileEmitters(js, fileVars[js.Path()])
		if err != nil {
			errs.Append(core.WrapErrf(err, "failed to handle pubsub in unit %s", unit.Name))
		}
	}

	return errs.ErrOrNil()
}

var emitterRE = regexp.MustCompile(`(?:\w+\.)?EventEmitter\(.*\)`)

const emitterRTName = "emitter"

func (p *Pubsub) rewriteFileEmitters(f *core.SourceFile, vars VarDeclarations) error {
	if len(vars) == 0 {
		return nil
	}
	content := string(f.Program())
	var errs multierr.Error

	var err error
	content, err = EnsureRuntimeImport(f.Path(), emitterRTName, emitterRTName, content)
	if err != nil {
		errs.Append(core.WrapErrf(err, "could not create runtime import"))
	}

	for spec, varReference := range vars {
		expr := varReference.Annotation.Node.Content()
		newExpr := emitterRE.ReplaceAllString(
			expr,
			fmt.Sprintf(`%sRuntime.Emitter("%s", "%s", "%s")`, emitterRTName, f.Path(), spec.VarName, varReference.Annotation.Capability.ID),
		)
		content = strings.ReplaceAll(content, expr, newExpr)
	}

	err = f.Reparse([]byte(content))
	errs.Append(err)
	return errs.ErrOrNil()
}

func (p *Pubsub) findProxiesNeeded(unit *core.ExecutionUnit) error {
	var errs multierr.Error
	for _, f := range unit.Files() {
		var emitters []EmitterSubscriberProxyEntry

		for spec, value := range p.emitters {
			log := zap.L().With(logging.FileField(f)).Sugar()
			js, ok := Language.ID.CastFile(f)
			if !ok {
				continue
			}

			pEvents := p.findPublisherTopics(js, spec)
			if len(pEvents) > 0 {
				for _, event := range pEvents {
					value.Resource.AddPublisher(event, unit.Key())
					value.Publishers = append(value.Publishers, emitterUsage{filePath: f.Path(), event: event, unitId: unit.Name})
					log.Debugf("Adding publisher to '%s'", event)
				}
				p.deps.Add(unit.Key(), value.Resource.Key())
				log.Infof("Found %d topics produced to %s#%s: %v", len(pEvents), spec.DefinedIn, spec.VarName, pEvents)
			}
			sEvents := p.findSubscriberTopics(js, spec)
			if len(sEvents) > 0 {
				for _, event := range sEvents {
					value.Resource.AddSubscriber(event, unit.Key())
					value.Subscribers = append(value.Subscribers, emitterUsage{filePath: f.Path(), event: event, unitId: unit.Name})
					log.Debugf("Adding subscriber to '%s'", event)
				}
				p.deps.Add(value.Resource.Key(), unit.Key())
				log.Infof("Found %d topics consumed from %s#%s: %v", len(sEvents), spec.DefinedIn, spec.VarName, sEvents)
				importPath, err := filepath.Rel(filepath.Dir(f.Path()), spec.DefinedIn)
				if err != nil {
					errs.Append(errors.Wrapf(err, "could not create relative import for %+v from '%s'", spec, f.Path()))
					continue
				}
				entry := EmitterSubscriberProxyEntry{
					ImportPath: FileToLocalModule(importPath),
					VarName:    spec.VarName,
					Events:     sEvents,
				}
				emitters = append(emitters, entry)
			}
			if len(pEvents) > 0 || len(sEvents) > 0 {
				err := p.runtime.AddPubsubRuntimeFiles(unit)
				if err != nil {
					if err != nil {
						errs.Append(err)
						continue
					}
				}
			}
		}

		p.proxyGenerationCalls = append(p.proxyGenerationCalls, proxyGenerationCall{
			filepath: f.Path(),
			emitters: emitters,
		})
	}
	return errs.ErrOrNil()
}

func (p *Pubsub) generateProxies(filepath string, emitters []EmitterSubscriberProxyEntry) (err error) {
	if len(emitters) == 0 {
		return nil
	}

	tData := EmitterSubscriberProxy{
		Path:    filepath,
		Entries: emitters,
	}
	// The template adds the `/dispatcher` so just find the root runtime directory
	tData.RuntimeImport, err = RuntimePath(filepath, "")
	if err != nil {
		return
	}
	buf := new(bytes.Buffer)
	err = tmplPubsubProxy.Execute(buf, tData)
	if err != nil {
		return err
	}
	// Because we're going to write the proxy content more than once
	// record it in bytes which aren't consumed (versus reading from the buffer, which does)
	proxyContent := buf.Bytes()
	f, err := NewFile(filepath, bytes.NewBuffer(proxyContent))
	if err != nil {
		return err
	}
	log := zap.L().With(logging.FileField(f)).Sugar()
	var merr multierr.Error
	appendBuf := new(bytes.Buffer)
	for _, unit := range core.GetResourcesOfType[*core.ExecutionUnit](p.result) {
		if existing := unit.Get(filepath); existing != nil {
			existing := existing.(*core.SourceFile)
			if bytes.Contains(existing.Program(), []byte(emitters[0].VarName)) {
				log.Debugf("File already contains var in unit %s", unit.Name)
				continue
			}
			appendBuf.Reset()
			appendBuf.Write(existing.Program())
			appendBuf.WriteRune('\n')
			appendBuf.Write(proxyContent)
			err = existing.Reparse(appendBuf.Bytes())
			if err != nil {
				merr.Append(err)
			}
			log.Debugf("Appending pubsub proxy to unit %s", unit.Name)
		} else {
			unit.Add(f)
		}
	}
	return merr.ErrOrNil()
}

// generateEmitterDefinitions handles making sure the emitters are defined in all the execution units its used in at the path they are originally defined in,
// even if the original definition is in a file marked only for a single execution unit. It also make sure that the subscribers are imported to register their handlers.
func (p *Pubsub) generateEmitterDefinitions() (err error) {
	for _, unit := range core.GetResourcesOfType[*core.ExecutionUnit](p.result) {
		emittersByFile := make(map[string]map[VarSpec]*emitterValue)
		for spec, emitter := range p.emitters {
			f, ok := emittersByFile[spec.DefinedIn]
			if !ok {
				f = make(map[VarSpec]*emitterValue)
				emittersByFile[spec.DefinedIn] = f
			}
			f[spec] = emitter
		}
		for path, emitters := range emittersByFile {
			f := unit.Get(path)
			var js *core.SourceFile
			if f != nil {
				// The file already exists, either because it was included in the execution unit
				// or if was previously generated from `generateProxies`
				// In this case, we can assume that it's a SourceFile, and specifically one whose
				// `LanguageId` is `Language.ID`: we populate this map from js files that had emitters.
				js = f.(*core.SourceFile)
				if err := EnsureRuntimeImportFile(emitterRTName, emitterRTName, js); err != nil {
					return err
				}
			} else {
				content := `/**
* klotho generated
* This file is generated to make the emitter(s) available to publishers/subscribers in this execution unit.
*/
`
				content, err = EnsureRuntimeImport(path, emitterRTName, emitterRTName, content)
				if err != nil {
					return err
				}

				js, err = NewFile(path, strings.NewReader(content))
				if err != nil {
					return err
				}
			}

			additional := "// klotho generated\n"
			hasAdditional := false
			for spec, emitter := range emitters {
				emitterExport := FindExportForVar(js.Tree().RootNode(), spec.VarName)
				if emitterExport == nil {
					// If the file already existed as original, the export should already be there
					// If the file already existed but was a proxy, we need to add the emitter creation & export
					// If the file didn't exist, we'll definitely need to add the emitter creation
					additional += fmt.Sprintf(`exports.%s = new %sRuntime.Emitter("%s", "%s")
`, spec.VarName, emitterRTName, js.Path(), spec.VarName)
					hasAdditional = true
				}
				addImports := "// import subscribers to make sure they register handlers:\n"
				hasAddImports := false
				for _, sub := range emitter.Subscribers {
					if sub.filePath == js.Path() {
						continue
					}
					modulePath, err := filepath.Rel(filepath.Dir(js.Path()), sub.filePath)
					if err != nil {
						return err
					}
					imports := FindImportsInFile(js)
					if filtered := imports.Filter(filter.NewSimpleFilter(IsRelativeImportOfModule(modulePath))); len(filtered) > 0 {
						continue
					}
					// Make sure subscribers are imported so they register their handlers.
					// In the original code, these may be imported elsewhere, but since execution units
					// that are invoked from the emitter go directly to the emitter's definition file,
					// this makes sure, since this is where the emitter is defined, that the handlers are imported
					// no matter what.
					addImports += fmt.Sprintf(`require('%s')
`, FileToLocalModule(modulePath))
					hasAddImports = true
				}
				hasAdditional = hasAdditional || hasAddImports
			}
			if hasAdditional {
				if err = js.Reparse(append(js.Program(), []byte(additional)...)); err != nil {
					return err
				}
				unit.Add(js)
			}
		}
	}
	return nil
}

func findTopics(f *core.SourceFile, spec VarSpec, query string, methodName string) (topics []string) {
	varName := findVarName(f, spec)
	if varName == "" {
		return
	}
	next := DoQuery(f.Tree().RootNode(), query)
	for {
		match, found := next()
		if !found {
			break
		}
		switch {
		case match["func"].Content() != methodName,
			match["emitter"].Content() != varName:
			continue
		}
		topic := StringLiteralContent(match["topic"])
		topics = append(topics, topic)
	}

	return
}

func (p *Pubsub) findPublisherTopics(f *core.SourceFile, spec VarSpec) (topics []string) {
	return findTopics(f, spec, pubsubPublisher, "emit")
}

func (p *Pubsub) findSubscriberTopics(f *core.SourceFile, spec VarSpec) (topics []string) {
	return findTopics(f, spec, pubsubSubscriber, "on")
}
