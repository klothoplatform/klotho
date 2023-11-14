package iac3

import (
	"errors"

	construct "github.com/klothoplatform/klotho/pkg/construct2"
)

func (tc *TemplatesCompiler) AddExtraResources(r construct.ResourceId) error {
	switch r.QualifiedTypeName() {
	case "aws:eks_cluster":
		err := addKubernetesProvider(tc.graph, r)
		err = errors.Join(err, addIngressRuleToCluster(tc.graph, r))
		return err

	case "aws:public_subnet", "aws:private_subnet":
		return addRouteTableAssociation(tc.graph, r)

	case "aws:target_group":
		return addTargetGroupAttachment(tc.graph, r)
	}

	return nil
}

func addKubernetesProvider(g construct.Graph, cluster construct.ResourceId) error {
	clusterRes, err := g.Vertex(cluster)
	if err != nil {
		return err
	}
	kubeConfig, ok := clusterRes.Properties["KubeConfig"].(construct.ResourceId)
	if !ok {
		return errors.New("cluster must have KubeConfig property")
	}

	gb := construct.NewGraphBatch(g)
	provider := &construct.Resource{
		ID: construct.ResourceId{
			Provider: "pulumi",
			Type:     "kubernetes_provider",
			Name:     cluster.Name,
		},
		Properties: construct.Properties{
			"KubeConfig": kubeConfig,
		},
	}
	gb.AddVertices(provider)
	gb.AddEdges(construct.Edge{Source: provider.ID, Target: kubeConfig})

	downstream, err := construct.DirectUpstreamDependencies(g, cluster)
	if err != nil {
		return err
	}
	for _, dep := range downstream {
		depR, _ := g.Vertex(dep)

		for name, prop := range depR.Properties {
			if prop == cluster {
				depR.Properties[name+"Provider"] = provider.ID
				gb.AddEdges(construct.Edge{Source: dep, Target: provider.ID})
			}
		}
	}
	return gb.Err
}

// addIngressRuleToCluster TODO move this into engine
func addIngressRuleToCluster(g construct.Graph, cluster construct.ResourceId) error {
	clusterRes, err := g.Vertex(cluster)
	if err != nil {
		return err
	}
	subnets, ok := clusterRes.Properties["Subnets"].([]construct.ResourceId)
	if !ok {
		return errors.New("cluster must have Subnets property")
	}

	cidrBlocks := make([]construct.PropertyRef, len(subnets))
	for i, subnet := range subnets {
		cidrBlocks[i] = construct.PropertyRef{
			Resource: subnet,
			Property: "cidr_block",
		}
	}

	sgRule := &construct.Resource{
		ID: construct.ResourceId{
			Provider:  "aws",
			Type:      "security_group_rule",
			Namespace: cluster.Name,
			Name:      "ingress",
		},
		Properties: construct.Properties{
			"Description": "Allows access to cluster from the VPCs private and public subnets",
			"FromPort":    0,
			"ToPort":      0,
			"Protocol":    "-1",
			"CidrBlocks":  cidrBlocks,
			"SecurityGroupId": construct.PropertyRef{
				Resource: cluster,
				Property: "cluster_security_group_id",
			},
			"Type": "ingress",
		},
	}

	gb := construct.NewGraphBatch(g)
	gb.AddVertices(sgRule)
	gb.AddEdges(construct.Edge{Source: sgRule.ID, Target: cluster})
	return gb.Err
}

// addRouteTableAssociation TODO move this into engine
func addRouteTableAssociation(g construct.Graph, subnet construct.ResourceId) error {
	upstream, err := construct.DirectUpstreamDependencies(g, subnet)
	if err != nil {
		return err
	}
	gb := construct.NewGraphBatch(g)
	for _, routeTable := range upstream {
		if routeTable.QualifiedTypeName() != "aws:route_table" {
			continue
		}

		association := &construct.Resource{
			ID: construct.ResourceId{
				Provider:  "aws",
				Type:      "route_table_association",
				Namespace: subnet.Name,
				Name:      "association",
			},
			Properties: construct.Properties{
				"Subnet":     subnet,
				"RouteTable": routeTable,
			},
		}
		gb.AddVertices(association)
		gb.AddEdges(
			construct.Edge{Source: association.ID, Target: subnet},
			construct.Edge{Source: association.ID, Target: routeTable},
		)
	}

	return gb.Err
}

// addTargetGroupAttachment TODO move this into engine
func addTargetGroupAttachment(g construct.Graph, tg construct.ResourceId) error {
	tgRes, err := g.Vertex(tg)
	if err != nil {
		return err
	}
	targets, ok := tgRes.Properties["Targets"].([]construct.ResourceId)
	if !ok {
		return errors.New("target group must have Targets property")
	}

	gb := construct.NewGraphBatch(g)
	for _, target := range targets {
		attachment := &construct.Resource{
			ID: construct.ResourceId{
				Provider:  "aws",
				Type:      "target_group_attachment",
				Namespace: tg.Name,
				Name:      target.Name,
			},
			Properties: construct.Properties{
				"Port":           construct.PropertyRef{Resource: target, Property: "Port"},
				"TargetGroupArn": construct.PropertyRef{Resource: tg, Property: "Arn"},
				"TargetId":       construct.PropertyRef{Resource: target, Property: "Id"},
			},
		}
		gb.AddVertices(attachment)
		gb.AddEdges(
			construct.Edge{Source: attachment.ID, Target: tg},
			construct.Edge{Source: attachment.ID, Target: target},
		)
	}
	return gb.Err
}
