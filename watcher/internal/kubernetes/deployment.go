package kubernetes

func getOnAddDeploymentSetFunc(ch chan any) func(any) {
	return func(obj any) {
		ch <- ResourceMessage{
			ResourceType: DEPLOYMENT,
			EventType:    ADD,
			Object:       obj,
		}
	}
}

func getOnUpdateDeploymentSetFunc(ch chan any) func(any, any) {
	return func(oldObj, newObj any) {
		ch <- ResourceMessage{
			ResourceType: DEPLOYMENT,
			EventType:    UPDATE,
			Object:       newObj,
		}
	}
}

func getOnDeleteDeploymentSetFunc(ch chan any) func(any) {
	return func(obj any) {
		ch <- ResourceMessage{
			ResourceType: DEPLOYMENT,
			EventType:    DELETE,
			Object:       obj,
		}
	}
}
