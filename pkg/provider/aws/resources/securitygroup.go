package resources

import (
	"fmt"

	"github.com/klothoplatform/klotho/pkg/config"
	"github.com/klothoplatform/klotho/pkg/core"
)

type SecurityGroup struct {
	Name          string
	Vpc           *Vpc
	ConstructsRef []core.AnnotationKey
	// TODO Add ingress rules - https://github.com/klothoplatform/klotho/issues/465
}

const SG_TYPE = "security_group"

// GetSecurityGroup returns the security group if one exists, otherwise creates one, then returns it
func GetSecurityGroup(cfg *config.Application, dag *core.ResourceGraph) *SecurityGroup {
	for _, r := range dag.ListResources() {
		if sg, ok := r.(*SecurityGroup); ok {
			return sg
		}
	}
	sg := &SecurityGroup{
		Name: cfg.AppName,
		Vpc:  GetVpc(cfg, dag),
	}
	dag.AddResource(sg)
	dag.AddDependency2(sg, sg.Vpc)
	return sg
}

// Provider returns name of the provider the resource is correlated to
func (sg *SecurityGroup) Provider() string {
	return AWS_PROVIDER
}

// KlothoResource returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (sg *SecurityGroup) KlothoConstructRef() []core.AnnotationKey {
	return sg.ConstructsRef
}

// ID returns the id of the cloud resource
func (sg *SecurityGroup) Id() string {
	return fmt.Sprintf("%s:%s:%s", sg.Provider(), SG_TYPE, sg.Name)
}
