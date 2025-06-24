import React from "react";

interface KubernetesNetworkProps {
  title: string;
}

const KubernetesNetwork: React.FC<KubernetesNetworkProps> = ({ title }) => {
  return (
    <div
      style={{
        padding: "4rem",
        textAlign: "center",
        color: "#555",
      }}
    >
      <h1 style={{ fontSize: "2rem", marginBottom: "1rem" }}>{title}</h1>
      <p>{"This page is under construction."}</p>
    </div>
  );
};

export default KubernetesNetwork;
