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
	ports := make([]*pb.AddressPort, len(service.Ports))
	for _, p := range service.Ports {
		ports = append(ports, &pb.AddressPort{
			Name:     p.Name,
			Port:     p.Dest,
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
		Uid:       ds.UID,
		EventType: eventType,
		Name:      ds.Name,
		Namespace: ds.Namespace,
	}
}

func ConvertSsToSsEvent(ss StatefulSet, eventType string) *pb.SsEvent {
	return &pb.SsEvent{
		Uid:       ss.UID,
		EventType: eventType,
		Name:      ss.Name,
		Namespace: ss.Namespace,
	}
}

func ConvertDepToDepEvent(d Deployment, eventType string) *pb.DepEvent {
	return &pb.DepEvent{
		Uid:       d.UID,
		EventType: eventType,
		Name:      d.Name,
		Namespace: d.Namespace,
		Replicas:  d.Replicas,
	}
}

func ConvertEpToEpEvent(ep Endpoints, eventType string) *pb.EpEvent {
	addresses := make([]*pb.Address, len(ep.Addresses))
	for _, a := range ep.Addresses {
		ips := make([]*pb.AddressIP, len(a.IPs))
		for _, ip := range a.IPs {
			ips = append(ips, &pb.AddressIP{
				Id:        ip.ID,
				Ip:        ip.IP,
				Type:      ip.Type,
				Name:      ip.Name,
				Namespace: ip.Namespace,
			})
		}
		ports := make([]*pb.AddressPort, len(a.Ports))
		for _, port := range a.Ports {
			ports = append(ports, &pb.AddressPort{
				Port:     port.Port,
				Protocol: port.Protocol,
				Name:     port.Name,
			})
		}
		addresses = append(addresses, &pb.Address{
			Ips:   ips,
			Ports: ports,
		})
	}
	return &pb.EpEvent{
		Uid:       ep.UID,
		EventType: eventType,
		Name:      ep.Name,
		Namespace: ep.Namespace,
		Addresses: addresses,
	}
}

func ConvertContainerToContainerEvent(c Container, eventType string) *pb.ContainerEvent {
	ports := make([]*pb.ContainerPort, len(c.Ports))
	for _, p := range c.Ports {
		ports = append(ports, &pb.ContainerPort{
			Port: &pb.AddressPort{
				Port:     p.Port,
				Protocol: p.Protocol,
				Name:     p.Name,
			},
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
