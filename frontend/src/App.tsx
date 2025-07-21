import "./App.css";
import { useRoutes } from "react-router-dom";
import { ThemeProvider } from "@/components/theme-provider";
import { Toaster } from "@/components/ui/sonner";
import allRoutes from "@/routes";

function App() {
  const routes = useRoutes(allRoutes);
  return (
    <ThemeProvider defaultTheme="light" storageKey="futura-ui-theme">
      {routes}
      <Toaster />
    </ThemeProvider>
  );
}

export default App;
