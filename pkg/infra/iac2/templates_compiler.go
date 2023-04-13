package iac2

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"path"
	"reflect"
	"sort"
	"strings"
	"text/template"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/infra/kubernetes"
	"github.com/klothoplatform/klotho/pkg/lang/javascript"
	"github.com/klothoplatform/klotho/pkg/multierr"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

type (
	stringTemplateValue struct {
		raw   any
		value string
	}

	templateValue interface {
		Parse() (string, error)
		Raw() any
	}

	templatesProvider struct {
		// templates is the fs.FS where we read all of our `<struct>/factory.ts` files
		templates fs.FS
		// resourceTemplatesByStructName is a cache from struct name (e.g. "CloudwatchLogs") to the template for that struct.
		resourceTemplatesByStructName map[string]ResourceCreationTemplate
		childTemplatesByPath          map[string]*template.Template
	}

	// TemplatesCompiler renders a graph of [core.Resource] nodes by combining each one with its corresponding
	// ResourceCreationTemplate
	TemplatesCompiler struct {
		*templatesProvider
		// resourceGraph is the graph of resources to render
		resourceGraph *core.ResourceGraph // TODO make this be a core.ResourceGraph, and un-expose that struct's Underlying
		// resourceVarNames is a set of all variable names
		resourceVarNames map[string]struct{}
		// resourceVarNamesById is a map from resource id to the variable name for that resource
		resourceVarNamesById map[string]string
		// ctx is a pointer to the current context being used within the templates compiler. This context is used when parsing values within nested templates.
		ctx *NestedCtx
	}
	NestedCtx struct {
		useDoubleQuotes bool
		appliedOutputs  *[]AppliedOutput
		rootVal         *reflect.Value
	}
)

var (
	//go:embed templates/*/factory.ts templates/*/package.json templates/*/*.ts.tmpl
	standardTemplates embed.FS
)

func (s stringTemplateValue) Parse() (string, error) {
	return s.value, nil
}

func (s stringTemplateValue) Raw() interface{} {
	return s.raw
}

func CreateTemplatesCompiler(resources *core.ResourceGraph) *TemplatesCompiler {
	return &TemplatesCompiler{
		templatesProvider:    standardTemplatesProvider(),
		resourceGraph:        resources,
		resourceVarNames:     make(map[string]struct{}),
		resourceVarNamesById: make(map[string]string),
	}
}

func standardTemplatesProvider() *templatesProvider {
	subTemplates, err := fs.Sub(standardTemplates, "templates")
	if err != nil {
		panic(err) // unexpected, since standardTemplates is statically built into klotho
	}
	return &templatesProvider{
		templates:                     subTemplates,
		resourceTemplatesByStructName: make(map[string]ResourceCreationTemplate),
		childTemplatesByPath:          make(map[string]*template.Template),
	}
}

func (tc TemplatesCompiler) RenderBody(out io.Writer) error {
	errs := multierr.Error{}
	vertexIds, err := tc.resourceGraph.VertexIdsInTopologicalOrder()
	if err != nil {
		return err
	}
	for i := len(vertexIds) - 1; i >= 0; i-- {
		id := vertexIds[i]
		resource := tc.resourceGraph.GetResource(id)
		err := tc.renderResource(out, resource)
		errs.Append(err)
		if i > 0 {
			_, err = out.Write([]byte("\n\n"))
			if err != nil {
				return err
			}
		}

	}
	return errs.ErrOrNil()
}

func (tc TemplatesCompiler) RenderImports(out io.Writer) error {
	errs := multierr.Error{}

	allImports := make(map[string]struct{})
	for _, res := range tc.resourceGraph.ListResources() {
		tmpl, err := tc.getTemplate(res)
		if err != nil {
			errs.Append(err)
			continue
		}
		for statement := range tmpl.Imports {
			allImports[statement] = struct{}{}
		}
	}
	if err := errs.ErrOrNil(); err != nil {
		return err
	}

	sortedImports := make([]string, 0, len(allImports))
	for statement := range allImports {
		sortedImports = append(sortedImports, statement)
	}

	sort.Strings(sortedImports)
	for _, statement := range sortedImports {
		if _, err := out.Write([]byte(statement)); err != nil {
			return err
		}
		if _, err := out.Write([]byte("\n")); err != nil {
			return err
		}
	}

	return nil
}

func (tc TemplatesCompiler) RenderPackageJSON() (*javascript.NodePackageJson, error) {
	errs := multierr.Error{}
	mainPJson := javascript.NodePackageJson{}
	for _, res := range tc.resourceGraph.ListResources() {
		pJson, err := tc.GetPackageJSON(res)
		if err != nil {
			errs.Append(err)
			continue
		}
		mainPJson.Merge(&pJson)
	}
	if err := errs.ErrOrNil(); err != nil {
		return &mainPJson, err
	}
	return &mainPJson, nil
}

func (tc TemplatesCompiler) renderResource(out io.Writer, resource core.Resource) error {
	defer func() {
		r := recover()
		if r == nil {
			return
		}
		panic(errors.Errorf("panic rendering resource %s: %v", resource.Id(), r))
	}()

	tmpl, err := tc.getTemplate(resource)
	if err != nil {
		return err
	}

	errs := multierr.Error{}

	resourceVal := reflect.ValueOf(resource)
	for resourceVal.Kind() == reflect.Pointer {
		resourceVal = resourceVal.Elem()
	}
	inputArgs := make(map[string]templateValue)
	for fieldName := range tmpl.InputTypes {
		func(fieldName string) {
			defer func() {
				r := recover()
				if r == nil {
					return
				}
				panic(errors.Errorf("panic rendering field %s: %v", fieldName, r))
			}()
		switch fieldName {
		// dependsOn will be a reserved field for us to use to map dependencies. If specified as an Arg we will automatically call resolveDependencies
		case "dependsOn":
			inputArgs[fieldName] = stringTemplateValue{value: tc.resolveDependencies(resource)}
			return
		case "protect":
			inputArgs[fieldName] = stringTemplateValue{value: "protect", raw: "protect"}
			return
		case "awsProfile":
			inputArgs[fieldName] = stringTemplateValue{value: "protect", raw: "awsProfile"}
			return
		}
		childVal := resourceVal.FieldByName(fieldName)

		var appliedoutputs []AppliedOutput
		buf := strings.Builder{}
		strValue, err := tc.resolveStructInput(&resourceVal, childVal, false, &appliedoutputs)
		if err != nil {
			errs.Append(err)
			return
		}
		uniqueOutputs, err := deduplicateAppliedOutputs(appliedoutputs)
		if err != nil {
			errs.Append(err)
			return
		}
		_, err = buf.WriteString(appliedOutputsToString(uniqueOutputs))
		if err != nil {
			errs.Append(err)
			return
		}
		buf.WriteString(strValue)
		if len(uniqueOutputs) > 0 {
			_, err = buf.WriteString("})")
			if err != nil {
				errs.Append(err)
				return
			}
		}

		var rawVal any
		if childVal.IsValid() {
			rawVal = childVal.Interface()
		}

		resolvedValue := stringTemplateValue{value: buf.String(), raw: rawVal}

			if err != nil {
				errs.Append(err)
			} else {
				inputArgs[fieldName] = resolvedValue
			}
		}(fieldName)
	}
	if err := errs.ErrOrNil(); err != nil {
		return err
	}

	if tmpl.OutputType != "void" {
		varName := tc.getVarName(resource)
		fmt.Fprintf(out, `const %s = `, varName)
	}
	errs.Append(tmpl.RenderCreate(out, inputArgs, tc))
	_, err = out.Write([]byte(";"))
	if err != nil {
		return err
	}
	errs.Append(tc.renderGlueVars(out, resource))
	return errs.ErrOrNil()
}

// resolveDependencies creates a string which models an array containing all the variable names, which the resource depends on.
func (tc TemplatesCompiler) resolveDependencies(resource core.Resource) string {
	buf := strings.Builder{}
	buf.WriteRune('[')
	upstreamResources := tc.resourceGraph.GetDownstreamResources(resource)
	numDeps := len(upstreamResources)
	for i := 0; i < numDeps; i++ {
		res := upstreamResources[i]
		buf.WriteString(tc.getVarName(res))
		if i < (numDeps - 1) {
			buf.WriteRune(',')
		}
	}
	buf.WriteRune(']')
	return buf.String()
}

// resolveStructInput translates a value to a form suitable to inject into the typescript as an input to a function.
func (tc TemplatesCompiler) resolveStructInput(resourceVal *reflect.Value, childVal reflect.Value, useDoubleQuotedStrings bool, appliedOutputs *[]AppliedOutput) (string, error) {
	tc.ctx = &NestedCtx{
		useDoubleQuotes: useDoubleQuotedStrings,
		appliedOutputs:  appliedOutputs,
		rootVal:         resourceVal,
	}
	var zeroValue reflect.Value
	if childVal == zeroValue {
		return `null`, nil
	}
	switch childVal.Kind() {
	case reflect.Bool,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return fmt.Sprintf("%v", childVal.Interface()), nil
	case reflect.String:
		return quoteTsString(childVal.Interface().(string), useDoubleQuotedStrings), nil
	case reflect.Struct, reflect.Pointer:
		if childVal.Kind() == reflect.Pointer && childVal.IsNil() {
			return "null", nil
		}
		if typedChild, ok := childVal.Interface().(core.Resource); ok {
			return tc.getVarName(typedChild), nil
		} else if typedChild, ok := childVal.Interface().(core.IaCValue); ok {
			output, err := tc.handleIaCValue(typedChild, appliedOutputs, resourceVal)
			if err != nil {
				return output, err
			}
			return output, nil
		} else {
			val := childVal
			correspondingStruct := val
			for correspondingStruct.Kind() == reflect.Pointer {
				correspondingStruct = val.Elem()
			}

			// Check to see if there is a nested tempalte and if there is use that
			tmpl, err := tc.getNestedTemplate(path.Join(
				camelToSnake(resourceVal.Type().Name()),
				camelToSnake(correspondingStruct.Type().Name()),
			), tc)
			if err != nil {
				return "", err
			}
			if tmpl != nil {
				zap.S().Debugf("Rendering nested template %s, for resource %s", tmpl.Name, correspondingStruct.Type())
				output := bytes.NewBuffer([]byte{})
				err = tmpl.Execute(output, childVal.Interface())
				return output.String(), err
			}
			zap.S().Debugf("Rendering resource %s, as document", correspondingStruct.Type())

			// Last resort, render as a document
			output := strings.Builder{}
			output.WriteString("{")
			for i := 0; i < correspondingStruct.NumField(); i++ {

				childVal := correspondingStruct.Field(i)
				fieldName := correspondingStruct.Type().Field(i).Name

				// If the struct type is PolicyDocument, pass that down to our recursive calls to keep field name upperCased
				if correspondingStruct.Type() == reflect.TypeOf((*resources.PolicyDocument)(nil)).Elem() {
					resourceVal = &correspondingStruct
				}

				resolvedValue, err := tc.resolveStructInput(resourceVal, childVal, false, appliedOutputs)

				if err != nil {
					return output.String(), err
				}

				// If the struct type is not PolicyDocument, we want to camelCase our field names to follow pulumi format
				if resourceVal.Type() != reflect.TypeOf((*resources.PolicyDocument)(nil)).Elem() {
					fieldName = strings.ToLower(string(fieldName[0])) + fieldName[1:]
				}

				// To Prevent us from rendering fields which are not set, only right if the value is non zero for its type
				if !childVal.IsZero() {
					output.WriteString(fmt.Sprintf("%s: %s,\n", fieldName, resolvedValue))
				}
			}
			output.WriteString("}")
			return output.String(), nil
		}
	case reflect.Array, reflect.Slice:
		sliceLen := childVal.Len()

		buf := strings.Builder{}
		buf.WriteRune('[')
		for i := 0; i < sliceLen; i++ {
			output, err := tc.resolveStructInput(resourceVal, childVal.Index(i), false, appliedOutputs)
			if err != nil {
				return output, err
			}
			buf.WriteString(output)
			if i < (sliceLen - 1) {
				buf.WriteRune(',')
			}
		}
		buf.WriteRune(']')
		return buf.String(), nil
	case reflect.Map:
		mapLen := childVal.Len()

		buf := strings.Builder{}
		buf.WriteRune('{')
		for i, key := range childVal.MapKeys() {
			output, err := tc.resolveStructInput(resourceVal, key, true, appliedOutputs)
			if err != nil {
				return output, nil
			}
			buf.WriteString(output)
			buf.WriteRune(':')
			output, err = tc.resolveStructInput(resourceVal, childVal.MapIndex(key), false, appliedOutputs)
			if err != nil {
				return output, err
			}
			buf.WriteString(output)
			if i < (mapLen - 1) {
				buf.WriteRune(',')
			}
		}
		buf.WriteRune('}')
		return buf.String(), nil
	case reflect.Interface:
		// This happens when the value is inside a map, slice, or array. Basically, the reflected type is interface{},
		// instead of being the actual type. So, we basically pull the item out of the collection, and then reflect on
		// it directly.
		underlyingVal := childVal.Interface()
		return tc.resolveStructInput(resourceVal, reflect.ValueOf(underlyingVal), false, appliedOutputs)
	}
	return "", nil
}

// handleIaCValue determines how to retrieve values from a resource given a specific value identifier.
func (tc TemplatesCompiler) handleIaCValue(v core.IaCValue, appliedOutputs *[]AppliedOutput, resourceVal *reflect.Value) (string, error) {
	resource := v.Resource
	property := v.Property

	if resource == nil {
		output, err := tc.resolveStructInput(nil, reflect.ValueOf(property), false, appliedOutputs)
		if err != nil {
			return output, err
		}
		return output, nil
	} else if _, ok := v.Resource.(*resources.AvailabilityZones); ok {
		return fmt.Sprintf("%s.names[%s]", tc.getVarName(v.Resource), v.Property), nil
	}
	switch property {
	case string(core.BUCKET_NAME):
		return fmt.Sprintf("%s.bucket", tc.getVarName(resource)), nil
	case resources.ARN_IAC_VALUE:
		return fmt.Sprintf("%s.arn", tc.getVarName(v.Resource)), nil
	case resources.NAME_IAC_VALUE:
		return fmt.Sprintf("%s.name", tc.getVarName(v.Resource)), nil
	case resources.ALL_BUCKET_DIRECTORY_IAC_VALUE:
		return fmt.Sprintf("pulumi.interpolate`${%s.arn}/*`", tc.getVarName(v.Resource)), nil
	case resources.DYNAMODB_TABLE_BACKUP_IAC_VALUE,
		resources.DYNAMODB_TABLE_INDEX_IAC_VALUE,
		resources.DYNAMODB_TABLE_EXPORT_IAC_VALUE,
		resources.DYNAMODB_TABLE_STREAM_IAC_VALUE:
		prop := strings.Split(property, "__")[1]
		return fmt.Sprintf("pulumi.interpolate`${%s.arn}/%s/*`", tc.getVarName(resource), prop), nil
	case resources.LAMBDA_INTEGRATION_URI_IAC_VALUE:
		return fmt.Sprintf("%s.invokeArn", tc.getVarName(resource)), nil
	case core.ALL_RESOURCES_IAC_VALUE:
		return "*", nil
	case string(core.HOST):
		switch resource.(type) {
		case *resources.Elasticache:
			return fmt.Sprintf("pulumi.interpolate`%s.cacheNodes[0].address`", tc.getVarName(resource)), nil
		default:
			return "", errors.Errorf("unsupported resource type %T for '%s'", resource, property)
		}
	case string(core.PORT):
		switch resource.(type) {
		case *resources.Elasticache:
			return fmt.Sprintf("pulumi.interpolate`%s.cacheNodes[0].port`", tc.getVarName(resource)), nil
		default:
			return "", errors.Errorf("unsupported resource type %T for '%s'", resource, property)
		}
	case string(core.CONNECTION_STRING):
		switch res := v.Resource.(type) {
		case *resources.RdsProxy:
			downResources := tc.resourceGraph.GetUpstreamDependencies(res)
			var instance *resources.RdsInstance
			for _, resource := range downResources {
				if rdsProxyTargetGroup, ok := resource.Source.(*resources.RdsProxyTargetGroup); ok {
					instance = rdsProxyTargetGroup.RdsInstance
				}
			}
			if instance == nil {
				return "", errors.Errorf("Rds Proxy, %s, must have an associated instance", v.Resource.Id())
			}

			fetchUsername := fmt.Sprintf(`fs.readFileSync('%s', 'utf-8').split("\n")[1].split('"')[3]`, instance.CredentialsPath)
			fetchPassword := fmt.Sprintf(`fs.readFileSync('%s', 'utf-8').split("\n")[2].split('"')[3]`, instance.CredentialsPath)
			return fmt.Sprintf("pulumi.interpolate`postgresql://${%s}:${%s}@${%s.endpoint}:5432/%s`", fetchUsername, fetchPassword,
				tc.getVarName(v.Resource), instance.DatabaseName), nil
		default:
			return "", errors.Errorf("unsupported resource type %T for '%s'", v.Resource, v.Property)
		}

	case resources.CLUSTER_OIDC_ARN_IAC_VALUE:
		varName := "cluster_oidc_url"
		*appliedOutputs = append(*appliedOutputs, AppliedOutput{
			appliedName: fmt.Sprintf("%s.openIdConnectIssuerUrl", tc.getVarName(v.Resource)),
			varName:     varName,
		})

		arnVarName := "cluster_arn"
		*appliedOutputs = append(*appliedOutputs, AppliedOutput{
			appliedName: fmt.Sprintf("%s.arn", tc.getVarName(v.Resource)),
			varName:     arnVarName,
		})
		return fmt.Sprintf("`arn:aws:iam::${%s.split(':')[4]}:oidc-provider/${%s}`", arnVarName, varName), nil
	case resources.CLUSTER_OIDC_URL_IAC_VALUE:
		varName := "cluster_oidc_url"
		*appliedOutputs = append(*appliedOutputs, AppliedOutput{
			appliedName: fmt.Sprintf("%s.openIdConnectIssuerUrl", tc.getVarName(v.Resource)),
			varName:     varName,
		})
		return fmt.Sprintf("[`${%s}:sub`]", varName), nil
	case resources.CLUSTER_CA_DATA_IAC_VALUE:
		return fmt.Sprintf("%s.certificateAuthorityData", tc.getVarName(v.Resource)), nil
	case resources.CLUSTER_ENDPOINT_IAC_VALUE:
		return fmt.Sprintf("%s.endpoint", tc.getVarName(v.Resource)), nil
	case resources.CLUSTER_PROVIDER_IAC_VALUE:
		if kcfg, ok := v.Resource.(*kubernetes.Kubeconfig); ok {
			p := &KubernetesProvider{Name: fmt.Sprintf("%s-provider", kcfg.Name)}
			return tc.getVarNameByResourceId(p.Id()), nil
		}
	case resources.ALL_RESOURCES_ARN_IAC_VALUE:
		method, ok := v.Resource.(*resources.ApiMethod)
		if !ok {
			return "", errors.Errorf("unsupported resource type %T for '%s'", v.Resource, v.Property)
		}
		verb := strings.ToUpper(method.HttpMethod)
		if verb == "ANY" {
			verb = "*"
		}
		accountId := resources.NewAccountId()
		region := resources.NewRegion()
		path := "/"
		if method.Resource != nil {
			path = fmt.Sprintf("${%s.path}", tc.getVarName(method.Resource))
		}
		return fmt.Sprintf("pulumi.interpolate`arn:aws:execute-api:${%s.name}:${%s.accountId}:${%s.id}/*/%s%s`", tc.getVarName(region),
			tc.getVarName(accountId), tc.getVarName(method.RestApi), verb, path), nil
	case resources.STAGE_INVOKE_URL_IAC_VALUE:
		return fmt.Sprintf("%s.invokeUrl.apply((d) => d.split('//')[1].split('/')[0])", tc.getVarName(v.Resource)), nil
	case resources.ECR_IMAGE_NAME_IAC_VALUE:
		return fmt.Sprintf(`%s.imageName`, tc.getVarName(resource)), nil
	case resources.NLB_INTEGRATION_URI_IAC_VALUE:
		integration, ok := resourceVal.Interface().(resources.ApiIntegration)
		if !ok {
			return "", errors.Errorf("Unable to handle iac value for %s on type %s", resources.NLB_INTEGRATION_URI_IAC_VALUE, resourceVal.Type().Name())
		}
		return fmt.Sprintf("pulumi.interpolate`http://${%s.dnsName}%s`", tc.getVarName(resource), integration.Route), nil
	case resources.RDS_CONNECTION_ARN_IAC_VALUE:
		switch res := v.Resource.(type) {
		case *resources.RdsInstance:
			accountId := resources.NewAccountId()
			region := resources.NewRegion()
			fetchUsername := fmt.Sprintf(`fs.readFileSync('%s', 'utf-8').split("\n")[1].split('"')[3]`, res.CredentialsPath)
			return fmt.Sprintf("pulumi.interpolate`arn:aws:rds-db:${%s.name}:${%s.accountId}:dbuser:${%s.resourceId}/${%s}`", tc.getVarName(region), tc.getVarName(accountId), tc.getVarName(res), fetchUsername), nil
		default:
			return "", errors.Errorf("unsupported resource type %T for '%s'", v.Resource, v.Property)
		}
	case resources.CIDR_BLOCK_IAC_VALUE:
		return fmt.Sprintf(`%s.cidrBlock`, tc.getVarName(resource)), nil
	}

	return "", errors.Errorf("unsupported IaC Value Property, %s", property)
}

func (tc TemplatesCompiler) handleSingleIaCValue(v core.IaCValue) (string, error) {
	return tc.handleIaCValue(v, nil, nil)
}

// getVarName gets a unique but nice-looking variable for the given item.
//
// It does this by first calculating an ideal variable name, which is a camel-cased ${structName}${Id}. For example, if
// you had an object CoolResource{id: "foo-bar"}, the ideal variable name is coolResourceFooBar.
//
// If that ideal variable name hasn't been used yet, this function returns it. If it has been used, we append `_${i}` to
// it, where ${i} is the lowest positive integer that would give us a new, unique variable name. This isn't expected
// to happen often, if at all, since ids are globally unique.
func (tc TemplatesCompiler) getVarName(v core.Resource) string {
	return tc.getVarNameByResourceId(v.Id())
}

func (tc TemplatesCompiler) getVarNameByResourceId(id string) string {
	if name, alreadyResolved := tc.resourceVarNamesById[id]; alreadyResolved {
		return name
	}
	// Generate something like "lambdaFoo", where Lambda is the name of the struct and "foo" is the id
	desiredName := lowercaseFirst(toUpperCamel(id))
	resolvedName := desiredName
	for i := 1; ; i++ {
		_, varNameTaken := tc.resourceVarNames[resolvedName]
		if varNameTaken {
			resolvedName = fmt.Sprintf("%s_%d", desiredName, i)
		} else {
			break
		}
	}
	tc.resourceVarNames[resolvedName] = struct{}{}
	tc.resourceVarNamesById[id] = resolvedName
	return resolvedName
}

// parseVal parses the supplied value for nested tempaltes
func (tc TemplatesCompiler) parseVal(val reflect.Value) (string, error) {
	return tc.resolveStructInput(tc.ctx.rootVal, val, tc.ctx.useDoubleQuotes, tc.ctx.appliedOutputs)
}

func (tp templatesProvider) getTemplate(v core.Resource) (ResourceCreationTemplate, error) {
	return tp.getTemplateForType(structName(v))
}

func (tp templatesProvider) getTemplateForType(typeName string) (ResourceCreationTemplate, error) {
	existing, ok := tp.resourceTemplatesByStructName[typeName]
	if ok {
		return existing, nil
	}
	templateName := camelToSnake(typeName)
	contents, err := fs.ReadFile(tp.templates, templateName+`/factory.ts`)
	if err != nil {
		return ResourceCreationTemplate{}, err
	}
	template := ParseResourceCreationTemplate(typeName, contents)
	tp.resourceTemplatesByStructName[typeName] = template
	return template, nil
}

func (tp templatesProvider) getNestedTemplate(templatePath string, tc TemplatesCompiler) (*template.Template, error) {
	templateFilePaths := []string{
		templatePath + ".ts.tmpl",
		templatePath + ".ts",
	}

	existing, ok := tp.childTemplatesByPath[templatePath]
	if ok {
		return existing, nil
	}

	var contents []byte
	var merr multierr.Error
	var err error
	for _, tfPath := range templateFilePaths {
		contents, err = fs.ReadFile(tp.templates, tfPath)
		if err == nil {
			break
		} else {
			merr.Append(err)
		}
	}
	// If we dont have any contents we dont have a nested template for the resource, so fall back to the document route
	if len(contents) == 0 {
		return nil, nil
	}

	tmpl, err := template.New(templatePath).Funcs(template.FuncMap{
		"parseVal": tc.parseVal,
	}).Parse(string(contents))
	if err != nil {
		return nil, errors.Wrapf(err, `while writing template for %s`, templatePath)
	}
	tp.childTemplatesByPath[templatePath] = tmpl
	return tmpl, nil
}

func (tc TemplatesCompiler) GetPackageJSON(v core.Resource) (javascript.NodePackageJson, error) {
	packageContent := javascript.NodePackageJson{}
	typeName := structName(v)
	templateName := camelToSnake(typeName)
	contents, err := fs.ReadFile(tc.templates, templateName+`/package.json`)
	if err != nil {
		return *packageContent.Clone(), err
	}
	err = json.NewDecoder(bytes.NewReader(contents)).Decode(&packageContent)
	if err != nil {
		return *packageContent.Clone(), err
	}
	return *packageContent.Clone(), nil
}

// renderGlueVars renders additional variables associated with a given resource that do not represent specific cloud resources
func (tc TemplatesCompiler) renderGlueVars(out io.Writer, resource core.Resource) error {
	var errs multierr.Error
	switch resource := resource.(type) {
	case *kubernetes.Kubeconfig:
		errs.Append(tc.renderKubernetesProvider(out, resource))
	}
	return errs.ErrOrNil()
}

func (tc TemplatesCompiler) renderKubernetesProvider(out io.Writer, kubeconfig *kubernetes.Kubeconfig) error {
	var errs multierr.Error
	provider := &KubernetesProvider{
		Name:          fmt.Sprintf("%s-provider", kubeconfig.Name),
		ConstructsRef: kubeconfig.ConstructsRef,
		KubeConfig:    kubeconfig,
	}
	_, err := out.Write([]byte("\n"))
	errs.Append(err)
	errs.Append(tc.renderResource(out, provider))
	return errs.ErrOrNil()
}
