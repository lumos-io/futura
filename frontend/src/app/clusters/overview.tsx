import React, { useEffect, useState } from "react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import Cluster from "@/models/kubernetes";
import { useAuth } from "@/hooks/auth_provider";

const ClustersOverview: React.FC = () => {
  const { user } = useAuth();

  const [clusters, setClusters] = useState<Cluster[]>([]);
  const [newCluster, setNewCluster] = useState({ name: "", description: "" });

  // Fetch clusters from API
  useEffect(() => {
    const fetchClusters = async () => {
      const orgId = user?.organizationId;

      const res = await fetch(`/api/organizations/${orgId}/clusters`);
      const data = await res.json();
      setClusters(data.data);
    };
    fetchClusters();
  }, [user?.organizationId]);

  const handleAddCluster = () => {
    const addedCluster: Cluster = {
      id: clusters.length + 1,
      name: newCluster.name,
      description: newCluster.description,
    };
    setClusters([...clusters, addedCluster]);
    setNewCluster({ name: "", description: "" });
  };

  return (
    <div className="p-10 space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-3xl font-semibold text-gray-800">
          Clusters Overview
        </h1>
        <Dialog>
          <DialogTrigger asChild>
            <Button>Add Cluster</Button>
          </DialogTrigger>
          <DialogContent>
            <DialogHeader>
              <DialogTitle>Add New Cluster</DialogTitle>
            </DialogHeader>
            <div className="space-y-4">
              <Input
                placeholder="Cluster Name"
                value={newCluster.name}
                onChange={(e) => {
                  setNewCluster({ ...newCluster, name: e.target.value });
                }}
              />
              <Input
                placeholder="Description"
                value={newCluster.description}
                onChange={(e) => {
                  setNewCluster({ ...newCluster, description: e.target.value });
                }}
              />
            </div>
            <DialogFooter>
              <DialogClose asChild>
                <Button onClick={handleAddCluster}>Save</Button>
              </DialogClose>
            </DialogFooter>
          </DialogContent>
        </Dialog>
      </div>

      {clusters.length === 0 ? (
        <p className="text-gray-500 text-center">No cluster connected.</p>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          {clusters.map((cluster) => (
            <Card key={cluster.id}>
              <CardHeader>
                <CardTitle>{cluster.name}</CardTitle>
              </CardHeader>
              <CardContent>
                <p className="text-gray-600">{cluster.description}</p>
              </CardContent>
            </Card>
          ))}
        </div>
      )}
    </div>
  );
};

export default ClustersOverview;
