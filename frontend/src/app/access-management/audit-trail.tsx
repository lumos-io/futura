import React from "react";

interface AuditTrailProps {
  title: string;
}

const AuditTrail: React.FC<AuditTrailProps> = ({ title }) => {
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

export default AuditTrail;
