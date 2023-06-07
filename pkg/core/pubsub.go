package core

type (
	PubSub struct {
		AnnotationKey
		Path   string
		Events map[string]*Event
	}

	Event struct {
		Name        string
		Publishers  []AnnotationKey
		Subscribers []AnnotationKey
	}

	EventReference struct {
		AnnotationKey
		FilePath string
	}
)

const PubSubKind = "pubsub"

func (p *PubSub) Provenance() AnnotationKey {
	return p.AnnotationKey
}

func (p *PubSub) Id() ResourceId {
	return ConstructId(p.AnnotationKey).ToRid()
}

func (p *PubSub) AddPublisher(event string, key AnnotationKey) {
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

func (p *PubSub) AddSubscriber(event string, key AnnotationKey) {
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
