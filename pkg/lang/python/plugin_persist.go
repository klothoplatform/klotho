package python

import (
	"fmt"
	"strings"

	"github.com/klothoplatform/klotho/pkg/multierr"

	"github.com/klothoplatform/klotho/pkg/annotation"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/logging"
	"github.com/klothoplatform/klotho/pkg/query"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

type Persist struct {
	runtime Runtime
}

func (p Persist) Name() string { return "Persist" }

func (p Persist) Transform(result *core.CompilationResult, deps *core.Dependencies) error {
	persister := &persister{result: result, deps: deps, runtime: p.runtime}

	var errs multierr.Error
	for _, res := range result.Resources() {
		unit, ok := res.(*core.ExecutionUnit)
		if !ok {
			continue
		}

		err := persister.handleFiles(unit)
		if err != nil {
			errs.Append(err)
			continue
		}
	}

	return errs.ErrOrNil()
}

type persister struct {
	result  *core.CompilationResult
	deps    *core.Dependencies
	runtime Runtime
}

func (p *persister) handleFiles(unit *core.ExecutionUnit) error {
	var errs multierr.Error
	for _, f := range unit.Files() {
		pySource, ok := Language.ID.CastFile(f)
		if !ok {
			continue
		}

		resources, err := p.handleFile(pySource, unit)
		if err != nil {
			errs.Append(core.WrapErrf(err, "failed to handle persist in unit %s", unit.Name))
		}

		for _, r := range resources {
			p.result.Add(r)

			// a file containing capabilities without an execution unit indicates that the file's capabilities
			// are imported by execution units in one or more separate files
			if core.FileExecUnitName(pySource) != "" || p.isFileReferencedByExecUnit(pySource, unit) {
				p.deps.Add(core.ResourceKey{
					Name: unit.Name,
					Kind: core.ExecutionUnitKind,
				}, r.Key())
			}
		}
	}

	return errs.ErrOrNil()
}

// isFileReferencedByExecUnit determines if the supplied resource, declared in file 'pySource',
// is imported by the supplied exec unit or if referenced from a gateway
// that exposes this exec unit as a direct dependency
func (p *persister) isFileReferencedByExecUnit(pySource *core.SourceFile, unit *core.ExecutionUnit) bool {
	// TODO: implement reference detection when implementing multi-exec_unit for python
	return true
}

func (p *persister) handleFile(f *core.SourceFile, unit *core.ExecutionUnit) ([]core.CloudResource, error) {
	annots := f.Annotations()
	newFile := f.CloneSourceFile()

	var resources []core.CloudResource

	var errs multierr.Error
	for _, annot := range annots {
		log := zap.L().With(
			logging.AnnotationField(annot),
			logging.FileField(f),
		)
		cap := annot.Capability
		if cap.Name != annotation.PersistCapability {
			continue
		}

		keyType, pResult := p.determinePersistType(f, annot)
		if pResult == nil {
			if annot.Capability.Directives.Object(core.EnvironmentVariableDirective) != nil {
				continue
			}
			log.Warn("Could not determine persist type")
			continue
		}

		if len(cap.ID) == 0 {
			errs.Append(core.NewCompilerError(f, annot, errors.New("'id' is required")))
		}

		var doTransform func(original *core.SourceFile, modified *core.SourceFile, cap *core.Annotation, result *persistResult, unit *core.ExecutionUnit) (core.CloudResource, error)
		var err error
		switch keyType {
		case core.PersistKVKind:
			doTransform = p.transformKV
			err = p.runtime.AddKvRuntimeFiles(unit)
		case core.PersistFileKind:
			doTransform = p.transformFS
			err = p.runtime.AddFsRuntimeFiles(unit)
		case core.PersistSecretKind:
			doTransform = p.transformSecret
			err = p.runtime.AddSecretRuntimeFiles(unit)
		case core.PersistORMKind:
			doTransform = p.transformORM
			err = p.runtime.AddOrmRuntimeFiles(unit)
		case core.PersistRedisClusterKind:
			doTransform = p.transformRedis
		case core.PersistRedisNodeKind:
			doTransform = p.transformRedis
		default:
			errs.Append(core.NewCompilerError(
				f,
				annot,
				fmt.Errorf("type '%s' is invalid for the persist capability", keyType),
			))
			continue
		}
		errs.Append(err)

		resource, err := doTransform(f, newFile, annot, pResult, unit)
		if err != nil {
			errs.Append(err)
		} else {
			resources = append(resources, resource)
		}
	}

	err := f.Reparse(newFile.Program())
	errs.Append(err)

	return resources, errs.ErrOrNil()
}

func (p *persister) transformKV(original *core.SourceFile, modified *core.SourceFile, cap *core.Annotation, kvR *persistResult, unit *core.ExecutionUnit) (core.CloudResource, error) {

	// add the kv runtime import to the file containing a persisted aiocache instance
	kvConfig := p.runtime.GetKvRuntimeConfig()
	err := AddRuntimeImport(kvConfig.Imports, modified)
	if err != nil {
		return nil, errors.Wrap(err, "could not reparse KV transformation")
	}

	// replace the aiocache.Cache() invocation's arguments with those required for the runtime
	nodeContent := cap.Node.Content(original.Program())
	directives := cap.Capability.Directives
	id, found := directives.String("id")
	if !found {
		return nil, errors.New("'id' directive not found")
	}

	cacheClassArg := kvConfig.CacheClassArg
	args := kvR.args
	if len(args) > 0 && args[0].Name == "" {
		args[0] = cacheClassArg
	} else {
		args = AddOrReplaceArg(cacheClassArg, args)
	}

	for _, arg := range kvConfig.AdditionalCacheConstructorArgs {
		args = AddOrReplaceArg(arg, args)
	}

	args = AddOrReplaceArg(FunctionArg{
		Name:  "table_name",
		Value: fmt.Sprintf(`"%s"`, p.runtime.GetAppName()),
	}, args)
	args = AddOrReplaceArg(FunctionArg{
		Name:  "map_id",
		Value: fmt.Sprintf(`"%s"`, id),
	}, args)

	var argStrings []string
	for _, arg := range args {
		argStrings = append(argStrings, arg.String())
	}

	argsList := strings.Join(argStrings, ", ")

	runtimeExpr := strings.SplitN(kvR.expression, "(", 2)[0] + "(" + argsList + ")"

	expression := strings.Replace(nodeContent, kvR.expression, runtimeExpr, -1)

	modifiedSrc := string(modified.Program())

	// replace original expression with new expression (uses string slicing over strings.replaceAll to minimize unintended consequences)
	for _, mCap := range modified.Annotations() {
		if cap.Capability.Name == mCap.Capability.Name && cap.Capability.ID == mCap.Capability.ID {
			startByte := mCap.Node.StartByte()
			endByte := mCap.Node.EndByte()
			modifiedSrc = modifiedSrc[0:startByte] + expression + modifiedSrc[endByte:]
		}
	}

	err = modified.Reparse([]byte(modifiedSrc))
	if err != nil {
		return nil, errors.Wrap(err, "could not reparse KV transformation")
	}

	result := &core.Persist{
		Kind: core.PersistKVKind,
		Name: cap.Capability.ID,
	}

	return result, nil
}

func (p *persister) transformFS(original *core.SourceFile, modified *core.SourceFile, cap *core.Annotation, fsR *persistResult, unit *core.ExecutionUnit) (core.CloudResource, error) {

	nodeContent := cap.Node.Content(original.Program())

	replaceString := p.runtime.GetFsRuntimeImportClass(fsR.name)

	newContent := nodeContent
	newExpression := strings.Replace(newContent, fsR.expression, replaceString, -1)
	modifiedSrc := string(modified.Program())

	// replace original expression with new expression (uses string slicing over strings.replaceAll to minimize unintended consequences)
	for _, mCap := range modified.Annotations() {
		if cap.Capability.Name == mCap.Capability.Name && cap.Capability.ID == mCap.Capability.ID {
			startByte := mCap.Node.StartByte()
			endByte := mCap.Node.EndByte()
			modifiedSrc = modifiedSrc[:startByte] + newExpression + modifiedSrc[endByte:]
		}
	}
	err := modified.Reparse([]byte(modifiedSrc))
	if err != nil {
		return nil, errors.Wrap(err, "could not reparse FS transformation")
	}

	result := &core.Persist{
		Kind: core.PersistFileKind,
		Name: cap.Capability.ID,
	}

	return result, nil
}

func (p *persister) transformSecret(original *core.SourceFile, modified *core.SourceFile, cap *core.Annotation, secretR *persistResult, unit *core.ExecutionUnit) (core.CloudResource, error) {

	nodeContent := cap.Node.Content(original.Program())

	replaceString := p.runtime.GetSecretRuntimeImportClass(secretR.name)

	newContent := nodeContent
	newExpression := strings.Replace(newContent, secretR.expression, replaceString, -1)
	modifiedSrc := string(modified.Program())

	// replace original expression with new expression (uses string slicing over strings.replaceAll to minimize unintended consequences)
	for _, mCap := range modified.Annotations() {
		if cap.Capability.Name == mCap.Capability.Name && cap.Capability.ID == mCap.Capability.ID {
			startByte := mCap.Node.StartByte()
			endByte := mCap.Node.EndByte()
			modifiedSrc = modifiedSrc[:startByte] + newExpression + modifiedSrc[endByte:]
		}
	}
	err := modified.Reparse([]byte(modifiedSrc))
	if err != nil {
		return nil, errors.Wrap(err, "could not reparse Secrets transformation")
	}
	// get secret file name
	secrets, err := p.querySecret(original, secretR.name)
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

func (p *persister) transformORM(original *core.SourceFile, modified *core.SourceFile, cap *core.Annotation, ormR *persistResult, unit *core.ExecutionUnit) (core.CloudResource, error) {

	nodeContent := cap.Node.Content(original.Program())

	newContent := nodeContent
	err := AddRuntimeImport("import os", modified)
	if err != nil {
		return nil, errors.Wrap(err, "could not reparse ORM transformation")
	}
	envVar := core.GenerateOrmConnStringEnvVar(cap.Capability.ID, string(ormR.kind))

	replaceContent := fmt.Sprintf(`os.environ.get("%s")`, envVar.Name)

	expression := strings.Replace(newContent, ormR.expression, replaceContent, -1)

	modifiedSrc := string(modified.Program())
	// replace original expression with new expression (uses string slicing over strings.replaceAll to minimize unintended consequences)
	for _, mCap := range modified.Annotations() {
		if cap.Capability.Name == mCap.Capability.Name && cap.Capability.ID == mCap.Capability.ID {
			startByte := mCap.Node.StartByte()
			endByte := mCap.Node.EndByte()
			modifiedSrc = modifiedSrc[:startByte] + expression + modifiedSrc[endByte:]
		}
	}
	err = modified.Reparse([]byte(modifiedSrc))
	if err != nil {
		return nil, errors.Wrap(err, "could not reparse ORM transformation")
	}

	result := &core.Persist{
		Kind: core.PersistORMKind,
		Name: cap.Capability.ID,
	}
	unit.EnvironmentVariables = append(unit.EnvironmentVariables, envVar)

	return result, nil
}

func (p *persister) transformRedis(original *core.SourceFile, modified *core.SourceFile, cap *core.Annotation, redisR *persistResult, unit *core.ExecutionUnit) (core.CloudResource, error) {

	nodeContent := cap.Node.Content(original.Program())

	err := AddRuntimeImport("import os", modified)
	if err != nil {
		return nil, errors.Wrap(err, "could not reparse Redis transformation")
	}

	newContent := nodeContent

	hostEnvVar := core.GenerateRedisHostEnvVar(cap.Capability.ID, string(redisR.kind))
	portEnvVar := core.GenerateRedisPortEnvVar(cap.Capability.ID, string(redisR.kind))

	args := redisR.args
	args = AddOrReplaceArg(FunctionArg{
		Name:  "host",
		Value: fmt.Sprintf(`os.environ.get("%s")`, hostEnvVar.Name),
	}, args)
	args = AddOrReplaceArg(FunctionArg{
		Name:  "port",
		Value: fmt.Sprintf(`os.environ.get("%s")`, portEnvVar.Name),
	}, args)
	if redisR.kind == core.PersistRedisClusterKind {
		args = AddOrReplaceArg(FunctionArg{
			Name:  "ssl",
			Value: "True",
		}, args)
		args = AddOrReplaceArg(FunctionArg{
			Name:  "skip_full_coverage_check",
			Value: "True",
		}, args)
	}

	var argStrings []string
	for _, arg := range args {
		argStrings = append(argStrings, arg.String())
	}

	argsList := strings.Join(argStrings, ", ")

	replaceContent := fmt.Sprintf(`(%s)`, argsList)

	expression := strings.Replace(newContent, redisR.expression, replaceContent, -1)

	modifiedSrc := string(modified.Program())
	// replace original expression with new expression (uses string slicing over strings.replaceAll to minimize unintended consequences)
	for _, mCap := range modified.Annotations() {
		if cap.Capability.Name == mCap.Capability.Name && cap.Capability.ID == mCap.Capability.ID {
			startByte := mCap.Node.StartByte()
			endByte := mCap.Node.EndByte()
			modifiedSrc = modifiedSrc[:startByte] + expression + modifiedSrc[endByte:]
		}
	}
	err = modified.Reparse([]byte(modifiedSrc))
	if err != nil {
		return nil, errors.Wrap(err, "could not reparse Redis transformation")
	}

	result := &core.Persist{
		Kind: redisR.kind,
		Name: cap.Capability.ID,
	}

	unit.EnvironmentVariables = append(unit.EnvironmentVariables, hostEnvVar)
	unit.EnvironmentVariables = append(unit.EnvironmentVariables, portEnvVar)

	return result, nil
}

type persistResult struct {
	expression string
	name       string
	args       []FunctionArg
	kind       core.PersistKind
}

func (p *persister) queryKV(file *core.SourceFile, annotation *core.Annotation, enableWarnings bool) *persistResult {
	log := zap.L().With(logging.FileField(file), logging.AnnotationField(annotation))

	imports := FindImports(file)

	aiocacheImport, ok := imports["aiocache"]
	if !ok {
		return nil
	}
	aiocacheImported := aiocacheImport.ImportedSelf
	cacheImport, cacheImported := aiocacheImport.ImportedAttributes["Cache"]
	functionHostName := aiocacheImport.Name
	cacheFunction := cacheImport.Name

	nextMatch := DoQuery(annotation.Node, persistKV)

	match, found := nextMatch()
	if !found {
		return nil
	}

	expression, name, functionHost, function := match["expression"], match["name"], match["functionHost"], match["function"]

	// this assignment/invocation is unrelated to aiocache.Cache instantiation
	if !aiocacheImported && !query.NodeContentEquals(function, file.Program(), cacheFunction) {
		return nil
	}

	// this Cache() invocation belongs to an object other the aiocache module
	if aiocacheImported && functionHost != nil && !query.NodeContentEquals(functionHost, file.Program(), functionHostName) {
		return nil
	}

	// this Cache() invocation is unrelated to aiocache
	if !aiocacheImported && !cacheImported {
		return nil
	}

	callDetails, found := getNextCallDetails(parentOfType(function, "call"), file.Program())
	if !found {
		if enableWarnings {
			log.Warn("function call details not found")
		}
		return nil
	}
	args := callDetails.Arguments

	if _, found := nextMatch(); found {
		if enableWarnings {
			log.Warn("too many assignments for kv_storage")
		}
		return nil
	}

	return &persistResult{
		name:       name.Content(file.Program()),
		expression: expression.Content(file.Program()),
		args:       args,
	}
}

func (p *persister) queryFS(file *core.SourceFile, annotation *core.Annotation, enableWarnings bool) *persistResult {
	log := zap.L().With(logging.FileField(file), logging.AnnotationField(annotation))

	imports := FindImports(file)

	fsSpecImport, ok := imports["aiofiles"]
	if !ok {
		return nil
	}

	varName := ""
	if fsSpecImport.Alias != "" {
		varName = fsSpecImport.Alias
	} else if fsSpecImport.ImportedSelf {
		varName = fsSpecImport.Name
	} else {
		return nil
	}

	nextMatch := DoQuery(annotation.Node, findImports)

	match, found := nextMatch()
	if !found {
		return nil
	}

	module, aliasedModule, alias, importStatement := match["module"], match["aliasedModule"], match["alias"], match["importStatement"]

	// this assignment/invocation is unrelated to aiofile instantiation found from the matching import
	if aliasedModule != nil {
		if !query.NodeContentEquals(alias, file.Program(), varName) {
			return nil
		}
	} else if !query.NodeContentEquals(module, file.Program(), varName) {
		return nil
	}

	if _, found := nextMatch(); found {
		if enableWarnings {
			log.Warn("too many assignments for fs_storage")
		}
		return nil
	}

	return &persistResult{
		name:       varName,
		expression: importStatement.Content(file.Program()),
	}
}

func (p *persister) queryORM(file *core.SourceFile, annotation *core.Annotation, enableWarnings bool) *persistResult {
	log := zap.L().With(logging.FileField(file), logging.AnnotationField(annotation))

	imports := FindImports(file)

	sqlalchemyImport, ok := imports["sqlalchemy"]
	if !ok {
		return nil
	}
	sqlalchemyImported := sqlalchemyImport.ImportedSelf
	sqlalchemyImportName := sqlalchemyImport.Name
	engineImport, engineImported := sqlalchemyImport.ImportedAttributes["create_engine"]
	engineFunction := engineImport.Name
	if engineImport.Alias != "" {
		engineFunction = engineImport.Alias
	}
	if sqlalchemyImport.Alias != "" {
		sqlalchemyImportName = sqlalchemyImport.Alias
	}

	nextMatch := DoQuery(annotation.Node, orm)

	match, found := nextMatch()
	if !found {
		return nil
	}

	engineVar, funcCall, connString, module := match["engineVar"], match["funcCall"], match["connString"], match["module"]

	// this assignment/invocation is unrelated to sqlAlchemy.create_engine instantiation
	if !sqlalchemyImported && !query.NodeContentEquals(funcCall, file.Program(), engineFunction) {
		return nil
	}

	// this create_engine() invocation belongs to an object other the aiocache module
	if sqlalchemyImported && module != nil && !query.NodeContentEquals(module, file.Program(), sqlalchemyImportName) {
		return nil
	}

	// this create_engine() invocation is unrelated to sqlAlchemy
	if !sqlalchemyImported && !engineImported {
		return nil
	}

	if _, found := nextMatch(); found {
		if enableWarnings {
			log.Warn("too many assignments for persist_orm")
		}
		return nil
	}

	return &persistResult{
		name:       engineVar.Content(file.Program()),
		expression: connString.Content(file.Program()),
		kind:       core.PersistORMKind,
	}
}

func (p *persister) querySecret(file *core.SourceFile, name string) ([]string, error) {
	// use the file tree root node since we need to check all instances of secret persist readFile
	nextMatch := DoQuery(file.Tree().RootNode(), aiofilesOpen)

	secrets := make([]string, 0)

	for {
		match, found := nextMatch()
		if !found {
			break
		}
		module, moduleMethod, varOut, varIn, funcCall, path := match["module"], match["moduleMethod"], match["varOut"], match["varIn"], match["func"], match["path"]

		if !query.NodeContentEquals(module, file.Program(), name) {
			continue
		}

		if !query.NodeContentEquals(moduleMethod, file.Program(), "open") {
			continue
		}

		if varIn.Content(file.Program()) != varOut.Content(file.Program()) {
			continue
		}

		if query.NodeContentEquals(funcCall, file.Program(), "read") {
			if path != nil {
				sn, err := stringLiteralContent(path, file.Program())
				if err != nil {
					return nil, errors.Errorf("'%s' unable to get path from.", path.Content(file.Program()))
				}
				secrets = append(secrets, sn)
			} else {
				return nil, errors.New("must supply static string for secret path")
			}
		} else {
			return nil, errors.Errorf("'%s' not implemented for secrets persist.", funcCall.Content(file.Program()))
		}
	}
	return secrets, nil

}

func (p *persister) queryRedis(file *core.SourceFile, annotation *core.Annotation, enableWarnings bool) *persistResult {
	log := zap.L().With(logging.FileField(file), logging.AnnotationField(annotation))

	imports := FindImports(file)

	redisImport, ok := imports["redis"]
	redisClusterImport, cok := imports["redis.cluster"]
	if !ok && !cok {
		return nil
	}
	redisImported := redisImport.ImportedSelf
	redisImportName := redisImport.Name
	constructorImport, constructorImported := redisImport.ImportedAttributes["Redis"]
	clusterConstructorImport, clusterConstructorImported := redisClusterImport.ImportedAttributes["RedisCluster"]
	clustermoduleImport, clusterModuleImported := redisImport.ImportedAttributes["cluster"]
	clusterRedisFunction := clusterConstructorImport.Name
	clustermoduleImportName := clustermoduleImport.Name

	redisFunction := constructorImport.Name
	if redisFunction == "" {
		redisFunction = "Redis"
	} else if constructorImport.Alias != "" {
		redisFunction = constructorImport.Alias
	}
	if clusterRedisFunction == "" {
		clusterRedisFunction = "RedisCluster"
	} else if clusterConstructorImport.Alias != "" {
		clusterRedisFunction = clusterConstructorImport.Alias
	}
	if redisImport.Alias != "" {
		redisImportName = redisImport.Alias
	}
	if clustermoduleImport.Alias != "" {
		clustermoduleImportName = clustermoduleImport.Alias
	}

	nextMatch := DoQuery(annotation.Node, redis)

	match, found := nextMatch()
	if !found {
		return nil
	}

	redisVar, funcCall, args, module, subModule := match["redisVar"], match["funcCall"], match["args"], match["module"], match["subModule"]

	// this Redis() or RedisCluster() invocation belongs to an object other the redis module
	if redisImported && !clusterModuleImported && module != nil && !query.NodeContentEquals(module, file.Program(), redisImportName) {
		return nil
	}

	// import is similar to `from redis import cluster` and the RedisCluster call does not use cluster module
	if clusterModuleImported && !query.NodeContentEquals(module, file.Program(), clustermoduleImportName) {
		return nil
	}

	// Redis is not self imported and the function call does not match the redis or redis cluster function call from the import
	if !redisImported && (!query.NodeContentEquals(funcCall, file.Program(), redisFunction) && (!query.NodeContentEquals(funcCall, file.Program(), clusterRedisFunction))) {
		return nil
	}

	// this Redis() or RedisCluster() invocation is unrelated to redis
	if !redisImported && !constructorImported && !clusterConstructorImported && !clusterModuleImported {
		return nil
	}

	// the redis.cluster.RedisCluster has an incorrect submodule for cluster
	if redisImported && subModule != nil && !query.NodeContentEquals(subModule, file.Program(), "cluster") {
		return nil
	}

	kind := core.PersistRedisNodeKind
	if funcCall.Content(file.Program()) == clusterRedisFunction {
		kind = core.PersistRedisClusterKind
	}

	callDetails, found := getNextCallDetails(parentOfType(funcCall, "call"), file.Program())
	if !found {
		if enableWarnings {
			log.Warn("function call details not found")
		}
		return nil
	}
	functionArgs := callDetails.Arguments

	if _, found := nextMatch(); found {
		if enableWarnings {
			log.Warn("too many assignments for persist_orm")
		}
		return nil
	}

	return &persistResult{
		name:       redisVar.Content(file.Program()),
		expression: args.Content(file.Program()),
		args:       functionArgs,
		kind:       kind,
	}
}

func (p *persister) determinePersistType(f *core.SourceFile, annotation *core.Annotation) (core.PersistKind, *persistResult) {
	log := zap.L().With(logging.FileField(f), logging.AnnotationField(annotation))

	kvR := p.queryKV(f, annotation, false)
	if kvR != nil {
		log.Sugar().Debugf("Determined persist type of '%s'", core.PersistKVKind)
		return core.PersistKVKind, kvR
	}
	fsR := p.queryFS(f, annotation, false)
	if fsR != nil {
		if secret, ok := annotation.Capability.Directives.Bool("secret"); ok && secret {
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
	redisR := p.queryRedis(f, annotation, false)
	if redisR != nil {
		log.Sugar().Debugf("Determined persist type of '%s'", redisR.kind)
		return redisR.kind, redisR
	}
	return "", nil
}
