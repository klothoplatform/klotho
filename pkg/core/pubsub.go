package core

type (
	PubSub struct {
		Path   string
		Name   string
		Events map[string]*Event
	}

	Event struct {
		Name        string
		Publishers  []ResourceKey
		Subscribers []ResourceKey
	}

	EventReference struct {
		ResourceKey
		FilePath string
	}
)

const PubSubKind = "pubsub"

func (p PubSub) Key() ResourceKey {
	return ResourceKey{
		Name: p.Name,
		Kind: PubSubKind,
	}
}

func (p *PubSub) AddPublisher(event string, key ResourceKey) {
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

func (p *PubSub) AddSubscriber(event string, key ResourceKey) {
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
