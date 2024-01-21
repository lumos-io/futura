package kubernetes

type Pod struct {
	UID       string // Pod UID
	Name      string // Pod Name
	Namespace string // Namespace
	Image     string // Main container image
	IP        string // Pod IP
	OwnerType string // ReplicaSet or nil
	OwnerID   string // ReplicaSet UID
	OwnerName string // ReplicaSet Name
}

type Service struct {
	UID        string
	Name       string
	Namespace  string
	Type       string
	ClusterIP  string
	ClusterIPs []string
	Ports      []struct {
		Src      int32  `json:"src"`
		Dest     int32  `json:"dest"`
		Protocol string `json:"protocol"`
	}
}

type ReplicaSet struct {
	UID       string // ReplicaSet UID
	Name      string // ReplicaSet Name
	Namespace string // Namespace
	OwnerType string // Deployment or nil
	OwnerID   string // Deployment UID
	OwnerName string // Deployment Name
	Replicas  int32  // Number of replicas
}

type DaemonSet struct {
	UID       string // ReplicaSet UID
	Name      string // ReplicaSet Name
	Namespace string // Namespace
}

type Deployment struct {
	UID       string // Deployment UID
	Name      string // Deployment Name
	Namespace string // Namespace
	Replicas  int32  // Number of replicas
}

type Endpoints struct {
	UID       string // Endpoints UID
	Name      string // Endpoints Name
	Namespace string // Namespace
	Addresses []Address
}

type AddressIP struct {
	Type      string `json:"type"` // pod or external
	ID        string `json:"id"`   // Pod UID or empty
	Name      string `json:"name"`
	Namespace string `json:"namespace"` // Pod Namespace or empty
	IP        string `json:"ip"`        // Pod IP or external IP
}

type AddressPort struct {
	Port     int32  `json:"port"`     // Port number
	Protocol string `json:"protocol"` // TCP or UDP
}

// Subsets
type Address struct {
	IPs   []AddressIP   `json:"ips"`
	Ports []AddressPort `json:"ports"`
}

type Container struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	PodUID    string `json:"pod"` // Pod UID
	Image     string `json:"image"`
	Ports     []struct {
		Port     int32  `json:"port"`
		Protocol string `json:"protocol"`
	} `json:"ports"`
}

type PodEvent struct {
	UID       string `json:"uid"`
	EventType string `json:"event_type"`
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	IP        string `json:"ip"`
	OwnerType string `json:"owner_type"`
	OwnerName string `json:"owner_name"`
	OwnerID   string `json:"owner_id"`
}

type SvcEvent struct {
	UID        string   `json:"uid"`
	EventType  string   `json:"event_type"`
	Name       string   `json:"name"`
	Namespace  string   `json:"namespace"`
	Type       string   `json:"type"`
	ClusterIPs []string `json:"cluster_ips"`
	Ports      []struct {
		Src      int32  `json:"src"`
		Dest     int32  `json:"dest"`
		Protocol string `json:"protocol"`
	} `json:"ports"`
}

type RsEvent struct {
	UID       string `json:"uid"`
	EventType string `json:"event_type"`
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Replicas  int32  `json:"replicas"`
	OwnerType string `json:"owner_type"`
	OwnerName string `json:"owner_name"`
	OwnerID   string `json:"owner_id"`
}

type DsEvent struct {
	UID       string `json:"uid"`
	EventType string `json:"event_type"`
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

type DepEvent struct {
	UID       string `json:"uid"`
	EventType string `json:"event_type"`
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Replicas  int32  `json:"replicas"`
}

type EpEvent struct {
	UID       string    `json:"uid"`
	EventType string    `json:"event_type"`
	Name      string    `json:"name"`
	Namespace string    `json:"namespace"`
	Addresses []Address `json:"addresses"`
}

type ContainerEvent struct {
	UID       string `json:"uid"`
	EventType string `json:"event_type"`
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Pod       string `json:"pod"`
	Image     string `json:"image"`
	Ports     []struct {
		Port     int32  `json:"port"`
		Protocol string `json:"protocol"`
	} `json:"ports"`
}

func convertPodToPodEvent(pod Pod, eventType string) PodEvent {
	return PodEvent{
		UID:       pod.UID,
		EventType: eventType,
		Name:      pod.Name,
		Namespace: pod.Namespace,
		IP:        pod.IP,
		OwnerType: pod.OwnerType,
		OwnerName: pod.OwnerName,
		OwnerID:   pod.OwnerID,
	}
}

func convertSvcToSvcEvent(service Service, eventType string) SvcEvent {
	return SvcEvent{
		UID:        service.UID,
		EventType:  eventType,
		Name:       service.Name,
		Namespace:  service.Namespace,
		Type:       service.Type,
		ClusterIPs: service.ClusterIPs,
		Ports:      service.Ports,
	}
}

func convertRsToRsEvent(rs ReplicaSet, eventType string) RsEvent {
	return RsEvent{
		UID:       rs.UID,
		EventType: eventType,
		Name:      rs.Name,
		Namespace: rs.Namespace,
		Replicas:  rs.Replicas,
		OwnerType: rs.OwnerType,
		OwnerName: rs.OwnerName,
		OwnerID:   rs.OwnerID,
	}
}

func convertDsToDsEvent(ds DaemonSet, eventType string) DsEvent {
	return DsEvent{
		UID:       ds.UID,
		EventType: eventType,
		Name:      ds.Name,
		Namespace: ds.Namespace,
	}
}

func convertDepToDepEvent(d Deployment, eventType string) DepEvent {
	return DepEvent{
		UID:       d.UID,
		EventType: eventType,
		Name:      d.Name,
		Namespace: d.Namespace,
		Replicas:  d.Replicas,
	}
}

func convertEpToEpEvent(ep Endpoints, eventType string) EpEvent {
	return EpEvent{
		UID:       ep.UID,
		EventType: eventType,
		Name:      ep.Name,
		Namespace: ep.Namespace,
		Addresses: ep.Addresses,
	}
}

func convertContainerToContainerEvent(c Container, eventType string) ContainerEvent {
	return ContainerEvent{
		EventType: eventType,
		Name:      c.Name,
		Namespace: c.Namespace,
		Pod:       c.PodUID,
		Image:     c.Image,
		Ports:     c.Ports,
	}
}
