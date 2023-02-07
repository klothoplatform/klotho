package javascript

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/klothoplatform/klotho/pkg/filter/predicate"

	"github.com/klothoplatform/klotho/pkg/multierr"

	"github.com/klothoplatform/klotho/pkg/annotation"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/logging"
	"github.com/klothoplatform/klotho/pkg/query"
	"github.com/pkg/errors"
	sitter "github.com/smacker/go-tree-sitter"
	"go.uber.org/zap"
)

var persistMethodsThatNeedAwait = map[string]struct{}{
	"get":     {},
	"set":     {},
	"clear":   {},
	"delete":  {},
	"entries": {},
	"has":     {},
	"keys":    {},
	"values":  {},
}

type Persist struct {
	runtime Runtime
}

func (p Persist) Name() string { return "Persist" }

func (p Persist) Transform(result *core.CompilationResult, deps *core.Dependencies) error {
	persister := &persister{result: result, deps: deps, runtime: p.runtime}

	// It's important for this to happen first, before the code gets transformed; otherwise we won't find the "new Map()"s.
	// Please be careful before moving this loop, or putting anything before it. The call to findUnawaitedCalls(units)
	// assumes that the code has not yet been rewritten (e.g. to turn Maps into our runtime map classes).
	// This code could use a cleanup: see CloudCompilers/klotho#431
	for _, res := range result.Resources() {
		unit, ok := res.(*core.ExecutionUnit)
		if !ok {
			continue
		}
		persister.findUnawaitedCalls(unit)
	}

	var errs multierr.Error
	for _, unit := range core.GetResourcesOfType[*core.ExecutionUnit](result) {
		err := persister.handleFiles(unit)
		if err != nil {
			errs.Append(err)
			continue
		}
	}

	return errs.ErrOrNil()
}

func (p *persister) hasKvAnnotation(declaringFile *core.SourceFile, annot *core.Annotation) bool {
	if annot.Capability.Name != annotation.PersistCapability {
		return false
	}
	pType, _ := p.determinePersistType(declaringFile, annot)
	return pType == core.PersistKVKind
}

func (p *persister) findUnawaitedCalls(unit *core.ExecutionUnit) {
	vars := DiscoverDeclarations(unit.Files(), "Map", "", false, p.hasKvAnnotation)
	for _, f := range unit.Files() {
		js, ok := Language.ID.CastFile(f)
		if !ok {
			continue
		}
		for spec := range vars {
			for _, node := range p.findUnwaitedCallsInFile(js, spec) {
				log := zap.L().With(logging.NodeField(node), logging.FileField(js)).Sugar()
				log.Warnf("%s", errors.Errorf("Call is async, but is missing \"await\""))
			}
		}
	}
}

func (p *persister) findUnwaitedCallsInFile(js *core.SourceFile, spec VarSpec) (errs []*sitter.Node) {
	specVarName := findVarName(js, spec)
	next := DoQuery(js.Tree().RootNode(), methodInvocation)
	for {
		match, found := next()
		if !found {
			break
		}
		if match["var.name"].Content() != specVarName {
			continue
		}

		methodName := match["method.name"].Content()
		_, methodNeedsAwait := persistMethodsThatNeedAwait[methodName]
		callIsAwaited := (match["full"].Parent().Type() == "await_expression")
		if methodNeedsAwait && !callIsAwaited {
			errs = append(errs, match["full"])
		}
	}
	return
}

type persister struct {
	result  *core.CompilationResult
	deps    *core.Dependencies
	runtime Runtime
}

func (p *persister) handleFiles(unit *core.ExecutionUnit) error {
	var errs multierr.Error
	for _, f := range unit.Files() {
		js, ok := Language.ID.CastFile(f)
		if !ok {
			continue
		}

		resources, err := p.handleFile(js, unit)
		if err != nil {
			errs.Append(core.WrapErrf(err, "failed to handle persist in unit %s", unit.Name))
		}

		for _, r := range resources {
			p.result.Add(r)

			_, isReferencedByExecUnit := unit.Executable.SourceFiles[js.Path()]

			// a file containing capabilities without an execution unit indicates that the file's capabilities
			// are imported by execution units in one or more separate files
			if core.FileExecUnitName(js) != "" || isReferencedByExecUnit {
				p.deps.Add(core.ResourceKey{
					Name: unit.Name,
					Kind: core.ExecutionUnitKind,
				}, r.Key())
			}
		}
	}

	return errs.ErrOrNil()
}

func (p *persister) handleFile(f *core.SourceFile, unit *core.ExecutionUnit) ([]core.CloudResource, error) {
	annots := f.Annotations()
	var resources []core.CloudResource

	var errs multierr.Error
	for _, annot := range annots {
		cap := annot.Capability
		if cap.Name != annotation.PersistCapability {
			continue
		}

		if annot.Capability.Directives.Object(core.EnvironmentVariableDirective) != nil {
			// This is handled by envvar.EnvVarInjection
			continue
		}

		keyType, pResult := p.determinePersistType(f, annot)

		if len(cap.ID) == 0 {
			errs.Append(core.NewCompilerError(f, annot, errors.New("'id' is required")))
		}

		var resource core.CloudResource
		var err, runtimeErr, transformErr error
		switch keyType {
		case core.PersistKVKind:
			resource, transformErr = p.transformKV(f, annot, pResult)
			runtimeErr = p.runtime.AddKvRuntimeFiles(unit)
		case core.PersistFileKind:
			resource, transformErr = p.transformFS(f, annot, pResult)
			runtimeErr = p.runtime.AddFsRuntimeFiles(unit)
		case core.PersistSecretKind:
			resource, transformErr = p.transformSecret(f, annot, pResult)
			runtimeErr = p.runtime.AddSecretRuntimeFiles(unit)
		case core.PersistORMKind:
			resource, transformErr = p.transformORM(unit, f, annot, pResult)
			runtimeErr = p.runtime.AddOrmRuntimeFiles(unit)
		case core.PersistRedisClusterKind:
			resource, transformErr = p.transformRedis(unit, f, annot, pResult, keyType)
			runtimeErr = p.runtime.AddRedisClusterRuntimeFiles(unit)
		case core.PersistRedisNodeKind:
			resource, transformErr = p.transformRedis(unit, f, annot, pResult, keyType)
			runtimeErr = p.runtime.AddRedisNodeRuntimeFiles(unit)
		default:
			err = fmt.Errorf("type '%s' is invalid for the persist capability", keyType)
		}
		if err != nil {
			errs.Append(core.NewCompilerError(f, annot, err))
			continue
		}
		if transformErr != nil || runtimeErr != nil {
			if transformErr != nil {
				errs.Append(core.NewCompilerError(f, annot, transformErr))
			}
			if runtimeErr != nil {
				errs.Append(core.NewCompilerError(f, annot, runtimeErr))
			}
			continue
		}

		// Do this after the specific transforms so that `pResult` nodes aren't invalidated
		if err := p.runtime.TransformPersist(f, annot, keyType); err != nil {
			return nil, err
		}
		resources = append(resources, resource)
	}

	return resources, errs.ErrOrNil()
}

func (p *persister) transformSecret(file *core.SourceFile, cap *core.Annotation, secretR *persistResult) (core.CloudResource, error) {
	if err := file.ReplaceNodeContent(secretR.expression, "secretRuntime"); err != nil {
		return nil, err
	}

	// get secret file name
	secrets, err := p.querySecretName(file, secretR.name)
	if err != nil {
		return nil, err
	}

	result := &core.Secrets{
		Persist: core.Persist{
			Kind: core.PersistSecretKind,
			Name: cap.Capability.ID,
		},
		Secrets: secrets,
	}

	return result, nil
}

func (p *persister) transformFS(file *core.SourceFile, cap *core.Annotation, fsR *persistResult) (core.CloudResource, error) {
	if err := file.ReplaceNodeContent(fsR.expression, "fsRuntime.fs"); err != nil {
		return nil, errors.Wrap(err, "could not reparse FS transformation")
	}

	result := &core.Persist{
		Kind: core.PersistFileKind,
		Name: cap.Capability.ID,
	}

	return result, nil
}

func (p *persister) transformKV(file *core.SourceFile, cap *core.Annotation, kvR *persistResult) (core.CloudResource, error) {
	directives := cap.Capability.Directives

	mapString := "new keyvalueRuntime.dMap("
	if len(directives) > 0 {
		j, err := json.Marshal(directives)
		if err != nil {
			return nil, errors.Wrap(err, "could not marshal directives to json")
		}
		mapString += string(j)
	}
	mapString += ")"

	if err := file.ReplaceNodeContent(kvR.expression, mapString); err != nil {
		return nil, err
	}

	result := &core.Persist{
		Kind: core.PersistKVKind,
		Name: cap.Capability.ID,
	}

	return result, nil
}

func (p *persister) transformORM(unit *core.ExecutionUnit, file *core.SourceFile, cap *core.Annotation, ormR *persistResult) (core.CloudResource, error) {
	envVar := core.GenerateOrmConnStringEnvVar(cap.Capability.ID, string(core.PersistORMKind))

	var replaceContent string
	switch ormR.ormType {
	case TypeOrmKind:
		replaceContent = fmt.Sprintf(`ormRuntime.getDataSourceParams("%s", %s)`, envVar.Name, ormR.expression.Content())
	case SequelizeKind:
		replaceContent = fmt.Sprintf(`ormRuntime.getDBConn("%s")`, envVar.Name)
	default:
		return nil, errors.New("unrecognized")
	}

	if err := file.ReplaceNodeContent(ormR.expression, replaceContent); err != nil {
		return nil, err
	}

	result := &core.Persist{
		Kind: core.PersistORMKind,
		Name: cap.Capability.ID,
	}

	unit.EnvironmentVariables = append(unit.EnvironmentVariables, envVar)

	return result, nil
}

func (p *persister) transformRedis(unit *core.ExecutionUnit, file *core.SourceFile, cap *core.Annotation, redisR *persistResult, kind core.PersistKind) (core.CloudResource, error) {
	// Because the redis client can be initialized with () or ({...}) we have to have the expression match it all.
	// We need to remove the outer () so that the runtime will process these correctly.
	newExpression := strings.TrimLeft(redisR.expression.Content(), "(")
	newExpression = strings.TrimRight(newExpression, ")")

	if newExpression == "" {
		newExpression = "{}"
	}

	importName := "redis_node"
	if kind == core.PersistRedisClusterKind {
		importName = "redis_cluster"
	}
	hostEnvVar := core.GenerateRedisHostEnvVar(cap.Capability.ID, string(kind))
	portEnvVar := core.GenerateRedisPortEnvVar(cap.Capability.ID, string(kind))

	replaceContent := fmt.Sprintf(`(%sRuntime.getParams("%s", "%s", %s))`, importName, hostEnvVar.Name, portEnvVar.Name, newExpression)

	if err := file.ReplaceNodeContent(redisR.expression, replaceContent); err != nil {
		return nil, err
	}

	result := &core.Persist{
		Kind: kind,
		Name: cap.Capability.ID,
	}

	unit.EnvironmentVariables = append(unit.EnvironmentVariables, hostEnvVar)
	unit.EnvironmentVariables = append(unit.EnvironmentVariables, portEnvVar)

	return result, nil
}

type OrmKind string

const (
	SequelizeKind OrmKind = "sequelize"
	TypeOrmKind   OrmKind = "typeorm"
)

type persistResult struct {
	expression *sitter.Node
	name       string
	ormType    OrmKind
}

func (p *persister) queryKV(file *core.SourceFile, annotation *core.Annotation, enableWarnings bool) *persistResult {
	log := zap.L().With(logging.FileField(file), logging.AnnotationField(annotation))

	nextMatch := DoQuery(annotation.Node, persistKV)

	for {
		match, found := nextMatch()
		if !found {
			return nil
		}

		name, constructor, object, expression := match["name"], match["constructor"], match["object"], match["expression"]

		if !query.NodeContentEquals(constructor, "Map") {
			continue
		}

		if object != nil && !query.NodeContentEquals(object, "exports") {
			if enableWarnings {
				log.Warn("expected object of assignment to be 'exports'")
			}
			return nil
		}

		if _, found := nextMatch(); found {
			if enableWarnings {
				log.Warn("too many assignments for kv_storage")
			}
			return nil
		}
		return &persistResult{
			name:       name.Content(),
			expression: expression,
		}
	}
}

func (p *persister) queryFS(file *core.SourceFile, annotation *core.Annotation) *persistResult {
	imports := FindNextImportStatement(annotation.Node)
	if len(imports) == 0 {
		return nil
	}

	var fsImport Import
	for _, imp := range imports {
		if predicate.AnyOf(
			predicate.AllOf(
				IsImportOfType(ImportTypeNamed),
				IsImportOfModule("fs"),
				ImportHasName("promises"),
			),
			predicate.AllOf(
				predicate.AnyOf(IsImportOfType(ImportTypeNamespace), IsImportOfType(ImportTypeDefault)),
				IsImportOfModule("fs/promises"),
			),
		)(imp) {
			fsImport = imp
		}
	}

	if fsImport == (Import{}) {
		return nil
	}

	return &persistResult{
		name:       fsImport.ImportedAs(),
		expression: fsImport.SourceNode,
	}
}

func (p *persister) querySecretName(file *core.SourceFile, fsName string) ([]string, error) {

	// use the file tree root node since we need to check all instances of secret persist readFile
	nextMatch := DoQuery(file.Tree().RootNode(), persistSecret)

	secrets := make([]string, 0)

	for {
		match, found := nextMatch()
		if !found {
			break
		}

		secretName, object, property := match["secretName"], match["object"], match["property"]
		if object != nil && property != nil && query.NodeContentEquals(object, fsName) {
			if query.NodeContentEquals(property, "readFile") {
				if secretName != nil {
					sn := StringLiteralContent(secretName)
					secrets = append(secrets, sn)
				} else {
					return nil, errors.New("must supply static string for secret path")
				}
			} else {
				return nil, errors.Errorf("'%s' not implemented for secrets persist.", property)
			}
		}
	}
	return secrets, nil
}

func (p *persister) queryORM(file *core.SourceFile, annotation *core.Annotation, enableWarnings bool) *persistResult {
	nextMatch := DoQuery(annotation.Node, persistORM)

	match, found := nextMatch()
	if !found {
		return nil
	}

	name, argstring := match["name"], match["argstring"]

	ormtype := match["type"].Content()
	var ormKind OrmKind
	switch ormtype {
	case "Sequelize":
		ormKind = SequelizeKind
	case "DataSource":
		ormKind = TypeOrmKind
	default:
		return nil
	}
	if obj := match["var.obj"]; obj != nil {
		if !query.NodeContentEquals(obj, "exports") {
			return nil
		}
	}

	return &persistResult{
		name:       name.Content(),
		expression: argstring,
		ormType:    ormKind,
	}
}

func (p *persister) queryRedis(file *core.SourceFile, annotation *core.Annotation, enableWarnings bool) (core.PersistKind, *persistResult) {
	nextMatch := DoQuery(annotation.Node, persistRedis)

	match, found := nextMatch()
	if !found {
		return "", nil
	}

	name, argstring, method := match["name"], match["argstring"], match["method"]

	kind := core.PersistRedisNodeKind
	if method.Content() == "createCluster" {
		kind = core.PersistRedisClusterKind
	}

	if method.Content() != "createClient" && method.Content() != "createCluster" {
		return "", nil
	}

	if obj := match["var.obj"]; obj != nil {
		if !query.NodeContentEquals(obj, "exports") {
			return "", nil
		}
	}

	return kind, &persistResult{
		name:       name.Content(),
		expression: argstring,
	}
}

func (p *persister) determinePersistType(f *core.SourceFile, annotation *core.Annotation) (core.PersistKind, *persistResult) {
	log := zap.L().With(logging.FileField(f), logging.AnnotationField(annotation))

	kvR := p.queryKV(f, annotation, false)
	if kvR != nil {
		log.Sugar().Debugf("Determined persist type of '%s'", core.PersistKVKind)
		return core.PersistKVKind, kvR
	}

	// We only check for FS and not Secrets because they are defined in the same way.
	// It's not possible to know which is intended, so defaulting to FS
	fsR := p.queryFS(f, annotation)
	if fsR != nil {
		secret, ok := annotation.Capability.Directives.Bool("secret")
		if ok && secret {
			log.Sugar().Debugf("Determined persist type of '%s'", core.PersistSecretKind)
			return core.PersistSecretKind, fsR
		}
		log.Sugar().Debugf("Determined persist type of '%s'", core.PersistFileKind)
		return core.PersistFileKind, fsR
	}

	ormR := p.queryORM(f, annotation, false)
	if ormR != nil {
		log.Sugar().Debugf("Determined persist type of '%s'", core.PersistORMKind)
		return core.PersistORMKind, ormR
	}

	redisKind, redis := p.queryRedis(f, annotation, false)
	if redis != nil {
		log.Sugar().Debugf("Determined persist type of '%s'", redisKind)
		return redisKind, redis
	}

	return "", nil
}
