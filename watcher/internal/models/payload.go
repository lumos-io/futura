package models

import (
	pb "github.com/opisvigilant/futura/proto/events/gen"
)

type HealthCheckPayload struct {
	// Metadata Metadata `json:"metadata"`
	Info struct {
		MetricsEnabled bool `json:"metrics"`
	} `json:"watcher_info"`
	Telemetry struct {
		KernelVersion string `json:"kernel_version"`
		K8sVersion    string `json:"k8s_version"`
		CloudProvider string `json:"cloud_provider"`
	} `json:"telemetry"`
}

func ConvertPodToPodEvent(pod Pod, eventType string) *pb.PodEvent {
	return &pb.PodEvent{
		Uid:       pod.UID,
		EventType: eventType,
		Name:      pod.Name,
		Namespace: pod.Namespace,
		Ip:        pod.IP,
		OwnerType: pod.OwnerType,
		OwnerName: pod.OwnerName,
		OwnerId:   pod.OwnerID,
	}
}

func ConvertSvcToSvcEvent(service Service, eventType string) *pb.SvcEvent {
	ports := make([]*pb.Port, len(service.Ports))
	for _, p := range service.Ports {
		ports = append(ports, &pb.Port{
			Name:     p.Name,
			Src:      p.Src,
			Dest:     p.Dest,
			Protocol: p.Protocol,
		})
	}
	return &pb.SvcEvent{
		Uid:        service.UID,
		ClusterIps: service.ClusterIPs,
		EventType:  eventType,
		Name:       service.Name,
		Namespace:  service.Namespace,
		Type:       service.Type,
		Ports:      ports,
	}
}

func ConvertRsToRsEvent(rs ReplicaSet, eventType string) *pb.RsEvent {
	return &pb.RsEvent{
		Uid:       rs.UID,
		EventType: eventType,
		Name:      rs.Name,
		Namespace: rs.Namespace,
		Replicas:  rs.Replicas,
		OwnerType: rs.OwnerType,
		OwnerName: rs.OwnerName,
		OwnerId:   rs.OwnerID,
	}
}

func ConvertDsToDsEvent(ds DaemonSet, eventType string) *pb.DsEvent {
	return &pb.DsEvent{
		UID:       ds.UID,
		EventType: eventType,
		Name:      ds.Name,
		Namespace: ds.Namespace,
	}
}

func ConvertSsToSsEvent(ss StatefulSet, eventType string) *pb.SsEvent {
	return &pb.SsEvent{
		UID:       ss.UID,
		EventType: eventType,
		Name:      ss.Name,
		Namespace: ss.Namespace,
	}
}

func ConvertDepToDepEvent(d Deployment, eventType string) *pb.DepEvent {
	return &pb.DepEvent{
		UID:       d.UID,
		EventType: eventType,
		Name:      d.Name,
		Namespace: d.Namespace,
		Replicas:  d.Replicas,
	}
}

func ConvertEpToEpEvent(ep Endpoints, eventType string) *pb.EpEvent {
	addresses := make([]*pb.Address, len(c.Ports))
	for _, a := range ep.Addresses {
		addr := &pb.Address{}
		for _, ip := range a.IPs {
			addr.
		}
		addresses = append(addresses)
	}
	return &pb.EpEvent{
		UID:       ep.UID,
		EventType: eventType,
		Name:      ep.Name,
		Namespace: ep.Namespace,
		Addresses: ep.Addresses,
	}
}

func ConvertContainerToContainerEvent(c Container, eventType string) *pb.ContainerEvent {
	ports := make([]*pb.ContainerPort, len(c.Ports))
	for _, p := range c.Ports {
		ports = append(ports, &pb.ContainerPort{
			Port:     p.Port,
			Protocol: p.Protocol,
		})
	}
	return &pb.ContainerEvent{
		EventType: eventType,
		Name:      c.Name,
		Namespace: c.Namespace,
		Pod:       c.PodUID,
		Image:     c.Image,
		Ports:     ports,
	}
}
