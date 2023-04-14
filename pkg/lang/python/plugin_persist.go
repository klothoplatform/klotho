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

func (p Persist) Transform(input *core.InputFiles, fileDeps *core.FileDependencies, constructGraph *core.ConstructGraph) error {
	persister := &persister{ConstructGraph: constructGraph, runtime: p.runtime}

	var errs multierr.Error
	for _, unit := range core.GetResourcesOfType[*core.ExecutionUnit](constructGraph) {
		err := persister.handleFiles(unit)
		if err != nil {
			errs.Append(err)
			continue
		}
	}

	return errs.ErrOrNil()
}

type persister struct {
	ConstructGraph *core.ConstructGraph
	runtime        Runtime
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
			errs.Append(core.WrapErrf(err, "failed to handle persist in unit %s", unit.ID))
		}

		for _, r := range resources {
			p.ConstructGraph.AddConstruct(r)

			// a file containing capabilities without an execution unit indicates that the file's capabilities
			// are imported by execution units in one or more separate files
			if core.FileExecUnitName(pySource) != "" || p.isFileReferencedByExecUnit(pySource, unit) {
				p.ConstructGraph.AddDependency(unit.Id(), r.Id())
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

func (p *persister) handleFile(f *core.SourceFile, unit *core.ExecutionUnit) ([]core.Construct, error) {
	annots := f.Annotations()
	newFile := f.CloneSourceFile()

	var resources []core.Construct

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

		construct, pResult := p.determinePersistType(f, annot)
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

		var doTransform func(original *core.SourceFile, modified *core.SourceFile, cap *core.Annotation, result *persistResult, unit *core.ExecutionUnit) (core.Construct, error)
		var err error
		switch construct.(type) {
		case *core.Kv:
			doTransform = p.transformKV
			err = p.runtime.AddKvRuntimeFiles(unit)
		case *core.Fs:
			doTransform = p.transformFS
		case *core.Secrets:
			doTransform = p.transformSecret
			err = p.runtime.AddSecretRuntimeFiles(unit)
		case *core.Orm:
			doTransform = p.transformORM
			err = p.runtime.AddOrmRuntimeFiles(unit)
		case *core.RedisCluster:
			doTransform = p.transformRedis
		case *core.RedisNode:
			doTransform = p.transformRedis
		default:
			errs.Append(core.NewCompilerError(
				f,
				annot,
				fmt.Errorf("type is invalid for the persist capability"),
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

func (p *persister) transformKV(original *core.SourceFile, modified *core.SourceFile, cap *core.Annotation, kvR *persistResult, unit *core.ExecutionUnit) (core.Construct, error) {

	// add the kv runtime import to the file containing a persisted aiocache instance
	kvConfig := p.runtime.GetKvRuntimeConfig()
	err := AddRuntimeImport(kvConfig.Imports, modified)
	if err != nil {
		return nil, errors.Wrap(err, "could not reparse KV transformation")
	}

	// replace the aiocache.Cache() invocation's arguments with those required for the runtime
	nodeContent := cap.Node.Content()
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

	result := &core.Kv{
		AnnotationKey: core.AnnotationKey{
			ID:         cap.Capability.ID,
			Capability: cap.Capability.Name,
		},
	}

	return result, nil
}

func (p *persister) transformFS(original *core.SourceFile, modified *core.SourceFile, cap *core.Annotation, fsR *persistResult, unit *core.ExecutionUnit) (core.Construct, error) {
	result := &core.Fs{
		AnnotationKey: core.AnnotationKey{
			ID:         cap.Capability.ID,
			Capability: cap.Capability.Name,
		},
	}
	nodeContent := cap.Node.Content()

	replaceString := p.runtime.GetFsRuntimeImportClass(cap.Capability.ID, fsR.name)

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
	fsEnvVar := core.GenerateBucketEnvVar(result)

	unit.EnvironmentVariables.Add(fsEnvVar)

	err = p.runtime.AddFsRuntimeFiles(unit, fsEnvVar.Name, cap.Capability.ID)
	if err != nil {
		return nil, errors.Wrap(err, "could not add FS runtime")
	}
	return result, nil
}

func (p *persister) transformSecret(original *core.SourceFile, modified *core.SourceFile, cap *core.Annotation, secretR *persistResult, unit *core.ExecutionUnit) (core.Construct, error) {

	nodeContent := cap.Node.Content()

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
		AnnotationKey: core.AnnotationKey{
			ID:         cap.Capability.ID,
			Capability: cap.Capability.Name,
		},
		Secrets: secrets,
	}

	return result, nil
}

func (p *persister) transformORM(original *core.SourceFile, modified *core.SourceFile, cap *core.Annotation, ormR *persistResult, unit *core.ExecutionUnit) (core.Construct, error) {
	result := &core.Orm{
		AnnotationKey: core.AnnotationKey{
			ID:         cap.Capability.ID,
			Capability: cap.Capability.Name,
		},
	}
	nodeContent := cap.Node.Content()

	newContent := nodeContent
	err := AddRuntimeImport("import os", modified)
	if err != nil {
		return nil, errors.Wrap(err, "could not reparse ORM transformation")
	}
	envVar := core.GenerateOrmConnStringEnvVar(result)

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

	unit.EnvironmentVariables.Add(envVar)

	return result, nil
}

func (p *persister) transformRedis(original *core.SourceFile, modified *core.SourceFile, cap *core.Annotation, redisR *persistResult, unit *core.ExecutionUnit) (core.Construct, error) {

	nodeContent := cap.Node.Content()

	err := AddRuntimeImport("import os", modified)
	if err != nil {
		return nil, errors.Wrap(err, "could not reparse Redis transformation")
	}
	var result core.Construct
	isCluster := false
	switch redisR.construct.(type) {
	case *core.RedisCluster:
		result = &core.RedisCluster{
			AnnotationKey: core.AnnotationKey{
				ID:         cap.Capability.ID,
				Capability: cap.Capability.Name,
			},
		}
		isCluster = true
	case *core.RedisNode:
		result = &core.RedisNode{
			AnnotationKey: core.AnnotationKey{
				ID:         cap.Capability.ID,
				Capability: cap.Capability.Name,
			},
		}
	}

	newContent := nodeContent

	hostEnvVar := core.GenerateRedisHostEnvVar(result)
	portEnvVar := core.GenerateRedisPortEnvVar(result)

	args := redisR.args
	args = AddOrReplaceArg(FunctionArg{
		Name:  "host",
		Value: fmt.Sprintf(`os.environ.get("%s")`, hostEnvVar.Name),
	}, args)
	args = AddOrReplaceArg(FunctionArg{
		Name:  "port",
		Value: fmt.Sprintf(`os.environ.get("%s")`, portEnvVar.Name),
	}, args)
	if isCluster {
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
	unit.EnvironmentVariables.Add(hostEnvVar)
	unit.EnvironmentVariables.Add(portEnvVar)

	return result, nil
}

type persistResult struct {
	expression string
	name       string
	args       []FunctionArg
	construct  core.Construct
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
	if !aiocacheImported && !query.NodeContentEquals(function, cacheFunction) {
		return nil
	}

	// this Cache() invocation belongs to an object other the aiocache module
	if aiocacheImported && functionHost != nil && !query.NodeContentEquals(functionHost, functionHostName) {
		return nil
	}

	// this Cache() invocation is unrelated to aiocache
	if !aiocacheImported && !cacheImported {
		return nil
	}

	callDetails, found := getNextCallDetails(parentOfType(function, "call"))
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
		name:       name.Content(),
		expression: expression.Content(),
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
		if !query.NodeContentEquals(alias, varName) {
			return nil
		}
	} else if !query.NodeContentEquals(module, varName) {
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
		expression: importStatement.Content(),
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
	if !sqlalchemyImported && !query.NodeContentEquals(funcCall, engineFunction) {
		return nil
	}

	// this create_engine() invocation belongs to an object other the aiocache module
	if sqlalchemyImported && module != nil && !query.NodeContentEquals(module, sqlalchemyImportName) {
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
		name:       engineVar.Content(),
		expression: connString.Content(),
		construct:  &core.Orm{},
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

		if !query.NodeContentEquals(module, name) {
			continue
		}

		if !query.NodeContentEquals(moduleMethod, "open") {
			continue
		}

		if varIn.Content() != varOut.Content() {
			continue
		}

		if query.NodeContentEquals(funcCall, "read") {
			if path != nil {
				sn, err := stringLiteralContent(path)
				if err != nil {
					return nil, errors.Errorf("'%s' unable to get path from.", path.Content())
				}
				secrets = append(secrets, sn)
			} else {
				return nil, errors.New("must supply static string for secret path")
			}
		} else {
			return nil, errors.Errorf("'%s' not implemented for secrets persist.", funcCall.Content())
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
	if redisImported && !clusterModuleImported && module != nil && !query.NodeContentEquals(module, redisImportName) {
		return nil
	}

	// import is similar to `from redis import cluster` and the RedisCluster call does not use cluster module
	if clusterModuleImported && !query.NodeContentEquals(module, clustermoduleImportName) {
		return nil
	}

	// Redis is not self imported and the function call does not match the redis or redis cluster function call from the import
	if !redisImported && (!query.NodeContentEquals(funcCall, redisFunction) && (!query.NodeContentEquals(funcCall, clusterRedisFunction))) {
		return nil
	}

	// this Redis() or RedisCluster() invocation is unrelated to redis
	if !redisImported && !constructorImported && !clusterConstructorImported && !clusterModuleImported {
		return nil
	}

	// the redis.cluster.RedisCluster has an incorrect submodule for cluster
	if redisImported && subModule != nil && !query.NodeContentEquals(subModule, "cluster") {
		return nil
	}

	var construct core.Construct

	construct = &core.RedisNode{}
	if funcCall.Content() == clusterRedisFunction {
		construct = &core.RedisCluster{}
	}

	callDetails, found := getNextCallDetails(parentOfType(funcCall, "call"))
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
		name:       redisVar.Content(),
		expression: args.Content(),
		args:       functionArgs,
		construct:  construct,
	}
}

func (p *persister) determinePersistType(f *core.SourceFile, annotation *core.Annotation) (core.Construct, *persistResult) {
	log := zap.L().With(logging.FileField(f), logging.AnnotationField(annotation))

	kvR := p.queryKV(f, annotation, false)
	if kvR != nil {
		log.Sugar().Debugf("Determined persist type of kv")
		return &core.Kv{}, kvR
	}
	fsR := p.queryFS(f, annotation, false)
	if fsR != nil {
		if secret, ok := annotation.Capability.Directives.Bool("secret"); ok && secret {
			log.Sugar().Debugf("Determined persist type of secrets")
			return &core.Secrets{}, fsR
		}
		log.Sugar().Debugf("Determined persist type of fs")
		return &core.Fs{}, fsR
	}
	ormR := p.queryORM(f, annotation, false)
	if ormR != nil {
		log.Sugar().Debugf("Determined persist type of orm")
		return &core.Orm{}, ormR
	}
	redisR := p.queryRedis(f, annotation, false)
	if redisR != nil {
		log.Sugar().Debugf("Determined persist type of redis")
		return redisR.construct, redisR
	}
	return nil, nil
}
