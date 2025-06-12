package kubernetes

func getOnAddStatefulSetFunc(ch chan any) func(any) {
	return func(obj any) {
		ch <- ResourceMessage{
			ResourceType: STATEFULSET,
			EventType:    ADD,
			Object:       obj,
		}
	}
}

func getOnUpdateStatefulSetFunc(ch chan any) func(any, any) {
	return func(oldObj, newObj any) {
		ch <- ResourceMessage{
			ResourceType: STATEFULSET,
			EventType:    UPDATE,
			Object:       newObj,
		}
	}
}

func getOnDeleteStatefulSetFunc(ch chan any) func(any) {
	return func(obj any) {
		ch <- ResourceMessage{
			ResourceType: STATEFULSET,
			EventType:    DELETE,
			Object:       obj,
		}
	}
}
