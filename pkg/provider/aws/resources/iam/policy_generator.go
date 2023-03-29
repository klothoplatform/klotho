package iam

import (
	"fmt"

	"github.com/klothoplatform/klotho/pkg/core"
)

type (
	PolicyGenerator struct {
		unitToRole    map[string]*IamRole
		unitsPolicies map[string]*PolicyDocument
	}

	PolicyDocument struct {
		Version   string
		Statement []StatementEntry `resource:"document"`
	}

	StatementEntry struct {
		Effect   string
		Action   []string
		Resource []core.IaCValue
	}
)

const (
	VERSION = "2012-10-17"
)

func NewPolicyGenerator() *PolicyGenerator {
	p := &PolicyGenerator{
		unitsPolicies: make(map[string]*PolicyDocument),
		unitToRole:    make(map[string]*IamRole),
	}
	return p
}

func (p *PolicyGenerator) AddAllowPolicyToUnit(unitId string, actions []string, resources []core.IaCValue) {
	policyDoc, found := p.unitsPolicies[unitId]
	if !found {
		p.unitsPolicies[unitId] = &PolicyDocument{
			Version: VERSION,
			Statement: []StatementEntry{
				{
					Effect:   "Allow",
					Action:   actions,
					Resource: resources,
				},
			},
		}
	} else {
		policyDoc.Statement = append(policyDoc.Statement, StatementEntry{
			Effect:   "Allow",
			Action:   actions,
			Resource: resources,
		})
	}
}

func (p *PolicyGenerator) AddUnitRole(unitId string, role *IamRole) error {
	_, found := p.unitToRole[unitId]
	if found {
		return fmt.Errorf("unit with id, %s, is already mapped to an IAM Role", unitId)
	}
	p.unitToRole[unitId] = role
	return nil
}

func (p *PolicyGenerator) GetUnitRole(unitId string) *IamRole {
	role := p.unitToRole[unitId]
	return role
}

func (p *PolicyGenerator) GetUnitPolicies(unitId string) *PolicyDocument {
	policyDoc := p.unitsPolicies[unitId]
	return policyDoc
}
