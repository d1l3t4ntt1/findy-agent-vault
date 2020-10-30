package faker

import (
	"fmt"
	"sort"

	"github.com/bxcodec/faker/v3"
	"github.com/findy-network/findy-agent-vault/tools/data"
	"github.com/lainio/err2"
)

func FakeEvents(count int) (events []data.InternalEvent, err error) {
	events = make([]data.InternalEvent, count)
	for i := 0; i < count; i++ {
		event := data.InternalEvent{}
		err2.Check(faker.FakeData(&event))
		events[i] = event
	}
	sort.Slice(events, func(i, j int) bool {
		return events[i].CreatedMs < events[j].CreatedMs
	})
	return
}

func fakeAndPrintEvents(
	count int,
	connections []data.InternalPairwise,
) {
	var err error
	defer err2.Annotate("fakeAndPrintEvents", &err)

	// Add connections to state so that events get a valid connection id
	for index := range connections {
		data.State.Connections.Append(&connections[index])
	}
	events, err := FakeEvents(count)

	fmt.Println("\nvar events = []InternalEvent{")
	for i := 0; i < len(events); i++ {
		fmt.Printf("	")
		printObject(&events[i], events[i], true)
	}
	fmt.Println("}")
}
