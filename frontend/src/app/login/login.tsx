import { GalleryVerticalEnd } from "lucide-react";
import { Button } from "@/components/ui/button";
import Placeholder from "@/assets/placeholder.svg";
import { useAuth } from "@/hooks/auth_provider";

export default function LoginPage() {
  const { login } = useAuth();

  return (
    <div className="grid min-h-svh lg:grid-cols-2">
      <div className="flex flex-col gap-4 p-6 md:p-10">
        <div className="flex justify-center gap-2 md:justify-start">
          <a href="#" className="flex items-center gap-2 font-medium">
            <div className="bg-primary text-primary-foreground flex size-6 items-center justify-center rounded-md">
              <GalleryVerticalEnd className="size-4" />
            </div>
            Futura
          </a>
        </div>
        <div className="flex flex-1 items-center justify-center">
          <div className="w-full max-w-xs">
            <div className="grid gap-6">
              <Button
                variant="outline"
                className="w-full"
                onClick={() => {
                  login("github");
                }}
              >
                <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24">
                  <path
                    d="M12 .297c-6.63 0-12 5.373-12 12 0 5.303 3.438 9.8 8.205 11.385.6.113.82-.258.82-.577 0-.285-.01-1.04-.015-2.04-3.338.724-4.042-1.61-4.042-1.61C4.422 
              18.07 3.633 17.7 3.633 17.7c-1.087-.744.084-.729.084-.729 1.205.084 1.838 1.236 1.838 1.236 1.07 1.835 2.809 1.305 
              3.495.998.108-.776.417-1.305.76-1.605-2.665-.3-5.466-1.332-5.466-5.93 0-1.31.465-2.38 1.235-3.22-.135-.303-.54-1.523.105-3.176 0 0 
              1.005-.322 3.3 1.23.96-.267 1.98-.399 3-.405 1.02.006 2.04.138 3 .405 2.28-1.552 3.285-1.23 3.285-1.23.645 1.653.24 2.873.12 3.176.765.84 
              1.23 1.91 1.23 3.22 0 4.61-2.805 5.625-5.475 5.92.42.36.81 1.096.81 2.22 0 1.606-.015 2.896-.015 3.286 0 .315.21.69.825.57C20.565 22.092 
              24 17.592 24 12.297c0-6.627-5.373-12-12-12"
                    fill="currentColor"
                  />
                </svg>
                Continue with GitHub
              </Button>
              <Button
                variant="outline"
                className="w-full"
                onClick={() => {
                  login("google");
                }}
              >
                <svg
                  xmlns="http://www.w3.org/2000/svg"
                  viewBox="0 0 48 48"
                  className="mr-2 h-5 w-5"
                >
                  <path
                    fill="#EA4335"
                    d="M24 9.5c3.14 0 5.82 1.08 7.97 2.84l5.96-5.96C33.04 2.41 28.84.5 24 .5 14.92.5 7.48 6.44 4.44 14.09l7.22 5.61C13.17 14.57 18.19 9.5 24 9.5z"
                  />
                  <path
                    fill="#34A853"
                    d="M24 44.5c6.12 0 11.26-2.02 15.02-5.49l-7.19-5.88c-2.08 1.4-4.75 2.22-7.83 2.22-5.78 0-10.7-3.91-12.46-9.17l-7.2 5.56c2.99 7.83 10.62 13.76 19.66 13.76z"
                  />
                  <path
                    fill="#4A90E2"
                    d="M44.5 24.5c0-1.38-.11-2.71-.32-4H24v7.58h11.54c-.5 2.58-2.08 4.77-4.38 6.23l7.19 5.88C42.77 36.8 44.5 30.94 44.5 24.5z"
                  />
                  <path
                    fill="#FBBC05"
                    d="M11.54 27.68A14.91 14.91 0 0 1 11 24c0-1.28.17-2.52.48-3.71l-7.2-5.56A24.17 24.17 0 0 0 0 24c0 3.77.89 7.34 2.48 10.51l9.06-6.83z"
                  />
                </svg>
                Continue with Google
              </Button>
            </div>
          </div>
        </div>
      </div>
      <div className="bg-muted relative hidden lg:block">
        <img
          src={Placeholder}
          alt="Image"
          className="absolute inset-0 h-full w-full object-cover dark:brightness-[0.2] dark:grayscale"
        />
      </div>
    </div>
  );
}
