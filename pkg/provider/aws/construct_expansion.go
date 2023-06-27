package aws

import (
	"fmt"
	"path/filepath"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/multierr"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
	"github.com/pkg/errors"
)

func (a *AWS) expandStaticUnit(dag *core.ResourceGraph, unit *core.StaticUnit) ([]core.Resource, error) {
	errs := multierr.Error{}
	bucket, err := core.CreateResource[*resources.S3Bucket](dag, resources.S3BucketCreateParams{
		AppName: a.AppName,
		Refs:    core.BaseConstructSetOf(unit),
		Name:    unit.Name,
	})
	if err != nil {
		return nil, err
	}
	for _, f := range unit.Files() {
		object, err := core.CreateResource[*resources.S3Object](dag, resources.S3ObjectCreateParams{
			AppName:  a.AppName,
			Refs:     core.BaseConstructSetOf(unit),
			Name:     fmt.Sprintf("%s-%s", unit.Name, filepath.Base(f.Path())),
			Key:      f.Path(),
			FilePath: filepath.Join(unit.Name, f.Path()),
		})
		if err != nil {
			errs.Append(err)
			continue
		}
		object.Bucket = bucket
		dag.AddDependenciesReflect(object)
	}
	return []core.Resource{bucket}, nil
}

func (a *AWS) expandSecrets(dag *core.ResourceGraph, construct *core.Secrets) ([]core.Resource, error) {
	mappedResources := []core.Resource{}
	for _, secretName := range construct.Secrets {
		secret, err := core.CreateResource[*resources.Secret](dag, resources.SecretCreateParams{
			AppName: a.AppName,
			Refs:    core.BaseConstructSetOf(construct),
			Name:    secretName,
		})
		if err != nil {
			return mappedResources, err
		}
		secretVersion, err := core.CreateResource[*resources.SecretVersion](dag, resources.SecretVersionCreateParams{
			AppName:      a.AppName,
			Refs:         core.BaseConstructSetOf(construct),
			Name:         secretName,
			DetectedPath: secretName,
		})
		if err != nil {
			return mappedResources, err
		}
		secretVersion.Secret = secret
		dag.AddDependenciesReflect(secretVersion)
		mappedResources = append(mappedResources, secret)
	}
	return mappedResources, nil
}

func (a *AWS) expandRedisNode(dag *core.ResourceGraph, construct *core.RedisNode) ([]core.Resource, error) {
	mappedResources := []core.Resource{}
	redis, err := core.CreateResource[*resources.ElasticacheCluster](dag, resources.ElasticacheClusterCreateParams{
		AppName: a.AppName,
		Refs:    core.BaseConstructSetOf(construct),
		Name:    construct.Name,
	})
	if err != nil {
		return mappedResources, err
	}
	mappedResources = append(mappedResources, redis)
	return mappedResources, nil
}

// expandOrm takes in a single orm construct and expands the generic construct into a set of resource's based on the units configuration.
func (a *AWS) expandOrm(dag *core.ResourceGraph, orm *core.Orm, constructType string) ([]core.Resource, error) {
	mappedResources := []core.Resource{}
	if constructType == "" {
		constructType = resources.RDS_INSTANCE_TYPE
	}
	switch constructType {
	case resources.RDS_INSTANCE_TYPE:
		instance, err := core.CreateResource[*resources.RdsInstance](dag, resources.RdsInstanceCreateParams{
			AppName: a.AppName,
			Refs:    core.BaseConstructSetOf(orm),
			Name:    orm.Name,
		})
		if err != nil {
			return mappedResources, err
		}
		mappedResources = append(mappedResources, instance)
	default:
		return mappedResources, fmt.Errorf("unsupported orm type %s", constructType)
	}
	return mappedResources, nil
}

func (a *AWS) expandKv(dag *core.ResourceGraph, kv *core.Kv) ([]core.Resource, error) {
	mappedResources := []core.Resource{}
	table, err := core.CreateResource[*resources.DynamodbTable](dag, resources.DynamodbTableCreateParams{
		AppName: a.AppName,
		Refs:    core.BaseConstructSetOf(kv),
		Name:    "kv",
	})
	if err != nil {
		return mappedResources, err
	}

	mappedResources = append(mappedResources, table)
	return mappedResources, nil
}

func (a *AWS) expandFs(dag *core.ResourceGraph, fs core.Construct) ([]core.Resource, error) {
	mappedResources := []core.Resource{}
	bucket, err := core.CreateResource[*resources.S3Bucket](dag, resources.S3BucketCreateParams{
		AppName: a.AppName,
		Refs:    core.BaseConstructSetOf(fs),
		Name:    fs.Id().Name,
	})
	if err != nil {
		return mappedResources, err
	}
	mappedResources = append(mappedResources, bucket)
	return mappedResources, nil
}

func (a *AWS) expandExpose(dag *core.ResourceGraph, expose *core.Gateway, constructType string) ([]core.Resource, error) {
	mappedResources := []core.Resource{}
	if constructType == "" {
		constructType = resources.API_GATEWAY_REST_TYPE
	}
	switch constructType {
	case resources.API_GATEWAY_REST_TYPE:
		stage, err := core.CreateResource[*resources.ApiStage](dag, resources.ApiStageCreateParams{
			AppName: a.AppName,
			Refs:    core.BaseConstructSetOf(expose),
			Name:    expose.Name,
		})
		if err != nil {
			return mappedResources, err
		}
		mappedResources = append(mappedResources, stage.RestApi)
	default:
		return mappedResources, fmt.Errorf("unsupported expose type %s", constructType)
	}
	return mappedResources, nil
}

// expandExecutionUnit takes in a single execution unit and expands the generic construct into a set of resource's based on the units configuration.
func (a *AWS) expandExecutionUnit(dag *core.ResourceGraph, unit *core.ExecutionUnit, constructType string, attributes map[string]any) ([]core.Resource, error) {
	mappedResources := []core.Resource{}
	if constructType == "" {
		constructType = resources.LAMBDA_FUNCTION_TYPE
	}
	switch constructType {
	case resources.LAMBDA_FUNCTION_TYPE:
		lambda, err := core.CreateResource[*resources.LambdaFunction](dag, resources.LambdaCreateParams{
			AppName: a.AppName,
			Refs:    core.BaseConstructSetOf(unit),
			Name:    unit.Name,
		})
		if err != nil {
			return mappedResources, err
		}
		mappedResources = append(mappedResources, lambda)
	case resources.EC2_INSTANCE_TYPE:
		instance, err := core.CreateResource[*resources.Ec2Instance](dag, resources.Ec2InstanceCreateParams{
			AppName: a.AppName,
			Refs:    core.BaseConstructSetOf(unit),
			Name:    unit.Name,
		})
		if err != nil {
			return mappedResources, err
		}
		mappedResources = append(mappedResources, instance)
	case resources.ECS_SERVICE_TYPE:
		var networkPlacement string
		np, found := attributes["networkPlacement"]
		if found {
			networkPlacement = np.(string)
		} else {
			networkPlacement = "private"
		}
		ecsService, err := core.CreateResource[*resources.EcsService](dag, resources.EcsServiceCreateParams{
			AppName:          a.AppName,
			Refs:             core.BaseConstructSetOf(unit),
			Name:             unit.Name,
			LaunchType:       resources.LAUNCH_TYPE_FARGATE,
			NetworkPlacement: networkPlacement,
		})
		if err != nil {
			return mappedResources, err
		}
		mappedResources = append(mappedResources, ecsService)
	default:
		return mappedResources, fmt.Errorf("unsupported execution unit type %s", constructType)
	}
	return mappedResources, nil
}

func (a *AWS) expandConfig(dag *core.ResourceGraph, construct *core.Config) ([]core.Resource, error) {
	mappedResources := []core.Resource{}
	if !construct.Secret {
		mappedResources := []core.Resource{}
		return mappedResources, errors.Errorf("unsupported: non-secret config for annotation '%s'", construct.Name)
	}
	secretVersion, err := core.CreateResource[*resources.SecretVersion](dag, resources.SecretVersionCreateParams{
		AppName: a.AppName,
		Refs:    core.BaseConstructSetOf(construct),
		Name:    construct.Name,
	})
	if err != nil {
		return mappedResources, err
	}

	mappedResources = append(mappedResources, secretVersion.Secret)
	return mappedResources, nil
}
