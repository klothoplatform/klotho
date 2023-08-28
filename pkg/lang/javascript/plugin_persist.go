package javascript

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/klothoplatform/klotho/pkg/compiler/types"
	klotho_errors "github.com/klothoplatform/klotho/pkg/errors"
	"github.com/klothoplatform/klotho/pkg/sanitization"

	"github.com/klothoplatform/klotho/pkg/filter/predicate"

	"github.com/klothoplatform/klotho/pkg/multierr"

	"github.com/klothoplatform/klotho/pkg/annotation"
	"github.com/klothoplatform/klotho/pkg/construct"
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

func (p Persist) Transform(input *types.InputFiles, fileDeps *types.FileDependencies, constructGraph *construct.ConstructGraph) error {
	persister := &persister{ConstructGraph: constructGraph, runtime: p.runtime}
	var errs multierr.Error

	// It's important for this to happen first, before the code gets transformed; otherwise we won't find the "new Map()"s.
	// Please be careful before moving this loop, or putting anything before it. The call to findUnawaitedCalls(units)
	// assumes that the code has not yet been rewritten (e.g. to turn Maps into our runtime map classes).
	// This code could use a cleanup: see CloudCompilers/klotho#431
	for _, unit := range construct.GetConstructsOfType[*types.ExecutionUnit](constructGraph) {
		persister.findUnawaitedCalls(unit)

		err := persister.handleFiles(unit)
		if err != nil {
			errs.Append(err)
			continue
		}
	}

	return errs.ErrOrNil()
}

func (p *persister) hasKvAnnotation(declaringFile *types.SourceFile, annot *types.Annotation) bool {
	if annot.Capability.Name != annotation.PersistCapability {
		return false
	}
	pType, _ := p.determinePersistType(declaringFile, annot)
	_, ok := pType.(*types.Kv)
	return ok
}

func (p *persister) findUnawaitedCalls(unit *types.ExecutionUnit) {
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

func (p *persister) findUnwaitedCallsInFile(js *types.SourceFile, spec VarSpec) (errs []*sitter.Node) {
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
	ConstructGraph *construct.ConstructGraph
	runtime        Runtime
}

func (p *persister) handleFiles(unit *types.ExecutionUnit) error {
	var errs multierr.Error
	for _, f := range unit.Files() {
		js, ok := Language.ID.CastFile(f)
		if !ok {
			continue
		}

		constructs, err := p.handleFile(js, unit)
		if err != nil {
			errs.Append(klotho_errors.WrapErrf(err, "failed to handle persist in unit %s", unit.Name))
		}

		for _, c := range constructs {
			p.ConstructGraph.AddConstruct(c)

			_, isReferencedByExecUnit := unit.Executable.SourceFiles[js.Path()]

			// a file containing capabilities without an execution unit indicates that the file's capabilities
			// are imported by execution units in one or more separate files
			if types.FileExecUnitName(js) != "" || isReferencedByExecUnit {
				p.ConstructGraph.AddDependency(unit.Id(), c.Id())
			}
		}
	}

	return errs.ErrOrNil()
}

func (p *persister) handleFile(f *types.SourceFile, unit *types.ExecutionUnit) ([]construct.Construct, error) {
	annots := f.Annotations()
	var resources []construct.Construct

	var errs multierr.Error
	for _, annot := range annots {
		cap := annot.Capability
		if cap.Name != annotation.PersistCapability {
			continue
		}

		if annot.Capability.Directives.Object(types.EnvironmentVariableDirective) != nil {
			// This is handled by envvar.EnvVarInjection
			continue
		}

		keyType, pResult := p.determinePersistType(f, annot)

		if len(cap.ID) == 0 {
			errs.Append(types.NewCompilerError(f, annot, errors.New("'id' is required")))
		}

		var construct construct.Construct
		var err, runtimeErr, transformErr error
		switch keyType.(type) {
		case *types.Kv:
			construct, transformErr = p.transformKV(unit, f, annot, pResult)
			runtimeErr = p.runtime.AddKvRuntimeFiles(unit)
		case *types.Fs:
			var envVarName string
			construct, envVarName, transformErr = p.transformFS(unit, f, annot, pResult)
			runtimeErr = p.runtime.AddFsRuntimeFiles(unit, envVarName, cap.ID)
		case *types.Secrets:
			construct, transformErr = p.transformSecret(f, annot, pResult)
			runtimeErr = p.runtime.AddSecretRuntimeFiles(unit)
		case *types.Orm:
			construct, transformErr = p.transformORM(unit, f, annot, pResult)
			runtimeErr = p.runtime.AddOrmRuntimeFiles(unit)
		case *types.RedisCluster:
			construct, transformErr = p.transformRedis(unit, f, annot, pResult, keyType)
			runtimeErr = p.runtime.AddRedisClusterRuntimeFiles(unit)
		case *types.RedisNode:
			construct, transformErr = p.transformRedis(unit, f, annot, pResult, keyType)
			runtimeErr = p.runtime.AddRedisNodeRuntimeFiles(unit)
		default:
			err = fmt.Errorf("type '%s' is invalid for the persist capability", keyType)
		}
		if err != nil {
			errs.Append(types.NewCompilerError(f, annot, err))
			continue
		}
		if transformErr != nil || runtimeErr != nil {
			if transformErr != nil {
				errs.Append(types.NewCompilerError(f, annot, transformErr))
			}
			if runtimeErr != nil {
				errs.Append(types.NewCompilerError(f, annot, runtimeErr))
			}
			continue
		}

		// Do this after the specific transforms so that `pResult` nodes aren't invalidated
		if err := p.runtime.TransformPersist(f, annot, keyType); err != nil {
			return nil, err
		}
		resources = append(resources, construct)
	}

	return resources, errs.ErrOrNil()
}

func (p *persister) transformSecret(file *types.SourceFile, cap *types.Annotation, secretR *persistResult) (construct.Construct, error) {
	if err := file.ReplaceNodeContent(secretR.expression, "secretRuntime"); err != nil {
		return nil, err
	}

	// get secret file name
	secrets, err := p.querySecretName(file, secretR.name)
	if err != nil {
		return nil, err
	}

	result := &types.Secrets{
		Name:    cap.Capability.Name,
		Secrets: secrets,
	}

	return result, nil
}

func (p *persister) transformFS(unit *types.ExecutionUnit, file *types.SourceFile, cap *types.Annotation, fsR *persistResult) (construct.Construct, string, error) {
	if err := file.ReplaceNodeContent(fsR.expression, sanitization.IdentifierSanitizer.Apply(fmt.Sprintf("fs_%sRuntime", cap.Capability.ID))+".fs"); err != nil {
		return nil, "", errors.Wrap(err, "could not reparse FS transformation")
	}

	result := &types.Fs{
		Name: cap.Capability.ID,
	}

	fsEnvVar := types.GenerateBucketEnvVar(result)

	unit.EnvironmentVariables.Add(fsEnvVar)

	return result, fsEnvVar.Name, nil
}

func (p *persister) transformKV(unit *types.ExecutionUnit, file *types.SourceFile, cap *types.Annotation, kvR *persistResult) (construct.Construct, error) {
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

	result := &types.Kv{
		Name: cap.Capability.ID,
	}

	envVar := types.GenerateKvTableNameEnvVar(result)
	unit.EnvironmentVariables.Add(envVar)

	return result, nil
}

func (p *persister) transformORM(unit *types.ExecutionUnit, file *types.SourceFile, cap *types.Annotation, ormR *persistResult) (construct.Construct, error) {
	result := &types.Orm{
		Name: cap.Capability.ID,
	}
	envVar := types.GenerateOrmConnStringEnvVar(result)

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

	unit.EnvironmentVariables.Add(envVar)

	return result, nil
}

func (p *persister) transformRedis(unit *types.ExecutionUnit, file *types.SourceFile, cap *types.Annotation, redisR *persistResult, c construct.Construct) (construct.Construct, error) {
	// Because the redis client can be initialized with () or ({...}) we have to have the expression match it all.
	// We need to remove the outer () so that the runtime will process these correctly.
	newExpression := strings.TrimLeft(redisR.expression.Content(), "(")
	newExpression = strings.TrimRight(newExpression, ")")

	if newExpression == "" {
		newExpression = "{}"
	}

	var importName string
	var result construct.Construct

	switch c.(type) {
	case *types.RedisCluster:
		result = &types.RedisCluster{
			Name: cap.Capability.ID,
		}
		importName = "redis_cluster"
	case *types.RedisNode:
		result = &types.RedisNode{
			Name: cap.Capability.ID,
		}
		importName = "redis_node"
	}

	hostEnvVar := types.GenerateRedisHostEnvVar(result)
	portEnvVar := types.GenerateRedisPortEnvVar(result)

	replaceContent := fmt.Sprintf(`(%sRuntime.getParams("%s", "%s", %s))`, importName, hostEnvVar.Name, portEnvVar.Name, newExpression)

	if err := file.ReplaceNodeContent(redisR.expression, replaceContent); err != nil {
		return nil, err
	}

	unit.EnvironmentVariables.Add(hostEnvVar)
	unit.EnvironmentVariables.Add(portEnvVar)

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

func (p *persister) queryKV(file *types.SourceFile, annotation *types.Annotation, enableWarnings bool) *persistResult {
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

func (p *persister) queryFS(file *types.SourceFile, annotation *types.Annotation) *persistResult {
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

func (p *persister) querySecretName(file *types.SourceFile, fsName string) ([]string, error) {

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

func (p *persister) queryORM(file *types.SourceFile, annotation *types.Annotation, enableWarnings bool) *persistResult {
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

func (p *persister) queryRedis(file *types.SourceFile, annotation *types.Annotation, enableWarnings bool) (construct.Construct, *persistResult) {
	nextMatch := DoQuery(annotation.Node, persistRedis)

	match, found := nextMatch()
	if !found {
		return nil, nil
	}

	name, argstring, method := match["name"], match["argstring"], match["method"]

	var kind construct.Construct
	if method.Content() == "createCluster" {
		kind = &types.RedisCluster{}
	} else {
		kind = &types.RedisNode{}
	}

	if method.Content() != "createClient" && method.Content() != "createCluster" {
		return nil, nil
	}

	if obj := match["var.obj"]; obj != nil {
		if !query.NodeContentEquals(obj, "exports") {
			return nil, nil
		}
	}

	return kind, &persistResult{
		name:       name.Content(),
		expression: argstring,
	}
}

func (p *persister) determinePersistType(f *types.SourceFile, annotation *types.Annotation) (construct.Construct, *persistResult) {
	log := zap.L().With(logging.FileField(f), logging.AnnotationField(annotation))

	kvR := p.queryKV(f, annotation, false)
	if kvR != nil {
		log.Sugar().Debugf("Determined persist type of kv")
		return &types.Kv{}, kvR
	}

	// We only check for FS and not Secrets because they are defined in the same way.
	// It's not possible to know which is intended, so defaulting to FS
	fsR := p.queryFS(f, annotation)
	if fsR != nil {
		secret, ok := annotation.Capability.Directives.Bool("secret")
		if ok && secret {
			log.Sugar().Debugf("Determined persist type of secrets")
			return &types.Secrets{}, fsR
		}
		log.Sugar().Debugf("Determined persist type of fs")
		return &types.Fs{}, fsR
	}

	ormR := p.queryORM(f, annotation, false)
	if ormR != nil {
		log.Sugar().Debugf("Determined persist type of orm")
		return &types.Orm{}, ormR
	}

	redisKind, redis := p.queryRedis(f, annotation, false)
	if redis != nil {
		log.Sugar().Debugf("Determined persist type of redis: '%T'", redisKind)
		return redisKind, redis
	}

	return nil, nil
}
