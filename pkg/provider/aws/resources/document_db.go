package resources

import (
	"fmt"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/sanitization/aws"
	"github.com/pkg/errors"
)

const (
	DOCUMENT_DB_CLUSTER_TYPE = "document_db_cluster"
)

type (
	DocumentDbCluster struct {
		Name              string
		ConstructsRef     core.AnnotationKeySet
		AvailabilityZones *AvailabilityZones
		MasterUsername    string
		MasterPassword    string
	}

	DocumentDbClusterCreateParams struct {
		Name string
		Refs core.AnnotationKeySet
	}

	DocumentDbClusterConfigureParams struct {
		MasterUsername string
		MasterPassword string
	}
)

func (dc *DocumentDbCluster) Id() core.ResourceId {
	return core.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     DOCUMENT_DB_CLUSTER_TYPE,
		Name:     dc.Name,
	}
}

func (dc *DocumentDbCluster) KlothoConstructRef() core.AnnotationKeySet {
	return dc.ConstructsRef
}

func (dc *DocumentDbCluster) Create(dag *core.ResourceGraph, params DocumentDbClusterCreateParams) error {
	dc.Name = aws.DocumentDbClusterSanitizer.Apply(params.Name)

	if existing := dag.GetResource(dc.Id()); existing != nil {
		// It's fine if this already exists; just add the refs and use the existing one. In that case, assume the AZs
		// and instances are already set up (but not the instances; we'll do those in a sec).
		existing, ok := existing.(*DocumentDbCluster)
		if !ok {
			return errors.Errorf(`found an existing element at %s, but it was not an S3Bucket`, existing.Id().String())
		}
		existing.ConstructsRef.AddAll(params.Refs)
		// TODO: warn if the
		return nil
	} else {
		dc.AvailabilityZones = NewAvailabilityZones()
	}

	// This really should be a configuration param, but it can't be: see issue #675
	// We can't use CreateDependencies here, because even though the instances are operational dependencies for the
	// cluster (that is, the cluster can't work without instances), in terms of the DAG, the cluster is the dependency
	// for the instances (that is, we can't create instances without the cluster). This is backwards from what the DAG
	// usually assumes, so we can't use the usual pattern. In particular, we can't use dag.CreateDependencies because
	// the cluster doesn't actually have any reference to its instances.
	alreadyHasInstance := false
	for _, instance := range core.ListResources[*DocumentDbInstance](dag) {
		if instance.Cluster.Id() == dc.Id() {
			alreadyHasInstance = true
		}
	}
	if alreadyHasInstance {
		instanceNum := 1
		instance, err := core.CreateResource[*DocumentDbInstance](dag, DocumentDbInstance{
			Name:            fmt.Sprintf(`%s-%03d`, dc.Name, instanceNum),
			ConstructorRefs: params.Refs,
			Cluster:         dc,
		})
		if err != nil {
			return err
		}
		dag.AddDependency(instance, dc)
	}
	return nil
}

func (dc *DocumentDbCluster) Configure(params DocumentDbClusterConfigureParams) error {
	dc.MasterUsername = params.MasterUsername
	dc.MasterPassword = params.MasterPassword

	if dc.MasterUsername == "" {
		dc.MasterUsername = generateUsername()
	}
	if dc.MasterPassword == "" {
		dc.MasterPassword = generatePassword()
	}

	return nil
}

type (
	DocumentDbInstance struct {
		Name            string
		ConstructorRefs core.AnnotationKeySet
		Cluster         *DocumentDbCluster
	}

	DocumentDbInstanceCreateParams struct {
		Name            string
		ConstructorRefs core.AnnotationKeySet
		Cluster         *DocumentDbCluster
	}
)

func (di *DocumentDbInstance) Id() core.ResourceId {
	//TODO implement me
	panic("implement me")
}

func (di *DocumentDbInstance) KlothoConstructRef() core.AnnotationKeySet {
	//TODO implement me
	panic("implement me")
}
