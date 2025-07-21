const DevBanner: React.FC = () => {
  if (import.meta.env.VITE_ENV !== "development") {
    return null;
  }

  return (
    <div className="dev-banner">
      Environment: {import.meta.env.VITE_ENV} | Commit SHA:{" "}
      {import.meta.env.VITE_COMMIT_SHA} | Build Time:{" "}
      {import.meta.env.VITE_BUILD_TIME}
    </div>
  );
};

export default DevBanner;
