package collector

import (
	"github.com/opisvigilant/futura/watcher/internal/kubernetes"
	"github.com/opisvigilant/futura/watcher/internal/logger"
	"github.com/opisvigilant/futura/watcher/internal/models"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	pb "github.com/opisvigilant/futura/proto/events/gen"
)

const (
	ADD    = "ADD"
	UPDATE = "UPDATE"
	DELETE = "DELETE"
)

func (c *Collector) persistPod(pod models.Pod, eventType string) {
	podEvent := models.ConvertPodToPodEvent(pod, eventType)
	c.sender.PodEventChan <- &pb.KubernetesEvent{
		Event: &pb.KubernetesEvent_Pod{
			Pod: podEvent,
		},
	}
}

func (a *Collector) processPod(d kubernetes.ResourceMessage) {
	pod := d.Object.(*corev1.Pod)

	var ownerType, ownerID, ownerName string
	if len(pod.OwnerReferences) > 0 {
		ownerType = pod.OwnerReferences[0].Kind
		ownerID = string(pod.OwnerReferences[0].UID)
		ownerName = pod.OwnerReferences[0].Name
	} else {
		logger.Logger().Debug().Msgf("Pod %s/%s has no owner, event: %s", pod.Namespace, pod.Name, d.EventType)
	}

	if pod.Status.PodIP == "" {
		logger.Logger().Debug().Msgf("Pod %s/%s has no IP, event: %s", pod.Namespace, pod.Name, d.EventType)
		return
	}

	dtoPod := models.Pod{
		UID:       string(pod.UID),
		Name:      pod.Name,
		Namespace: pod.Namespace,
		Image:     pod.Spec.Containers[0].Image, // main containers
		IP:        pod.Status.PodIP,

		// Assuming that there is only one owner
		OwnerType: ownerType,
		OwnerID:   ownerID,
		OwnerName: ownerName,
	}

	switch d.EventType {
	case kubernetes.ADD:
		go a.persistPod(dtoPod, ADD)
	case kubernetes.UPDATE:
		go a.persistPod(dtoPod, UPDATE)
	case kubernetes.DELETE:
		go a.persistPod(dtoPod, DELETE)
	}
}

func (a *Collector) persistSvc(service models.Service, eventType string) {
	svcEvent := models.ConvertSvcToSvcEvent(service, eventType)
	a.sender.ServiceEventChan <- &pb.KubernetesEvent{
		Event: &pb.KubernetesEvent_Svc{
			Svc: svcEvent,
		},
	}
}

func (a *Collector) processSvc(d kubernetes.ResourceMessage) {
	service := d.Object.(*corev1.Service)

	ports := []struct {
		Name     string "json:\"name\""
		Src      int32  "json:\"src\""
		Dest     int32  "json:\"dest\""
		Protocol string "json:\"protocol\""
	}{}

	for _, port := range service.Spec.Ports {
		ports = append(ports, struct {
			Name     string "json:\"name\""
			Src      int32  "json:\"src\""
			Dest     int32  "json:\"dest\""
			Protocol string "json:\"protocol\""
		}{
			Name:     port.Name, // https://kubernetes.io/docs/concepts/services-networking/service/#field-spec-ports
			Src:      port.Port,
			Dest:     int32(port.TargetPort.IntValue()),
			Protocol: string(port.Protocol),
		})
	}

	dtoSvc := models.Service{
		UID:        string(service.UID),
		Name:       service.Name,
		Namespace:  service.Namespace,
		Type:       string(service.Spec.Type),
		ClusterIPs: service.Spec.ClusterIPs,
		Ports:      ports,
	}

	switch d.EventType {
	case kubernetes.ADD:
		go a.persistSvc(dtoSvc, ADD)
	case kubernetes.UPDATE:
		go a.persistSvc(dtoSvc, UPDATE)
	case kubernetes.DELETE:
		go a.persistSvc(dtoSvc, DELETE)
	}
}

func (a *Collector) persistReplicaSet(rs models.ReplicaSet, eventType string) {
	rsEvent := models.ConvertRsToRsEvent(rs, eventType)
	a.sender.ReplicaSetEventChan <- &pb.KubernetesEvent{
		Event: &pb.KubernetesEvent_Rs{
			Rs: rsEvent,
		},
	}
}

func (a *Collector) processReplicaSet(d kubernetes.ResourceMessage) {
	replicaSet := d.Object.(*appsv1.ReplicaSet)

	var ownerType, ownerID, ownerName string
	if len(replicaSet.OwnerReferences) > 0 {
		ownerType = replicaSet.OwnerReferences[0].Kind
		ownerID = string(replicaSet.OwnerReferences[0].UID)
		ownerName = replicaSet.OwnerReferences[0].Name
	} else {
		logger.Logger().Debug().Msgf("ReplicaSet %s/%s has no owner, event: %s", replicaSet.Namespace, replicaSet.Name, d.EventType)
	}

	dtoReplicaSet := models.ReplicaSet{
		UID:       string(replicaSet.UID),
		Name:      ownerName,
		Namespace: replicaSet.Namespace,
		OwnerType: ownerType,
		OwnerID:   ownerID,
		OwnerName: ownerName,
		Replicas:  replicaSet.Status.Replicas,
	}

	switch d.EventType {
	case kubernetes.ADD:
		go a.persistReplicaSet(dtoReplicaSet, ADD)
	case kubernetes.UPDATE:
		go a.persistReplicaSet(dtoReplicaSet, UPDATE)
	case kubernetes.DELETE:
		go a.persistReplicaSet(dtoReplicaSet, DELETE)
	}

}

func (a *Collector) processDeployment(d kubernetes.ResourceMessage) {
	deployment := d.Object.(*appsv1.Deployment)

	dto := models.Deployment{
		UID:       string(deployment.UID),
		Name:      deployment.Name,
		Namespace: deployment.Namespace,
		Replicas:  deployment.Status.Replicas,
	}

	var depEvent *pb.DepEvent
	switch d.EventType {
	case kubernetes.ADD:
		depEvent = models.ConvertDepToDepEvent(dto, ADD)
	case kubernetes.UPDATE:
		depEvent = models.ConvertDepToDepEvent(dto, UPDATE)
	case kubernetes.DELETE:
		depEvent = models.ConvertDepToDepEvent(dto, DELETE)
	}
	a.sender.DeploymentEventChan <- &pb.KubernetesEvent{
		Event: &pb.KubernetesEvent_Dep{
			Dep: depEvent,
		},
	}
}

func (a *Collector) processContainer(d kubernetes.ResourceMessage) {
	c := d.Object.(*kubernetes.Container)

	ports := make([]models.AddressPort, len(c.Ports))
	for _, port := range c.Ports {
		ports = append(ports, models.AddressPort{
			Port:     port.Port,
			Protocol: port.Protocol,
			Name:     port.Name,
		})
	}

	dto := models.Container{
		Name:      c.Name,
		Namespace: c.Namespace,
		PodUID:    c.PodUID,
		Image:     c.Image,
		Ports:     ports,
	}

	var cEvent *pb.ContainerEvent
	switch d.EventType {
	case kubernetes.ADD:
		cEvent = models.ConvertContainerToContainerEvent(dto, ADD)
	case kubernetes.UPDATE:
		cEvent = models.ConvertContainerToContainerEvent(dto, UPDATE)
	}
	a.sender.ContainerEventChan <- &pb.KubernetesEvent{
		Event: &pb.KubernetesEvent_Container{
			Container: cEvent,
		},
	}
}

func (a *Collector) processEndpoints(ep kubernetes.ResourceMessage) {
	endpoints := ep.Object.(*corev1.Endpoints)

	// subsets
	adrs := []models.Address{}

	// subset[0].address -> ips
	// subset[0].ports -> ports

	for _, subset := range endpoints.Subsets {
		ips := []models.AddressIP{}
		ports := []models.AddressPort{}

		for _, addr := range subset.Addresses {
			// Probably external IP
			if addr.TargetRef == nil {
				ips = append(ips, models.AddressIP{
					IP: addr.IP,
				})
				continue
			}

			// TargetRef: Pod probably
			ips = append(ips, models.AddressIP{
				Type:      string(addr.TargetRef.Kind),
				ID:        string(addr.TargetRef.UID),
				Name:      addr.TargetRef.Name,
				Namespace: addr.TargetRef.Namespace,
				IP:        addr.IP,
			})
		}

		for _, port := range subset.Ports {
			ports = append(ports, models.AddressPort{
				Port:     port.Port,
				Protocol: string(port.Protocol),
				Name:     port.Name,
			})
		}

		adrs = append(adrs, models.Address{
			IPs:   ips,
			Ports: ports,
		})
	}

	dto := models.Endpoints{
		UID:       string(endpoints.UID),
		Name:      endpoints.Name,
		Namespace: endpoints.Namespace,
		Addresses: adrs,
	}

	var epEvent *pb.EpEvent
	switch ep.EventType {
	case kubernetes.ADD:
		epEvent = models.ConvertEpToEpEvent(dto, ADD)
	case kubernetes.UPDATE:
		epEvent = models.ConvertEpToEpEvent(dto, UPDATE)
	case kubernetes.DELETE:
		epEvent = models.ConvertEpToEpEvent(dto, DELETE)
	}
	a.sender.EndpointEventChan <- &pb.KubernetesEvent{
		Event: &pb.KubernetesEvent_Ep{
			Ep: epEvent,
		},
	}
}

func (a *Collector) processDaemonSet(d kubernetes.ResourceMessage) {
	daemonSet := d.Object.(*appsv1.DaemonSet)

	dtoDaemonSet := models.DaemonSet{
		UID:       string(daemonSet.UID),
		Name:      daemonSet.Name,
		Namespace: daemonSet.Namespace,
	}

	var dsEvent *pb.DsEvent
	switch d.EventType {
	case kubernetes.ADD:
		dsEvent = models.ConvertDsToDsEvent(dtoDaemonSet, ADD)
	case kubernetes.UPDATE:
		dsEvent = models.ConvertDsToDsEvent(dtoDaemonSet, UPDATE)
	case kubernetes.DELETE:
		dsEvent = models.ConvertDsToDsEvent(dtoDaemonSet, DELETE)
	}
	a.sender.DaemonSetEventChan <- &pb.KubernetesEvent{
		Event: &pb.KubernetesEvent_Ds{
			Ds: dsEvent,
		},
	}
}

func (a *Collector) processStatefulSet(d kubernetes.ResourceMessage) {
	statefulSet := d.Object.(*appsv1.StatefulSet)

	dtoStatefulSet := models.StatefulSet{
		UID:       string(statefulSet.UID),
		Name:      statefulSet.Name,
		Namespace: statefulSet.Namespace,
	}

	var ssEvent *pb.SsEvent
	switch d.EventType {
	case kubernetes.ADD:
		ssEvent = models.ConvertSsToSsEvent(dtoStatefulSet, ADD)
	case kubernetes.UPDATE:
		ssEvent = models.ConvertSsToSsEvent(dtoStatefulSet, UPDATE)
	case kubernetes.DELETE:
		ssEvent = models.ConvertSsToSsEvent(dtoStatefulSet, DELETE)
	}
	a.sender.StatefulSetEventChan <- &pb.KubernetesEvent{
		Event: &pb.KubernetesEvent_Ss{
			Ss: ssEvent,
		},
	}
}
