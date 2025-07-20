/*
Danger zone!

These values are used in the backend to access a map[string]string.
Before changing any of these values, make sure the backend is aware
otherwise the Temporal workflows will epically fail.

*/

export const AWS_ACCOUNT = "Account";
export const AWS_ACCESS_KEY = "AccessKey";
export const AWS_SECRET_ACCESS_KEY = "SecretAccessKey";
export const AWS_REGION = "Region";

export const GCP_PROJECT_ID = "ProjectId";
export const GCP_CREDENTIALS_JSON = "CredentialsJson";

export const AZURE_TENANT_ID = "TenantId";
export const AZURE_CLIENT_ID = "ClientId";
export const AZURE_CLIENT_SECRET = "ClientSecret";

export const DIGITALOCEAN_ACCESS_TOKEN = "AccessToken";

export const ALIBABA_ACCESS_KEY_ID = "AccessKeyId";
export const ALIBABA_ACCESS_SECRET = "AccessSecret";

export const PROVIDER_OPTIONS = ["ALIBABA", "AWS", "GCP", "DIGITALOCEAN", "AZURE"];