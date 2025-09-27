import { Toaster } from "@/components/ui/toaster";
import { Toaster as Sonner } from "@/components/ui/sonner";
import { TooltipProvider } from "@/components/ui/tooltip";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { BrowserRouter, Routes, Route } from "react-router-dom";
import { Layout } from "./components/Layout";
import Pipelines from "./pages/Pipelines";
import PipelineNew from "./pages/PipelineNew";
import PipelineJobs from "./pages/PipelineJobs";
import Jobs from "./pages/Jobs";
import JobDetails from "./pages/JobDetails";
import History from "./pages/History";
import Analytics from "./pages/Analytics";
import Environments from "./pages/Environments";
import Settings from "./pages/Settings";
import NotFound from "./pages/NotFound";

const queryClient = new QueryClient();

const App = () => (
  <QueryClientProvider client={queryClient}>
    <TooltipProvider>
      <Toaster />
      <Sonner />
      <BrowserRouter>
        <Layout>
          <Routes>
            <Route path="/" element={<Pipelines />} />
            <Route path="/pipelines/new" element={<PipelineNew />} />
            <Route path="/pipelines/:id/edit" element={<PipelineNew />} />
            <Route path="/pipelines/:pipelineId/jobs" element={<PipelineJobs />} />
            <Route path="/jobs" element={<Jobs />} />
            <Route path="/jobs/:jobId" element={<JobDetails />} />
            <Route path="/history" element={<History />} />
            <Route path="/analytics" element={<Analytics />} />
            <Route path="/environments" element={<Environments />} />
            <Route path="/settings" element={<Settings />} />
            {/* ADD ALL CUSTOM ROUTES ABOVE THE CATCH-ALL "*" ROUTE */}
            <Route path="*" element={<NotFound />} />
          </Routes>
        </Layout>
      </BrowserRouter>
    </TooltipProvider>
  </QueryClientProvider>
);

export default App;
