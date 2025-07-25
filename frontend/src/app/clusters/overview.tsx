import React, { useEffect, useState } from "react";
import Cluster from "@/models/kubernetes";
import { useAuth } from "@/hooks/auth-provider";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { CloudProviderConnection } from "@/models/cloud-provider";
import OverviewClusterCard from "./components/overview-card";

const ClustersOverview: React.FC = () => {
  const { user } = useAuth();

  const [providersConnection, setProvidersConnection] = useState<
    CloudProviderConnection[]
  >([]);
  const [selectedProvider, setSelectedProvider] =
    useState<CloudProviderConnection>();
  const [clusters, setClusters] = useState<Cluster[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const orgId = user?.organizationId;

  useEffect(() => {
    const fetchProviders = async () => {
      if (!orgId) {
        return;
      }

      const res = await fetch(`/api/organizations/${orgId}/connects`);
      if (!res.ok) {
        return;
      }

      const data = await res.json();
      setProvidersConnection(data.data);
    };

    fetchProviders();
  }, [orgId]);

  useEffect(() => {
    if (providersConnection.length > 0 && !selectedProvider) {
      setSelectedProvider(providersConnection[0]);
    }
  }, [providersConnection, selectedProvider]);

  useEffect(() => {
    if (!selectedProvider || !user?.organizationId) return;

    const fetchClusters = async () => {
      setLoading(true);
      setError(null);

      try {
        const orgId = user.organizationId;
        const res = await fetch(
          `/api/organizations/${orgId}/connects/${selectedProvider.id}/clusters`
        );
        if (!res.ok) {
          throw new Error("Failed to fetch clusters");
        }

        const data = await res.json();
        setClusters(data.data ?? []);
      } catch (err) {
        console.error(err);
        setError("Unknown error");
        setClusters([]);
      } finally {
        setLoading(false);
      }
    };

    fetchClusters();
  }, [selectedProvider, user?.organizationId]);

  return (
    <div className="p-10 space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-3xl font-semibold text-gray-800">
          Clusters Overview
        </h1>

        <div className="w-48">
          <Select
            value={selectedProvider ? selectedProvider.id.toString() : ""}
            onValueChange={(value) => {
              const found = providersConnection.find(
                (p) => p.id.toString() === value
              );
              if (found) {
                setSelectedProvider(found);
              }
            }}
          >
            <SelectTrigger>
              <SelectValue placeholder="Select Cloud Provider" />
            </SelectTrigger>
            <SelectContent>
              {providersConnection.map((provider) => (
                <SelectItem key={provider.id} value={provider.id.toString()}>
                  {provider.connection_name}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      </div>

      {loading && <p className="text-gray-500">Loading clusters...</p>}
      {error && <p className="text-red-500">{error}</p>}

      {clusters.length === 0 && !loading && !error && (
        <p className="text-gray-500 text-center">
          No clusters found for{" "}
          {selectedProvider ? selectedProvider.connection_name : ""}.
        </p>
      )}

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
        {clusters.map((cluster) => (
          <OverviewClusterCard provider={selectedProvider} cluster={cluster} />
        ))}
      </div>
    </div>
  );
};

export default ClustersOverview;
