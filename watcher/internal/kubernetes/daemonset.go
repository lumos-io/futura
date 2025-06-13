package kubernetes

func getOnAddDaemonSetFunc(ch chan any) func(any) {
	return func(obj any) {
		ch <- ResourceMessage{
			ResourceType: DAEMONSET,
			EventType:    ADD,
			Object:       obj,
		}
	}
}

func getOnUpdateDaemonSetFunc(ch chan any) func(any, any) {
	return func(oldObj, newObj any) {
		ch <- ResourceMessage{
			ResourceType: DAEMONSET,
			EventType:    UPDATE,
			Object:       newObj,
		}
	}
}

func getOnDeleteDaemonSetFunc(ch chan any) func(any) {
	return func(obj any) {
		ch <- ResourceMessage{
			ResourceType: DAEMONSET,
			EventType:    DELETE,
			Object:       obj,
		}
	}
}
