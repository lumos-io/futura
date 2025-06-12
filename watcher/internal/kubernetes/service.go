package kubernetes

func getOnAddServiceFunc(ch chan any) func(any) {
	return func(obj any) {
		ch <- ResourceMessage{
			ResourceType: SERVICE,
			EventType:    ADD,
			Object:       obj,
		}
	}
}

func getOnUpdateServiceFunc(ch chan any) func(any, any) {
	return func(oldObj, newObj any) {
		ch <- ResourceMessage{
			ResourceType: SERVICE,
			EventType:    UPDATE,
			Object:       newObj,
		}
	}
}

func getOnDeleteServiceFunc(ch chan any) func(any) {
	return func(obj any) {
		ch <- ResourceMessage{
			ResourceType: SERVICE,
			EventType:    DELETE,
			Object:       obj,
		}
	}
}
