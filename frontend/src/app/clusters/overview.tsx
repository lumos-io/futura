import React, { useEffect, useState } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import Cluster from "@/models/kubernetes";
import { useAuth } from "@/hooks/auth_provider";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { cloudProviderToJSON } from "@proto/backend/backend";
import { CloudProviderConnection } from "@/models/cloud-provider";

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

  // Fetch connected providers on mount
  useEffect(() => {
    const fetchProviders = async () => {
      const orgId = user?.organizationId;
      if (!orgId) {
        return;
      }
      const res = await fetch(`/api/organizations/${orgId}/connects`);
      if (!res.ok) {
        return;
      }
      const data = await res.json();
      console.log(data);
      setProvidersConnection(data.data);
    };
    fetchProviders();
  }, [user?.organizationId]);

  useEffect(() => {
    const fetchClusters = async (provider: CloudProviderConnection) => {
      setLoading(true);
      setError(null);

      try {
        const orgId = user?.organizationId;
        // Adjust API URL accordingly; assuming it accepts provider param
        const res = await fetch(
          `/api/organizations/${orgId}/clusters?provider=${provider.id}`
        );
        if (!res.ok) {
          throw new Error(
            `Failed to fetch clusters for ${cloudProviderToJSON(
              provider.provider
            )}`
          );
        }
        const data = await res.json();
        console.log(data);
        setClusters(data.data ?? []);
      } catch (err: unknown) {
        console.error(err);
        setError("Unknown error");
        setClusters([]);
      } finally {
        setLoading(false);
      }
    };

    if (providersConnection.length > 0) {
      setSelectedProvider(providersConnection[0]);
      if (selectedProvider) {
        fetchClusters(selectedProvider);
      }
    }
  }, [providersConnection, selectedProvider, user?.organizationId]);

  return (
    <div className="p-10 space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-3xl font-semibold text-gray-800">
          Clusters Overview
        </h1>

        <div className="w-48">
          <Select
            value={
              selectedProvider
                ? cloudProviderToJSON(selectedProvider.provider)
                : ""
            }
            onValueChange={(value) => {
              // Find the full object by the string value
              const found = providersConnection.find(
                (p) => cloudProviderToJSON(p.provider) === value
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
                <SelectItem
                  key={cloudProviderToJSON(provider.provider)}
                  value={cloudProviderToJSON(provider.provider)}
                >
                  {cloudProviderToJSON(provider.provider)}
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
          {selectedProvider
            ? cloudProviderToJSON(selectedProvider.provider)
            : ""}
          .
        </p>
      )}

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
        {clusters.map((cluster) => (
          <Card key={cluster.id}>
            <CardHeader>
              <CardTitle>{cluster.name}</CardTitle>
            </CardHeader>
            <CardContent>
              <p className="text-gray-600">
                {cluster.description || "No description"}
              </p>
            </CardContent>
          </Card>
        ))}
      </div>
    </div>
  );
};

export default ClustersOverview;
