// components/ProviderCard.tsx
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  AlertDialog,
  AlertDialogTrigger,
  AlertDialogContent,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogFooter,
  AlertDialogCancel,
  AlertDialogAction,
} from "@/components/ui/alert-dialog";
import { Button } from "@/components/ui/button";
import { Tooltip, TooltipContent } from "@/components/ui/tooltip";
import { TooltipTrigger } from "@radix-ui/react-tooltip";
import { Cloud, CloudRain, CloudSun, Ghost, Trash2, Zap } from "lucide-react";
import {
  ActivationStatus,
  activationStatusFromJSON,
  CloudProvider,
  cloudProviderFromJSON,
} from "@proto/backend/backend";
import { Badge } from "@/components/ui/badge";
import { CloudProviderConnection } from "@/models/cloud-provider";

const CloudIcon = ({ provider }: { provider: CloudProvider }) => {
  switch (cloudProviderFromJSON(provider)) {
    case CloudProvider.AWS:
      return <Cloud className="w-5 h-5 text-yellow-500" />;
    case CloudProvider.GCP:
      return <CloudSun className="w-5 h-5 text-blue-500" />;
    case CloudProvider.AZURE:
      return <Zap className="w-5 h-5 text-blue-700" />;
    case CloudProvider.DIGITALOCEAN:
      return <Cloud className="w-5 h-5 text-indigo-500" />;
    case CloudProvider.ALIBABA:
      return <CloudRain className="w-5 h-5 text-orange-500" />;
    case CloudProvider.KIND:
      return <Ghost className="w-5 h-5 text-purple-500" />;
    default:
      return null;
  }
};

const RenderActivationStatus = ({ status }: { status: ActivationStatus }) => {
  const value = activationStatusFromJSON(status);
  switch (value) {
    case ActivationStatus.ACTIVE:
      return (
        <Badge
          className={`text-xs h-5 px-2 rounded-full bg-green-500 text-white`}
        >
          {value}
        </Badge>
      );
    case ActivationStatus.FAILED:
      return (
        <Badge
          className={`text-xs h-5 px-2 rounded-full bg-red-500 text-white`}
        >
          {value}
        </Badge>
      );
    case ActivationStatus.IN_PROGRESS:
      return (
        <Badge
          className={`text-xs h-5 px-2 rounded-full bg-blue-500 text-white`}
        >
          {value}
        </Badge>
      );
    case ActivationStatus.PENDING:
      return (
        <Badge
          className={`text-xs h-5 px-2 rounded-full bg-gray-500 text-white`}
        >
          {value}
        </Badge>
      );
    case ActivationStatus.SUSPENDED:
      return (
        <Badge
          className={`text-xs h-5 px-2 rounded-full bg-orange-500 text-white`}
        >
          {value}
        </Badge>
      );
    default:
      console.log("im here");
      return null;
  }
};

export const CloudProviderCard = ({
  provider,
  handleDelete,
  disableDelete,
}: {
  provider: CloudProviderConnection;
  handleDelete: () => void;
  disableDelete: boolean;
}) => {
  return (
    <Card className="shadow-md hover:shadow-lg transition-shadow rounded-2xl border border-muted">
      <CardHeader className="pb-2 space-y-2">
        <div className="flex justify-between items-start">
          <div className="space-y-1">
            <div className="flex items-center gap-2">
              <CloudIcon provider={provider.provider} />
              <CardTitle className="text-lg">
                {cloudProviderFromJSON(provider.provider)}
              </CardTitle>
            </div>
            <CardDescription className="text-sm text-muted-foreground">
              <div>{provider.connection_name}</div>
              <div>{provider.secret_id}</div>
            </CardDescription>
          </div>

          <AlertDialog>
            <AlertDialogTrigger asChild>
              {disableDelete ? (
                <Tooltip>
                  <TooltipTrigger className="cursor-not-allowed">
                    <Button variant="ghost" size="icon" disabled>
                      <Trash2 className="w-4 h-4 text-red-600" />
                    </Button>
                  </TooltipTrigger>
                  <TooltipContent>
                    <p>Fetching clusters metadata in progress...</p>
                  </TooltipContent>
                </Tooltip>
              ) : (
                <Button variant="ghost" size="icon" onClick={handleDelete}>
                  <Trash2 className="w-4 h-4 text-red-600" />
                </Button>
              )}
            </AlertDialogTrigger>
            <AlertDialogContent>
              <AlertDialogHeader>
                <AlertDialogTitle>
                  Are you sure you want to delete this provider?
                </AlertDialogTitle>
              </AlertDialogHeader>
              <AlertDialogFooter>
                <AlertDialogCancel>Cancel</AlertDialogCancel>
                <AlertDialogAction onClick={handleDelete}>
                  Delete
                </AlertDialogAction>
              </AlertDialogFooter>
            </AlertDialogContent>
          </AlertDialog>
        </div>
      </CardHeader>

      <CardContent className="text-sm text-gray-700 space-y-3">
        <div className="flex items-center justify-between">
          <span className="text-muted-foreground font-semibold">Status</span>
          <RenderActivationStatus status={provider.status} />
        </div>

        {provider.imported_clusters && (
          <div className="flex items-center justify-between">
            <span className="text-muted-foreground font-semibold">
              Clusters available
            </span>
            {provider.imported_clusters}
          </div>
        )}
      </CardContent>
    </Card>
  );
};
