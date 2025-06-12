package kubernetes

func getOnAddEndpointsSetFunc(ch chan any) func(any) {
	return func(obj any) {
		ch <- ResourceMessage{
			ResourceType: ENDPOINTS,
			EventType:    ADD,
			Object:       obj,
		}
	}
}

func getOnUpdateEndpointsSetFunc(ch chan any) func(any, any) {
	return func(oldObj, newObj any) {
		ch <- ResourceMessage{
			ResourceType: ENDPOINTS,
			EventType:    UPDATE,
			Object:       newObj,
		}
	}
}

func getOnDeleteEndpointsSetFunc(ch chan any) func(any) {
	return func(obj any) {
		ch <- ResourceMessage{
			ResourceType: ENDPOINTS,
			EventType:    DELETE,
			Object:       obj,
		}
	}
}
