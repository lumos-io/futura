package kubernetes

func getOnAddReplicaSetFunc(ch chan any) func(any) {
	return func(obj any) {
		ch <- ResourceMessage{
			ResourceType: REPLICASET,
			EventType:    ADD,
			Object:       obj,
		}
	}
}

func getOnUpdateReplicaSetFunc(ch chan any) func(any, any) {
	return func(oldObj, newObj any) {
		ch <- ResourceMessage{
			ResourceType: REPLICASET,
			EventType:    UPDATE,
			Object:       newObj,
		}
	}
}

func getOnDeleteReplicaSetFunc(ch chan any) func(any) {
	return func(obj any) {
		ch <- ResourceMessage{
			ResourceType: REPLICASET,
			EventType:    DELETE,
			Object:       obj,
		}
	}
}
