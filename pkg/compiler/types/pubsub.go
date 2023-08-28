package types

import (
	"github.com/klothoplatform/klotho/pkg/annotation"
	"github.com/klothoplatform/klotho/pkg/construct"
)

type (
	PubSub struct {
		Name   string
		Path   string
		Events map[string]*Event
	}

	Event struct {
		Name        string
		Publishers  []construct.ResourceId
		Subscribers []construct.ResourceId
	}

	EventReference struct {
		AnnotationKey
		FilePath string
	}
)

const PUBSUB_TYPE = "pubsub"

func (p *PubSub) Id() construct.ResourceId {
	return construct.ResourceId{
		Provider: construct.AbstractConstructProvider,
		Type:     PUBSUB_TYPE,
		Name:     p.Name,
	}
}

func (p *PubSub) AnnotationCapability() string {
	return annotation.PubSubCapability
}

func (p *PubSub) Functionality() construct.Functionality {
	return construct.Unknown
}

func (p *PubSub) Attributes() map[string]any {
	return map[string]any{}
}

func (p *PubSub) AddPublisher(event string, key construct.ResourceId) {
	if p.Events == nil {
		p.Events = make(map[string]*Event)
	}
	e, ok := p.Events[event]
	if !ok {
		e = &Event{Name: event}
		p.Events[event] = e
	}
	e.Publishers = append(e.Publishers, key)
}

func (p *PubSub) AddSubscriber(event string, key construct.ResourceId) {
	if p.Events == nil {
		p.Events = make(map[string]*Event)
	}
	e, ok := p.Events[event]
	if !ok {
		e = &Event{Name: event}
		p.Events[event] = e
	}
	e.Subscribers = append(e.Subscribers, key)
}

func (p *PubSub) EventNames() (e []string) {
	for k := range p.Events {
		e = append(e, k)
	}
	return
}
